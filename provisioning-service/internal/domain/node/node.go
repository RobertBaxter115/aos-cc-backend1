package node

import (
	"sync"
	"time"
)

// NodeStatus represents the state of a node
type NodeStatus string

const (
	NodeStatusBooting    NodeStatus = "booting"
	NodeStatusReady      NodeStatus = "ready"
	NodeStatusAllocated  NodeStatus = "allocated"
	NodeStatusTerminated NodeStatus = "terminated"
)

// Node represents a GPU node in the system
type Node struct {
	ID        string
	Status    NodeStatus
	UserID    string // Empty if not allocated
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NodePool manages the collection of nodes
type NodePool struct {
	mu    sync.RWMutex
	nodes map[string]*Node
}

// NewNodePool creates a new node pool
func NewNodePool() *NodePool {
	return &NodePool{
		nodes: make(map[string]*Node),
	}
}

// Add adds or updates a node in the pool
func (p *NodePool) Add(node *Node) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nodes[node.ID] = node
}

// Get retrieves a node by ID
func (p *NodePool) Get(nodeID string) (*Node, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	node, ok := p.nodes[nodeID]
	return node, ok
}

// Remove removes a node from the pool
func (p *NodePool) Remove(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.nodes, nodeID)
}

// GetAllByStatus returns all nodes with a specific status
func (p *NodePool) GetAllByStatus(status NodeStatus) []*Node {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*Node
	for _, node := range p.nodes {
		if node.Status == status {
			result = append(result, node)
		}
	}
	return result
}

// GetReadyNode returns a ready node and marks it as allocated
func (p *NodePool) GetReadyNode() *Node {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, node := range p.nodes {
		if node.Status == NodeStatusReady {
			return node
		}
	}
	return nil
}

// AllocateNode allocates a node to a user
func (p *NodePool) AllocateNode(nodeID, userID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	node, ok := p.nodes[nodeID]
	if !ok || node.Status != NodeStatusReady {
		return false
	}

	node.Status = NodeStatusAllocated
	node.UserID = userID
	node.UpdatedAt = time.Now()
	return true
}

// DeallocateNode deallocates a node from a user
func (p *NodePool) DeallocateNode(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if node, ok := p.nodes[nodeID]; ok {
		node.Status = NodeStatusReady
		node.UserID = ""
		node.UpdatedAt = time.Now()
	}
}

// UpdateStatus updates the status of a node
func (p *NodePool) UpdateStatus(nodeID string, status NodeStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if node, ok := p.nodes[nodeID]; ok {
		node.Status = status
		node.UpdatedAt = time.Now()
	}
}

// Count returns the total number of nodes
func (p *NodePool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.nodes)
}

// CountByStatus returns the count of nodes by status
func (p *NodePool) CountByStatus(status NodeStatus) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, node := range p.nodes {
		if node.Status == status {
			count++
		}
	}
	return count
}

// GetAll returns all nodes
func (p *NodePool) GetAll() []*Node {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*Node, 0, len(p.nodes))
	for _, node := range p.nodes {
		result = append(result, node)
	}
	return result
}
