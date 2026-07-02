package relayer

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type NATSEventBus struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewNATSEventBus(url string) (*NATSEventBus, error) {
	// Connect to NATS with reconnect options
	nc, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			fmt.Printf("NATS disconnected: %v\n", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			fmt.Println("NATS reconnected successfully")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize NATS JetStream: %w", err)
	}

	// Declare/Create standard stream if it doesn't exist
	// In production, stream configs are predefined. We auto-provision them on startup for ease of development.
	_, _ = js.AddStream(&nats.StreamConfig{
		Name:      "BRIDGE",
		Subjects:  []string{"bridge.>"},
		Retention: nats.LimitsPolicy,
		MaxAge:    365 * 24 * time.Hour, // 365-day retention as planned
		Replicas:  3,                    // 3-node JetStream cluster R=3
	})

	return &NATSEventBus{nc: nc, js: js}, nil
}

func (b *NATSEventBus) Publish(subject string, data []byte) error {
	_, err := b.js.Publish(subject, data)
	return err
}

func (b *NATSEventBus) Subscribe(subject string, handler func(msg []byte)) error {
	// We use a durable queue group subscription or push subscription
	// Durable name is derived from subject to avoid conflicts
	durableName := fmt.Sprintf("durable-%s", getSafeDurableName(subject))
	_, err := b.js.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
		_ = msg.Ack()
	}, nats.Durable(durableName), nats.ManualAck())
	return err
}

func (b *NATSEventBus) Close() {
	if b.nc != nil {
		b.nc.Close()
	}
}

func getSafeDurableName(subject string) string {
	// Convert subjects containing wildcards into filesystem-safe names
	safe := ""
	for _, c := range subject {
		if c == '.' || c == '*' || c == '>' {
			safe += "_"
		} else {
			safe += string(c)
		}
	}
	return safe
}

// PublishAsync with offset/nats_published checks for DB syncing
func (b *NATSEventBus) PublishAsync(ctx context.Context, subject string, data []byte) (<-chan error, error) {
	errChan := make(chan error, 1)
	go func() {
		_, err := b.js.Publish(subject, data)
		errChan <- err
	}()
	return errChan, nil
}
