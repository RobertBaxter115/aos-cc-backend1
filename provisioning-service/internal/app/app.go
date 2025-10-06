package app

import (
	"context"
	"time"

	"github.com/aos-cc/provisioning-service/internal/domain/allocator"
	"github.com/aos-cc/provisioning-service/internal/domain/node"
	"github.com/aos-cc/provisioning-service/internal/domain/predictor"
	"github.com/aos-cc/provisioning-service/internal/domain/user"
	"github.com/aos-cc/provisioning-service/internal/infra/config"
	"github.com/aos-cc/provisioning-service/internal/infra/http"
	"github.com/aos-cc/provisioning-service/internal/infra/nodeapi"
	"github.com/aos-cc/provisioning-service/internal/infra/redis"
	"github.com/aos-cc/provisioning-service/internal/service"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides all application dependencies
var Module = fx.Options(
	// Configuration
	fx.Provide(provideConfig),
	fx.Provide(provideLogger),

	// Domain
	fx.Provide(provideNodePool),
	fx.Provide(provideUserTracker),
	fx.Provide(provideNodeAllocator),
	fx.Provide(providePredictor),

	// Infrastructure
	fx.Provide(provideRedisClient),
	fx.Provide(provideNodeAPIClient),
	fx.Provide(provideNodeManager),
	fx.Provide(provideHTTPServer),

	// Service
	fx.Provide(provideProvisioner),
	fx.Provide(provideSubscriber),
)

func provideConfig() (*config.Config, error) {
	return config.Load("")
}

func provideLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}

func provideNodePool() *node.NodePool {
	return node.NewNodePool()
}

func provideUserTracker(cfg *config.Config) *user.UserTracker {
	return user.NewUserTracker(cfg.Prediction.ActivityWindow)
}

func provideNodeAllocator(nodePool *node.NodePool, userTracker *user.UserTracker) *allocator.NodeAllocator {
	return allocator.NewNodeAllocator(nodePool, userTracker)
}

func providePredictor(cfg *config.Config, userTracker *user.UserTracker, nodePool *node.NodePool) *predictor.Predictor {
	predConfig := predictor.PredictionConfig{
		ActivityWindow:         cfg.Prediction.ActivityWindow,
		ActivityThreshold:      cfg.Prediction.ActivityThreshold,
		PredictionWindow:       cfg.Prediction.PredictionWindow,
		MinReadyNodes:          cfg.Prediction.MinReadyNodes,
		MaxReadyNodes:          cfg.Prediction.MaxReadyNodes,
		IdleTerminationTimeout: cfg.Prediction.IdleTerminationTimeout,
		BootingNodeTimeout:     cfg.Prediction.BootingNodeTimeout,
	}
	return predictor.NewPredictor(predConfig, userTracker, nodePool)
}

func provideRedisClient(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*redis.Client, error) {
	client, err := redis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if err := client.Close(); err != nil {
				logger.Error("error closing Redis client", zap.Error(err))
				return err
			}
			logger.Info("Redis client closed")
			return nil
		},
	})

	return client, nil
}

func provideNodeAPIClient(cfg *config.Config, logger *zap.Logger) *nodeapi.Client {
	return nodeapi.NewClient(cfg.NodeAPI.BaseURL, cfg.NodeAPI.Timeout, logger)
}

func provideNodeManager(client *nodeapi.Client, logger *zap.Logger) *nodeapi.NodeManager {
	return nodeapi.NewNodeManager(client, logger)
}

func provideHTTPServer(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger, nodePool *node.NodePool, userTracker *user.UserTracker) *http.Server {
	server := http.NewServer(cfg.Server.Port, logger, nodePool, userTracker)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := server.Start(); err != nil {
					logger.Error("HTTP server error", zap.Error(err))
				}
			}()
			logger.Info("HTTP server started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error("error shutting down HTTP server", zap.Error(err))
				return err
			}
			logger.Info("HTTP server stopped")
			return nil
		},
	})

	return server
}

func provideProvisioner(
	lc fx.Lifecycle,
	nodePool *node.NodePool,
	userTracker *user.UserTracker,
	alloc *allocator.NodeAllocator,
	pred *predictor.Predictor,
	nodeManager *nodeapi.NodeManager,
	cfg *config.Config,
	logger *zap.Logger,
) *service.Provisioner {
	provisioner := service.NewProvisioner(
		nodePool,
		userTracker,
		alloc,
		pred,
		nodeManager,
		logger,
		cfg.Prediction.ScalingCheckInterval,
	)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := provisioner.Start(context.Background()); err != nil {
					logger.Error("provisioner error", zap.Error(err))
				}
			}()
			logger.Info("provisioner started")
			return nil
		},
	})

	return provisioner
}

func provideSubscriber(lc fx.Lifecycle, client *redis.Client, provisioner *service.Provisioner, logger *zap.Logger) *redis.Subscriber {
	subscriber := redis.NewSubscriber(client, provisioner, logger)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := subscriber.Start(context.Background()); err != nil {
					logger.Error("subscriber error", zap.Error(err))
				}
			}()
			logger.Info("subscriber started")
			return nil
		},
	})

	return subscriber
}

