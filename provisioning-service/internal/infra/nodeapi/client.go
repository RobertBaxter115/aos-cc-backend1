package nodeapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"resty.dev/v3"
)

// Client is an HTTP client for the Node Management API
type Client struct {
	baseURL string
	resty   *resty.Client
	logger  *zap.Logger
}

// NewClient creates a new Node API client
func NewClient(baseURL string, timeout time.Duration, logger *zap.Logger) *Client {
	restyClient := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(timeout).
		SetHeader("Content-Type", "application/json")

	return &Client{
		baseURL: baseURL,
		resty:   restyClient,
		logger:  logger,
	}
}

// CreateNode creates a new node
func (c *Client) CreateNode(ctx context.Context) (string, error) {
	var result CreateNodeResponse
	var errResp ErrorResponse

	resp, err := c.resty.R().
		SetContext(ctx).
		SetResult(&result).
		SetError(&errResp).
		Post("/api/nodes")
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != http.StatusAccepted && resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode(), errResp.Error)
	}

	c.logger.Info("node created",
		zap.String("node_id", result.ID),
	)

	return result.ID, nil
}

// DeleteNode terminates a node
func (c *Client) DeleteNode(ctx context.Context, nodeID string) error {
	var errResp ErrorResponse

	resp, err := c.resty.R().
		SetContext(ctx).
		SetError(&errResp).
		SetPathParam("nodeID", nodeID).
		Delete("/api/nodes/{nodeID}")
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != http.StatusAccepted &&
		resp.StatusCode() != http.StatusOK &&
		resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode(), errResp.Error)
	}

	c.logger.Info("node deletion requested",
		zap.String("node_id", nodeID),
	)

	return nil
}

// NodeManager handles node lifecycle operations
type NodeManager struct {
	client *Client
	logger *zap.Logger
}

// NewNodeManager creates a new node manager
func NewNodeManager(client *Client, logger *zap.Logger) *NodeManager {
	return &NodeManager{
		client: client,
		logger: logger,
	}
}

// ProvisionNode provisions a new node
func (m *NodeManager) ProvisionNode(ctx context.Context) (string, error) {
	m.logger.Info("provisioning new node")

	nodeID, err := m.client.CreateNode(ctx)
	if err != nil {
		m.logger.Error("failed to provision node", zap.Error(err))
		return "", err
	}

	m.logger.Info("node provisioned successfully",
		zap.String("node_id", nodeID),
	)

	return nodeID, nil
}

// TerminateNode terminates a node
func (m *NodeManager) TerminateNode(ctx context.Context, nodeID string) error {
	m.logger.Info("terminating node",
		zap.String("node_id", nodeID),
	)

	if err := m.client.DeleteNode(ctx, nodeID); err != nil {
		m.logger.Error("failed to terminate node",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return err
	}

	m.logger.Info("node terminated successfully",
		zap.String("node_id", nodeID),
	)

	return nil
}
