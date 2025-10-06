package allocator

import (
	"errors"

	"github.com/your-org/provisioning-service/internal/domain/node"
	"github.com/your-org/provisioning-service/internal/domain/user"
)

var (
	ErrNoReadyNode      = errors.New("no ready node available")
	ErrUserNotFound     = errors.New("user not found")
	ErrNodeNotFound     = errors.New("node not found")
	ErrNodeNotReady     = errors.New("node is not ready")
	ErrAlreadyAllocated = errors.New("user already has allocated node")
)

// NodeAllocator handles the allocation of nodes to users
type NodeAllocator struct {
	nodePool    *node.NodePool
	userTracker *user.UserTracker
}

// NewNodeAllocator creates a new node allocator
func NewNodeAllocator(nodePool *node.NodePool, userTracker *user.UserTracker) *NodeAllocator {
	return &NodeAllocator{
		nodePool:    nodePool,
		userTracker: userTracker,
	}
}

// AllocateNodeToUser allocates a ready node to a user
func (a *NodeAllocator) AllocateNodeToUser(userID string) (string, error) {
	// Check if user already has a node
	state, exists := a.userTracker.GetUserState(userID)
	if exists && state.IsConnected && state.AllocatedNodeID != "" {
		return state.AllocatedNodeID, ErrAlreadyAllocated
	}

	// Get a ready node
	node := a.nodePool.GetReadyNode()
	if node == nil {
		return "", ErrNoReadyNode
	}

	// Allocate the node
	success := a.nodePool.AllocateNode(node.ID, userID)
	if !success {
		return "", ErrNodeNotReady
	}

	// Mark user as connected
	a.userTracker.MarkConnected(userID, node.ID)

	return node.ID, nil
}

// DeallocateNodeFromUser deallocates a node from a user
func (a *NodeAllocator) DeallocateNodeFromUser(userID string) error {
	// Get user state
	state, exists := a.userTracker.GetUserState(userID)
	if !exists || !state.IsConnected {
		return ErrUserNotFound
	}

	nodeID := state.AllocatedNodeID
	if nodeID == "" {
		return ErrNodeNotFound
	}

	// Deallocate the node
	a.nodePool.DeallocateNode(nodeID)

	// Mark user as disconnected
	a.userTracker.MarkDisconnected(userID)

	return nil
}

// GetAllocation returns the current allocation for a user
func (a *NodeAllocator) GetAllocation(userID string) (string, bool) {
	state, exists := a.userTracker.GetUserState(userID)
	if !exists || !state.IsConnected {
		return "", false
	}
	return state.AllocatedNodeID, true
}

// GetNodeAllocation returns the user allocated to a node
func (a *NodeAllocator) GetNodeAllocation(nodeID string) (string, bool) {
	n, exists := a.nodePool.Get(nodeID)
	if !exists || n.Status != node.NodeStatusAllocated {
		return "", false
	}
	return n.UserID, true
}
