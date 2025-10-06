package predictor

import (
	"time"

	"github.com/your-org/provisioning-service/internal/domain/node"
	"github.com/your-org/provisioning-service/internal/domain/user"
)

// PredictionConfig holds configuration for the predictive algorithm
type PredictionConfig struct {
	// ActivityWindow is the time window to consider for user activity
	ActivityWindow time.Duration

	// ActivityThreshold is the number of activities within the window
	// that suggests a user is likely to connect
	ActivityThreshold int

	// PredictionWindow is how far ahead we predict connections
	PredictionWindow time.Duration

	// MinReadyNodes is the minimum number of ready nodes to maintain
	MinReadyNodes int

	// MaxReadyNodes is the maximum number of ready nodes to maintain
	MaxReadyNodes int

	// IdleTerminationTimeout is how long a ready node can be idle before termination
	IdleTerminationTimeout time.Duration

	// BootingNodeTimeout is the timeout for booting nodes
	BootingNodeTimeout time.Duration
}

// DefaultPredictionConfig returns default prediction configuration
func DefaultPredictionConfig() PredictionConfig {
	return PredictionConfig{
		ActivityWindow:         2 * time.Minute,
		ActivityThreshold:      3,
		PredictionWindow:       1 * time.Minute,
		MinReadyNodes:          1,
		MaxReadyNodes:          5,
		IdleTerminationTimeout: 5 * time.Minute,
		BootingNodeTimeout:     2 * time.Minute,
	}
}

// Predictor implements the predictive scaling algorithm
type Predictor struct {
	config      PredictionConfig
	userTracker *user.UserTracker
	nodePool    *node.NodePool
}

// NewPredictor creates a new predictor
func NewPredictor(config PredictionConfig, userTracker *user.UserTracker, nodePool *node.NodePool) *Predictor {
	return &Predictor{
		config:      config,
		userTracker: userTracker,
		nodePool:    nodePool,
	}
}

// ScalingDecision represents a decision to scale nodes
type ScalingDecision struct {
	ShouldScaleUp   bool
	ShouldScaleDown bool
	TargetNodes     int
	Reason          string
}

// CalculateScaling determines if we need to scale up or down
func (p *Predictor) CalculateScaling() ScalingDecision {
	// Get current node counts
	readyCount := p.nodePool.CountByStatus(node.NodeStatusReady)
	bootingCount := p.nodePool.CountByStatus(node.NodeStatusBooting)
	allocatedCount := p.nodePool.CountByStatus(node.NodeStatusAllocated)

	// Get likely-to-connect users
	likelyUsers := p.userTracker.GetLikelyToConnect(
		p.config.ActivityThreshold,
		p.config.ActivityWindow,
	)

	// Calculate demand: number of users likely to connect
	demand := len(likelyUsers)

	// Calculate available capacity (ready + booting nodes)
	availableCapacity := readyCount + bootingCount

	// Decision logic
	decision := ScalingDecision{}

	// Scale up if:
	// 1. Demand exceeds available capacity
	// 2. Ready nodes are below minimum threshold
	if demand > availableCapacity {
		decision.ShouldScaleUp = true
		decision.TargetNodes = demand - availableCapacity
		decision.Reason = "demand exceeds capacity"
	} else if readyCount < p.config.MinReadyNodes && (readyCount+bootingCount) < p.config.MinReadyNodes {
		decision.ShouldScaleUp = true
		decision.TargetNodes = p.config.MinReadyNodes - (readyCount + bootingCount)
		decision.Reason = "maintaining minimum ready nodes"
	}

	// Cap scale-up to max ready nodes
	if decision.ShouldScaleUp {
		totalNodes := readyCount + bootingCount + allocatedCount + decision.TargetNodes
		if totalNodes > p.config.MaxReadyNodes {
			decision.TargetNodes = p.config.MaxReadyNodes - (readyCount + bootingCount + allocatedCount)
			if decision.TargetNodes <= 0 {
				decision.ShouldScaleUp = false
			}
		}
	}

	// Scale down if:
	// 1. Ready nodes exceed max threshold
	// 2. Too many ready nodes for current demand
	excessNodes := readyCount - p.config.MinReadyNodes
	if excessNodes > 0 && demand == 0 {
		decision.ShouldScaleDown = true
		decision.TargetNodes = excessNodes
		decision.Reason = "excess capacity with no demand"
	}

	return decision
}

// GetIdleNodes returns nodes that have been idle for too long
func (p *Predictor) GetIdleNodes() []*node.Node {
	readyNodes := p.nodePool.GetAllByStatus(node.NodeStatusReady)
	cutoff := time.Now().Add(-p.config.IdleTerminationTimeout)

	var idleNodes []*node.Node
	for _, n := range readyNodes {
		if n.UpdatedAt.Before(cutoff) {
			idleNodes = append(idleNodes, n)
		}
	}

	// Ensure we don't terminate below minimum
	readyCount := len(readyNodes)
	maxTerminations := readyCount - p.config.MinReadyNodes
	if maxTerminations < 0 {
		maxTerminations = 0
	}

	if len(idleNodes) > maxTerminations {
		idleNodes = idleNodes[:maxTerminations]
	}

	return idleNodes
}

// GetStuckBootingNodes returns nodes that have been booting for too long
func (p *Predictor) GetStuckBootingNodes() []*node.Node {
	bootingNodes := p.nodePool.GetAllByStatus(node.NodeStatusBooting)
	cutoff := time.Now().Add(-p.config.BootingNodeTimeout)

	var stuckNodes []*node.Node
	for _, n := range bootingNodes {
		if n.CreatedAt.Before(cutoff) {
			stuckNodes = append(stuckNodes, n)
		}
	}

	return stuckNodes
}

// ShouldAllocateNode determines if a ready node should be allocated to a user
func (p *Predictor) ShouldAllocateNode(userID string) bool {
	// Check if user has recent activity
	state, exists := p.userTracker.GetUserState(userID)
	if !exists {
		return true // Allow allocation if no state (first time user)
	}

	// Check if already connected
	if state.IsConnected {
		return false
	}

	return true
}
