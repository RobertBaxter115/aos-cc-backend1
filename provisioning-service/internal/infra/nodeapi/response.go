package nodeapi

// CreateNodeResponse represents the response from creating a node
type CreateNodeResponse struct {
	ID string `json:"id"`
}

// DeleteNodeResponse represents the response from deleting a node
type DeleteNodeResponse struct {
	Message string `json:"message,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}
