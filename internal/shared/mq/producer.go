package mq

import (
	"context"
	"fmt"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"mycourse-io-be/internal/shared/setting"
)

// PublishTopic publishes a message to the configured topic exchange with routingKey.
func PublishTopic(ctx context.Context, routingKey string, msg Publishing) error {
	if !Available() {
		return fmt.Errorf("lavinmq: connection not available")
	}
	routingKey = strings.TrimSpace(routingKey)
	if routingKey == "" {
		return fmt.Errorf("lavinmq: routing key is required")
	}

	ch, err := connection().Channel()
	if err != nil {
		return fmt.Errorf("lavinmq: open channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	contentType := strings.TrimSpace(msg.ContentType)
	if contentType == "" {
		contentType = "application/json"
	}
	ts := msg.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	pub := amqp.Publishing{
		ContentType:  contentType,
		DeliveryMode: amqp.Persistent,
		MessageId:    msg.MessageID,
		Timestamp:    ts,
		Body:         msg.Body,
		Headers:      msg.Headers,
	}
	if pub.Headers == nil {
		pub.Headers = amqp.Table{}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return ch.PublishWithContext(
		ctx,
		setting.LavinMQSetting.Exchange,
		routingKey,
		false,
		false,
		pub,
	)
}
