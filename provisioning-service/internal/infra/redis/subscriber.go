package redis

import (
	"context"
	"encoding/json"

	"github.com/aos-cc/provisioning-service/internal/domain/events"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// EventHandler handles different types of events
type EventHandler interface {
	HandleUserActivity(ctx context.Context, event events.UserActivityEvent) error
	HandleUserConnect(ctx context.Context, event events.UserConnectEvent) error
	HandleUserDisconnect(ctx context.Context, event events.UserDisconnectEvent) error
	HandleNodeStatus(ctx context.Context, event events.NodeStatusEvent) error
}

// Subscriber listens to Redis pub/sub channels
type Subscriber struct {
	client  *Client
	handler EventHandler
	logger  *zap.Logger
}

// NewSubscriber creates a new Redis subscriber
func NewSubscriber(client *Client, handler EventHandler, logger *zap.Logger) *Subscriber {
	return &Subscriber{
		client:  client,
		handler: handler,
		logger:  logger,
	}
}

// Start starts listening to all channels
func (s *Subscriber) Start(ctx context.Context) error {
	channels := []string{
		events.ChannelUserActivity,
		events.ChannelUserConnect,
		events.ChannelUserDisconnect,
		events.ChannelNodeStatus,
	}

	pubsub := s.client.GetClient().Subscribe(ctx, channels...)
	defer pubsub.Close()

	// Wait for confirmation that subscription is created
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return err
	}

	s.logger.Info("subscribed to channels", zap.Strings("channels", channels))

	// Listen for messages
	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("subscriber stopping")
			return ctx.Err()
		case msg := <-ch:
			if msg == nil {
				continue
			}
			s.handleMessage(ctx, msg)
		}
	}
}

func (s *Subscriber) handleMessage(ctx context.Context, msg *redis.Message) {
	s.logger.Debug("received message",
		zap.String("channel", msg.Channel),
		zap.String("payload", msg.Payload),
	)

	var err error

	switch msg.Channel {
	case events.ChannelUserActivity:
		var event events.UserActivityEvent
		if err = json.Unmarshal([]byte(msg.Payload), &event); err == nil {
			err = s.handler.HandleUserActivity(ctx, event)
		}

	case events.ChannelUserConnect:
		var event events.UserConnectEvent
		if err = json.Unmarshal([]byte(msg.Payload), &event); err == nil {
			err = s.handler.HandleUserConnect(ctx, event)
		}

	case events.ChannelUserDisconnect:
		var event events.UserDisconnectEvent
		if err = json.Unmarshal([]byte(msg.Payload), &event); err == nil {
			err = s.handler.HandleUserDisconnect(ctx, event)
		}

	case events.ChannelNodeStatus:
		var event events.NodeStatusEvent
		if err = json.Unmarshal([]byte(msg.Payload), &event); err == nil {
			err = s.handler.HandleNodeStatus(ctx, event)
		}

	default:
		s.logger.Warn("unknown channel", zap.String("channel", msg.Channel))
		return
	}

	if err != nil {
		s.logger.Error("failed to handle message",
			zap.String("channel", msg.Channel),
			zap.Error(err),
		)
	}
}
