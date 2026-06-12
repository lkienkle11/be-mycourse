package mq

import (
	"context"
	"fmt"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"mycourse-io-be/internal/shared/constants"
	"mycourse-io-be/internal/shared/setting"
)

// Handler processes one consumed topic message. Return nil to ack, non-nil to nack (no requeue).
type Handler func(ctx context.Context, d *Delivery) error

// Subscription binds a durable queue to a topic routing key on the configured exchange.
type Subscription struct {
	QueueName   string
	RoutingKey  string
	Handler     Handler
	ConsumerTag string
}

// StartConsumers launches one goroutine per subscription on the shared connection.
// Each consumer uses its own channel (CloudAMQP / LavinMQ recommendation).
func StartConsumers(ctx context.Context, subs ...Subscription) {
	if !Available() {
		zap.L().Info("lavinmq consumers skipped", zap.String("reason", "connection not available"))
		return
	}
	for _, sub := range subs {
		sub := sub
		if sub.Handler == nil {
			continue
		}
		if strings.TrimSpace(sub.RoutingKey) == "" {
			zap.L().Warn("lavinmq consumer skipped", zap.String("reason", "empty routing key"))
			continue
		}
		go runConsumer(ctx, sub)
	}
}

func runConsumer(ctx context.Context, sub Subscription) {
	queueName, consumerTag := resolveSubscriptionNames(sub)
	ch, err := openConsumerChannel(queueName)
	if err != nil {
		return
	}
	defer func() { _ = ch.Close() }()

	deliveries, err := bindAndConsume(ch, queueName, consumerTag, sub.RoutingKey)
	if err != nil {
		return
	}

	zap.L().Info("lavinmq consumer started",
		zap.String("queue", queueName),
		zap.String("routing_key", sub.RoutingKey),
		zap.String("exchange", setting.LavinMQSetting.Exchange),
	)
	consumeLoop(ctx, queueName, sub.Handler, deliveries)
}

func resolveSubscriptionNames(sub Subscription) (queueName, consumerTag string) {
	queueName = strings.TrimSpace(sub.QueueName)
	if queueName == "" {
		queueName = defaultQueueName(sub.RoutingKey)
	}
	consumerTag = strings.TrimSpace(sub.ConsumerTag)
	if consumerTag == "" {
		consumerTag = queueName
	}
	return queueName, consumerTag
}

func openConsumerChannel(queueName string) (*amqp.Channel, error) {
	ch, err := connection().Channel()
	if err != nil {
		zap.L().Error("lavinmq consumer channel failed",
			zap.String("queue", queueName),
			zap.Error(err),
		)
		return nil, err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		zap.L().Error("lavinmq consumer qos failed",
			zap.String("queue", queueName),
			zap.Error(err),
		)
		return nil, err
	}
	return ch, nil
}

func bindAndConsume(ch *amqp.Channel, queueName, consumerTag, routingKey string) (<-chan amqp.Delivery, error) {
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		zap.L().Error("lavinmq queue declare failed", zap.String("queue", queueName), zap.Error(err))
		return nil, err
	}
	if err := ch.QueueBind(queueName, routingKey, setting.LavinMQSetting.Exchange, false, nil); err != nil {
		zap.L().Error("lavinmq queue bind failed",
			zap.String("queue", queueName),
			zap.String("routing_key", routingKey),
			zap.Error(err),
		)
		return nil, err
	}
	deliveries, err := ch.Consume(queueName, consumerTag, false, false, false, false, nil)
	if err != nil {
		zap.L().Error("lavinmq consume failed", zap.String("queue", queueName), zap.Error(err))
		return nil, err
	}
	return deliveries, nil
}

func consumeLoop(ctx context.Context, queueName string, handler Handler, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("lavinmq consumer stopped", zap.String("queue", queueName))
			return
		case d, ok := <-deliveries:
			if !ok {
				zap.L().Warn("lavinmq delivery channel closed", zap.String("queue", queueName))
				return
			}
			handleDelivery(ctx, queueName, handler, d)
		}
	}
}

func handleDelivery(ctx context.Context, queueName string, handler Handler, d amqp.Delivery) {
	wrapped := &Delivery{delivery: d}
	if err := handler(ctx, wrapped); err != nil {
		zap.L().Warn("lavinmq handler error",
			zap.String("queue", queueName),
			zap.String("routing_key", d.RoutingKey),
			zap.Error(err),
		)
		_ = wrapped.Nack(false)
		return
	}
	if err := wrapped.Ack(); err != nil {
		zap.L().Warn("lavinmq ack failed", zap.String("queue", queueName), zap.Error(err))
	}
}

func defaultQueueName(routingKey string) string {
	safe := strings.ReplaceAll(routingKey, ".", "_")
	return fmt.Sprintf("%s.%s", constants.LavinMQQueuePrefix, safe)
}
