package mq

import (
	"context"

	"go.uber.org/zap"

	"mycourse-io-be/internal/shared/constants"
)

// StartDefaultConsumers registers built-in topic listeners for the API process.
func StartDefaultConsumers(ctx context.Context) {
	StartConsumers(ctx,
		Subscription{
			RoutingKey: constants.TopicHealthPing,
			Handler:    logHealthPingHandler,
		},
	)
}

func logHealthPingHandler(ctx context.Context, d *Delivery) error {
	_ = ctx
	zap.L().Debug("lavinmq topic received",
		zap.String("routing_key", d.RoutingKey()),
		zap.String("message_id", d.MessageID()),
		zap.Int("body_bytes", len(d.Body())),
	)
	return nil
}
