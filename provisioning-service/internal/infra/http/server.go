package http

import (
	"context"
	"fmt"
	"time"

	"github.com/aos-cc/provisioning-service/internal/domain/node"
	"github.com/aos-cc/provisioning-service/internal/domain/user"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

// Server is the HTTP server for health checks and metrics
type Server struct {
	app        *fiber.App
	port       int
	logger     *zap.Logger
	nodePool   *node.NodePool
	userTracker *user.UserTracker
}

// NewServer creates a new HTTP server
func NewServer(port int, logger *zap.Logger, nodePool *node.NodePool, userTracker *user.UserTracker) *Server {
	app := fiber.New()

	s := &Server{
		app:         app,
		port:        port,
		logger:      logger,
		nodePool:    nodePool,
		userTracker: userTracker,
	}

	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	s.app.Get("/health", s.healthHandler)
	s.app.Get("/metrics", s.metricsHandler)
	s.app.Get("/status", s.statusHandler)
}

func (s *Server) healthHandler(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
		"time":   time.Now().Unix(),
	})
}

func (s *Server) metricsHandler(c fiber.Ctx) error {
	metrics := fiber.Map{
		"nodes": fiber.Map{
			"total":      s.nodePool.Count(),
			"booting":    s.nodePool.CountByStatus(node.NodeStatusBooting),
			"ready":      s.nodePool.CountByStatus(node.NodeStatusReady),
			"allocated":  s.nodePool.CountByStatus(node.NodeStatusAllocated),
			"terminated": s.nodePool.CountByStatus(node.NodeStatusTerminated),
		},
		"users": fiber.Map{
			"connected": len(s.userTracker.GetConnectedUsers()),
		},
		"timestamp": time.Now().Unix(),
	}

	return c.JSON(metrics)
}

func (s *Server) statusHandler(c fiber.Ctx) error {
	nodes := s.nodePool.GetAll()
	connectedUsers := s.userTracker.GetConnectedUsers()

	nodeDetails := make([]fiber.Map, 0, len(nodes))
	for _, node := range nodes {
		nodeDetails = append(nodeDetails, fiber.Map{
			"id":         node.ID,
			"status":     node.Status,
			"user_id":    node.UserID,
			"created_at": node.CreatedAt.Unix(),
			"updated_at": node.UpdatedAt.Unix(),
		})
	}

	userDetails := make([]fiber.Map, 0, len(connectedUsers))
	for _, user := range connectedUsers {
		userDetails = append(userDetails, fiber.Map{
			"user_id":           user.UserID,
			"allocated_node_id": user.AllocatedNodeID,
			"last_activity":     user.LastActivityTime.Unix(),
			"activity_count":    user.ActivityCount,
		})
	}

	return c.JSON(fiber.Map{
		"nodes":     nodeDetails,
		"users":     userDetails,
		"timestamp": time.Now().Unix(),
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Info("starting HTTP server", zap.String("addr", addr))
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")
	return s.app.ShutdownWithContext(ctx)
}
