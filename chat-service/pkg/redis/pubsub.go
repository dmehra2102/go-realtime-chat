package redispkg

import (
	"context"

	"github.com/dmehra2102/go-realtime-chat/shared/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type RedisPubSub struct {
	client *redis.Client
	logger *logger.Logger
}

func NewRedisPubSub(client *redis.Client, logger *logger.Logger) *RedisPubSub {
	return &RedisPubSub{
		client: client,
		logger: logger,
	}
}

func (r *RedisPubSub) Publish(ctx context.Context, channel, message string) error {
	return r.client.Publish(ctx, channel, message).Err()
}

func (r *RedisPubSub) Subscribe(ctx context.Context, channel string) <-chan string {
	pubSub := r.client.Subscribe(ctx, channel)
	msgChan := make(chan string, 100)

	if _,err := pubSub.Receive(ctx); err != nil {
		r.logger.Error("failed to subscribe to channel", "channel", channel, "error", err)
		pubSub.Close()
		closedChan := make(chan string)
		close(closedChan)
		return closedChan
	}

	go func() {
		defer close(msgChan)
		defer pubSub.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := pubSub.ReceiveMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					r.logger.Error("Redis subscription error", "channel", channel, "error", err)
					continue
				}

				select {
				case msgChan <- msg.Payload:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return msgChan
}
