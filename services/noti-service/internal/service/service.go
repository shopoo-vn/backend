package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/marketplace/noti-service/internal/model"
	"github.com/marketplace/noti-service/internal/push"
	"github.com/marketplace/noti-service/internal/repo"
	"github.com/redis/go-redis/v9"
)

// ErrInvalidInput is returned for malformed REST requests.
var ErrInvalidInput = errors.New("invalid input")

// Re-exported so handlers can map it without importing repo directly.
var ErrNotFound = repo.ErrNotFound

// NotiService owns device-token management and turns consumed events into
// persisted notifications + FCM pushes. Idempotency is enforced in Redis so a
// redelivered event is processed at most once.
type NotiService struct {
	repo         *repo.Repo
	redis        *redis.Client
	push         push.Sender
	log          *slog.Logger
	processedTTL time.Duration
}

func NewNotiService(r *repo.Repo, rdb *redis.Client, p push.Sender, log *slog.Logger, processedTTL time.Duration) *NotiService {
	return &NotiService{repo: r, redis: rdb, push: p, log: log, processedTTL: processedTTL}
}

// ── device tokens (REST) ─────────────────────────────────────────────────────

// RegisterDevice upserts a device token for the authenticated user.
func (s *NotiService) RegisterDevice(ctx context.Context, userID, tokenStr, platform string) (*model.DeviceToken, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return nil, fmt.Errorf("%w: token is required", ErrInvalidInput)
	}
	platform = strings.TrimSpace(platform)
	return s.repo.UpsertDeviceToken(ctx, userID, tokenStr, platform)
}

// UnregisterDevice removes a device token owned by the authenticated user.
func (s *NotiService) UnregisterDevice(ctx context.Context, userID, tokenStr string) error {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return fmt.Errorf("%w: token is required", ErrInvalidInput)
	}
	return s.repo.DeleteDeviceToken(ctx, userID, tokenStr)
}

// ── event handling (consumer) ────────────────────────────────────────────────

// markProcessed claims an eventId via Redis SETNX. It returns true if this is
// the first time we've seen the event (and we should process it), false if it
// was already processed.
func (s *NotiService) markProcessed(ctx context.Context, eventID string) (bool, error) {
	ok, err := s.redis.SetNX(ctx, "processed:"+eventID, "1", s.processedTTL).Result()
	if err != nil {
		return false, fmt.Errorf("setnx processed:%s: %w", eventID, err)
	}
	return ok, nil
}

// unmarkProcessed releases an eventId claim so the event can be retried after a
// downstream failure (we only want to dedupe successfully-processed events).
func (s *NotiService) unmarkProcessed(ctx context.Context, eventID string) {
	if err := s.redis.Del(ctx, "processed:"+eventID).Err(); err != nil {
		s.log.Warn("failed to release processed marker", "event_id", eventID, "err", err)
	}
}

// ChatMessageData is the data section of a chat.message.created event.
type ChatMessageData struct {
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
	SenderID       string `json:"senderId"`
	RecipientID    string `json:"recipientId"`
}

// ListingModerationData is the data section of listing.approved / listing.rejected.
// The publisher (admin-service) sends listingId (+ reason on reject); sellerId is
// accepted opportunistically if a future publisher includes it.
type ListingModerationData struct {
	ListingID string `json:"listingId"`
	SellerID  string `json:"sellerId"`
	UserID    string `json:"userId"`
	Reason    string `json:"reason"`
}

// HandleChatMessage notifies the recipient of a new chat message.
func (s *NotiService) HandleChatMessage(ctx context.Context, eventID string, d ChatMessageData) error {
	if d.RecipientID == "" {
		s.log.Warn("chat.message.created missing recipientId", "event_id", eventID)
		return nil // nothing to deliver; treat as handled (don't DLQ)
	}
	return s.deliver(ctx, eventID, d.RecipientID, "chat.message.created", push.Notification{
		Title: "Tin nhắn mới",
		Body:  "Bạn có tin nhắn mới",
		Data: map[string]string{
			"type":           "chat.message.created",
			"conversationId": d.ConversationID,
			"messageId":      d.MessageID,
			"senderId":       d.SenderID,
		},
	}, d)
}

// HandleListingModeration notifies the seller of an approval/rejection decision.
func (s *NotiService) HandleListingModeration(ctx context.Context, eventID, eventType string, d ListingModerationData) error {
	target := d.SellerID
	if target == "" {
		target = d.UserID
	}
	if target == "" {
		// The current admin-service payload only carries listingId, so we have no
		// user to push to. Log and treat as handled rather than dead-lettering.
		s.log.Warn("listing moderation event has no seller to notify",
			"event_id", eventID, "type", eventType, "listing_id", d.ListingID)
		return nil
	}

	var body string
	if eventType == "listing.approved" {
		body = "Tin đăng của bạn đã được duyệt"
	} else {
		body = "Tin đăng của bạn đã bị từ chối"
	}

	return s.deliver(ctx, eventID, target, eventType, push.Notification{
		Title: "Cập nhật tin đăng",
		Body:  body,
		Data: map[string]string{
			"type":      eventType,
			"listingId": d.ListingID,
			"reason":    d.Reason,
		},
	}, d)
}

// deliver runs the idempotent core: claim the eventId, persist the notification
// row, then push. On any failure it releases the claim so the event can be
// retried (the message will be nacked without requeue → DLQ, but the marker no
// longer blocks a manual replay).
func (s *NotiService) deliver(ctx context.Context, eventID, userID, notiType string, n push.Notification, payload any) error {
	first, err := s.markProcessed(ctx, eventID)
	if err != nil {
		return err
	}
	if !first {
		s.log.Info("event already processed; skipping", "event_id", eventID, "type", notiType)
		return nil
	}

	if _, err := s.repo.InsertNotification(ctx, userID, notiType, payload); err != nil {
		s.unmarkProcessed(ctx, eventID)
		return fmt.Errorf("persist notification: %w", err)
	}

	tokens, err := s.repo.TokensForUser(ctx, userID)
	if err != nil {
		s.unmarkProcessed(ctx, eventID)
		return fmt.Errorf("load device tokens: %w", err)
	}

	if err := s.push.Send(ctx, tokens, n); err != nil {
		s.unmarkProcessed(ctx, eventID)
		return fmt.Errorf("send push: %w", err)
	}

	s.log.Info("notification delivered",
		"event_id", eventID, "type", notiType, "user_id", userID, "tokens", len(tokens))
	return nil
}
