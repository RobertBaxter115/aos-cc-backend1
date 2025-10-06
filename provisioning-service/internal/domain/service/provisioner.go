package service

import (
	"context"
	"time"

	"github.com/aos-cc/provisioning-service/internal/domain/allocator"
	"github.com/aos-cc/provisioning-service/internal/domain/events"
	"github.com/aos-cc/provisioning-service/internal/domain/node"
	"github.com/aos-cc/provisioning-service/internal/domain/predictor"
	"github.com/aos-cc/provisioning-service/internal/domain/user"
	"github.com/aos-cc/provisioning-service/internal/infra/nodeapi"
	"go.uber.org/zap"
)

// Provisioner is the core service that orchestrates node provisioning
type Provisioner struct {
	nodePool      *node.NodePool
	userTracker   *user.UserTracker
	allocator     *allocator.NodeAllocator
	predictor     *predictor.Predictor
	nodeManager   *nodeapi.NodeManager
	logger        *zap.Logger
	checkInterval time.Duration
}

// NewProvisioner creates a new provisioner service
func NewProvisioner(
	nodePool *node.NodePool,
	userTracker *user.UserTracker,
	alloc *allocator.NodeAllocator,
	pred *predictor.Predictor,
	nodeManager *nodeapi.NodeManager,
	logger *zap.Logger,
	checkInterval time.Duration,
) *Provisioner {
	return &Provisioner{
		nodePool:      nodePool,
		userTracker:   userTracker,
		allocator:     alloc,
		predictor:     pred,
		nodeManager:   nodeManager,
		logger:        logger,
		checkInterval: checkInterval,
	}
}

// Start starts the provisioner service
func (p *Provisioner) Start(ctx context.Context) error {
	p.logger.Info("provisioner service started")

	ticker := time.NewTicker(p.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("provisioner service stopping")
			return ctx.Err()
		case <-ticker.C:
			p.performScalingCheck(ctx)
			p.cleanupIdleNodes(ctx)
			p.cleanupStuckNodes(ctx)
		}
	}
}

func (p *Provisioner) performScalingCheck(ctx context.Context) {
	decision := p.predictor.CalculateScaling()

	if decision.ShouldScaleUp {
		p.logger.Info("scaling up nodes",
			zap.Int("target_nodes", decision.TargetNodes),
			zap.String("reason", decision.Reason),
		)

		for i := 0; i < decision.TargetNodes; i++ {
			if err := p.provisionNode(ctx); err != nil {
				p.logger.Error("failed to provision node", zap.Error(err))
			}
		}
	}

	if decision.ShouldScaleDown {
		p.logger.Info("scaling down consideration",
			zap.Int("target_nodes", decision.TargetNodes),
			zap.String("reason", decision.Reason),
		)
		// Scale down is handled by idle cleanup
	}
}

func (p *Provisioner) provisionNode(ctx context.Context) error {
	nodeID, err := p.nodeManager.ProvisionNode(ctx)
	if err != nil {
		return err
	}

	// Add node to pool with booting status
	n := &node.Node{
		ID:        nodeID,
		Status:    node.NodeStatusBooting,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	p.nodePool.Add(n)

	p.logger.Info("node added to pool",
		zap.String("node_id", nodeID),
		zap.String("status", string(node.NodeStatusBooting)),
	)

	return nil
}

func (p *Provisioner) cleanupIdleNodes(ctx context.Context) {
	idleNodes := p.predictor.GetIdleNodes()

	for _, n := range idleNodes {
		p.logger.Info("terminating idle node",
			zap.String("node_id", n.ID),
			zap.Duration("idle_duration", time.Since(n.UpdatedAt)),
		)

		if err := p.nodeManager.TerminateNode(ctx, n.ID); err != nil {
			p.logger.Error("failed to terminate idle node",
				zap.String("node_id", n.ID),
				zap.Error(err),
			)
			continue
		}

		// Update status to terminated
		p.nodePool.UpdateStatus(n.ID, node.NodeStatusTerminated)
	}
}

func (p *Provisioner) cleanupStuckNodes(ctx context.Context) {
	stuckNodes := p.predictor.GetStuckBootingNodes()

	for _, n := range stuckNodes {
		p.logger.Warn("terminating stuck booting node",
			zap.String("node_id", n.ID),
			zap.Duration("booting_duration", time.Since(n.CreatedAt)),
		)

		if err := p.nodeManager.TerminateNode(ctx, n.ID); err != nil {
			p.logger.Error("failed to terminate stuck node",
				zap.String("node_id", n.ID),
				zap.Error(err),
			)
			continue
		}

		// Remove from pool
		p.nodePool.Remove(n.ID)
	}
}

// HandleUserActivity handles user activity events
func (p *Provisioner) HandleUserActivity(ctx context.Context, event events.UserActivityEvent) error {
	timestamp := time.Unix(event.Timestamp, 0)
	p.userTracker.RecordActivity(event.UserID, timestamp)

	p.logger.Debug("user activity recorded",
		zap.String("user_id", event.UserID),
		zap.Time("timestamp", timestamp),
	)

	return nil
}

// HandleUserConnect handles user connect events
func (p *Provisioner) HandleUserConnect(ctx context.Context, event events.UserConnectEvent) error {
	p.logger.Info("user connect request",
		zap.String("user_id", event.UserID),
	)

	nodeID, err := p.allocator.AllocateNodeToUser(event.UserID)
	if err != nil {
		switch err {
		case allocator.ErrNoReadyNode:
			p.logger.Error("CRITICAL: no ready node available for user",
				zap.String("user_id", event.UserID),
			)
			// Emergency provision
			if provErr := p.provisionNode(ctx); provErr != nil {
				p.logger.Error("failed to emergency provision node", zap.Error(provErr))
			}
		case allocator.ErrAlreadyAllocated:
			p.logger.Info("user already has allocated node",
				zap.String("user_id", event.UserID),
				zap.String("node_id", nodeID),
			)
			return nil
		default:
			p.logger.Error("failed to allocate node",
				zap.String("user_id", event.UserID),
				zap.Error(err),
			)
		}
		return err
	}

	p.logger.Info("node allocated to user",
		zap.String("user_id", event.UserID),
		zap.String("node_id", nodeID),
	)

	return nil
}

// HandleUserDisconnect handles user disconnect events
func (p *Provisioner) HandleUserDisconnect(ctx context.Context, event events.UserDisconnectEvent) error {
	p.logger.Info("user disconnect",
		zap.String("user_id", event.UserID),
	)

	if err := p.allocator.DeallocateNodeFromUser(event.UserID); err != nil {
		p.logger.Error("failed to deallocate node",
			zap.String("user_id", event.UserID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// HandleNodeStatus handles node status events
func (p *Provisioner) HandleNodeStatus(ctx context.Context, event events.NodeStatusEvent) error {
	p.logger.Info("node status update",
		zap.String("node_id", event.NodeID),
		zap.String("status", event.Status),
	)

	if _, exists := p.nodePool.Get(event.NodeID); !exists {
		n := &node.Node{
			ID:        event.NodeID,
			Status:    node.NodeStatus(event.Status),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		p.nodePool.Add(n)
	} else {
		p.nodePool.UpdateStatus(event.NodeID, node.NodeStatus(event.Status))
	}

	return nil
}
