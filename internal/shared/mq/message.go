package mq

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publishing is the outbound message shape for topic publish.
type Publishing struct {
	Body        []byte
	ContentType string
	MessageID   string
	Headers     amqp.Table
	Timestamp   time.Time
}

// Delivery wraps a consumed AMQP delivery with ack helpers.
type Delivery struct {
	delivery amqp.Delivery
}

func (d *Delivery) Body() []byte {
	return d.delivery.Body
}

func (d *Delivery) RoutingKey() string {
	return d.delivery.RoutingKey
}

func (d *Delivery) MessageID() string {
	return d.delivery.MessageId
}

func (d *Delivery) ContentType() string {
	return d.delivery.ContentType
}

func (d *Delivery) Timestamp() time.Time {
	return d.delivery.Timestamp
}

func (d *Delivery) Ack() error {
	return d.delivery.Ack(false)
}

func (d *Delivery) Nack(requeue bool) error {
	return d.delivery.Nack(false, requeue)
}
