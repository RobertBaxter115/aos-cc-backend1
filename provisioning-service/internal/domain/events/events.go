package events

// Event types for Redis pub/sub
const (
	ChannelUserActivity   = "user:activity"
	ChannelUserConnect    = "user:connect"
	ChannelUserDisconnect = "user:disconnect"
	ChannelNodeStatus     = "node:status"
)

// UserActivityEvent represents a user activity message
type UserActivityEvent struct {
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

// UserConnectEvent represents a user connect message
type UserConnectEvent struct {
	UserID string `json:"user_id"`
}

// UserDisconnectEvent represents a user disconnect message
type UserDisconnectEvent struct {
	UserID string `json:"user_id"`
}

// NodeStatusEvent represents a node status change message
type NodeStatusEvent struct {
	NodeID string `json:"node_id"`
	Status string `json:"status"` // booting|ready|terminated
}
