package push

import (
	"context"
	"fmt"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Notification is the push payload delivered to a set of device tokens.
type Notification struct {
	Title string
	Body  string
	Data  map[string]string
}

// Sender delivers push notifications to device tokens.
type Sender interface {
	Send(ctx context.Context, tokens []string, n Notification) error
}

// fcmSender delivers through Firebase Cloud Messaging.
type fcmSender struct {
	client *messaging.Client
	log    *slog.Logger
}

// noopSender logs the push instead of delivering it. Used when no FCM
// credentials are configured so local dev works without Firebase.
type noopSender struct {
	log *slog.Logger
}

// New returns an FCM-backed Sender when credsFile is set, otherwise a
// logging/no-op Sender so the service runs without Firebase credentials.
func New(ctx context.Context, credsFile string, log *slog.Logger) (Sender, error) {
	if credsFile == "" {
		log.Info("FCM credentials not configured; push runs in logging/no-op mode")
		return &noopSender{log: log}, nil
	}

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credsFile))
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("init messaging client: %w", err)
	}
	log.Info("FCM push enabled", "creds_file", credsFile)
	return &fcmSender{client: client, log: log}, nil
}

func (s *fcmSender) Send(ctx context.Context, tokens []string, n Notification) error {
	if len(tokens) == 0 {
		return nil
	}
	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: n.Title,
			Body:  n.Body,
		},
		Data: n.Data,
	}
	resp, err := s.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		return fmt.Errorf("send multicast: %w", err)
	}
	if resp.FailureCount > 0 {
		s.log.Warn("some pushes failed",
			"success", resp.SuccessCount, "failure", resp.FailureCount, "total", len(tokens))
	}
	return nil
}

func (s *noopSender) Send(_ context.Context, tokens []string, n Notification) error {
	s.log.Info("noop push",
		"tokens", len(tokens), "title", n.Title, "body", n.Body, "data", n.Data)
	return nil
}
