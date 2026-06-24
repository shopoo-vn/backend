package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/marketplace/noti-service/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueName = "noti-service.events"

	routeChatMessage    = "chat.message.created"
	routeListingApprove = "listing.approved"
	routeListingReject  = "listing.rejected"
)

// envelope is the project-standard event envelope. data is decoded per-type.
type envelope struct {
	EventID    string          `json:"eventId"`
	Type       string          `json:"type"`
	OccurredAt string          `json:"occurredAt"`
	Data       json.RawMessage `json:"data"`
}

// Consumer connects to RabbitMQ, binds a durable queue to the events this
// service cares about, and dispatches each message to the NotiService. It acks
// on success and nacks (no requeue → DLQ) on handler failure.
type Consumer struct {
	url      string
	exchange string
	svc      *service.NotiService
	log      *slog.Logger

	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewConsumer(url, exchange string, svc *service.NotiService, log *slog.Logger) *Consumer {
	return &Consumer{url: url, exchange: exchange, svc: svc, log: log}
}

// Start opens the connection/channel, declares topology and begins consuming.
// Delivered messages are handled in a goroutine that exits when the deliveries
// channel closes (on Close or connection loss).
func (c *Consumer) Start(ctx context.Context) error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(c.exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("declare exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("declare queue: %w", err)
	}
	for _, key := range []string{routeChatMessage, routeListingApprove, routeListingReject} {
		if err := ch.QueueBind(queueName, key, c.exchange, false, nil); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return fmt.Errorf("bind %q: %w", key, err)
		}
	}
	// Process one message at a time per consumer for predictable redelivery.
	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("set qos: %w", err)
	}

	deliveries, err := ch.Consume(queueName, "", false /* manual ack */, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("consume: %w", err)
	}

	c.conn = conn
	c.ch = ch

	go c.loop(ctx, deliveries)
	c.log.Info("consuming events", "queue", queueName,
		"keys", []string{routeChatMessage, routeListingApprove, routeListingReject})
	return nil
}

func (c *Consumer) loop(ctx context.Context, deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		c.handle(ctx, d)
	}
	c.log.Info("delivery channel closed; consumer loop exiting")
}

func (c *Consumer) handle(ctx context.Context, d amqp.Delivery) {
	var env envelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		c.log.Error("malformed envelope; dropping", "err", err)
		_ = d.Nack(false, false) // unparseable → DLQ, never requeue
		return
	}

	if err := c.dispatch(ctx, env); err != nil {
		c.log.Error("event handling failed; nacking",
			"event_id", env.EventID, "type", env.Type, "err", err)
		_ = d.Nack(false, false) // no requeue → DLQ
		return
	}
	if err := d.Ack(false); err != nil {
		c.log.Error("ack failed", "event_id", env.EventID, "type", env.Type, "err", err)
	}
}

func (c *Consumer) dispatch(ctx context.Context, env envelope) error {
	switch env.Type {
	case routeChatMessage:
		var data service.ChatMessageData
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return fmt.Errorf("decode chat data: %w", err)
		}
		return c.svc.HandleChatMessage(ctx, env.EventID, data)

	case routeListingApprove, routeListingReject:
		var data service.ListingModerationData
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return fmt.Errorf("decode listing data: %w", err)
		}
		return c.svc.HandleListingModeration(ctx, env.EventID, env.Type, data)

	default:
		// Bound only to keys we handle, so this is unexpected; log and treat as
		// handled so we don't endlessly DLQ.
		c.log.Warn("unhandled event type", "type", env.Type, "event_id", env.EventID)
		return nil
	}
}

// Close tears down the channel and connection.
func (c *Consumer) Close() {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
