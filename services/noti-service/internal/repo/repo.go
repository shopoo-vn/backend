package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marketplace/noti-service/internal/model"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Repo holds SQL access to the noti schema (device_tokens + notifications).
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// UpsertDeviceToken inserts or refreshes a device token for a user. The token is
// the primary key, so re-registering an existing token reassigns ownership and
// platform.
func (r *Repo) UpsertDeviceToken(ctx context.Context, userID, tokenStr, platform string) (*model.DeviceToken, error) {
	var dt model.DeviceToken
	err := r.pool.QueryRow(ctx,
		`INSERT INTO device_tokens (user_id, token, platform)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (token)
		 DO UPDATE SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform
		 RETURNING user_id, token, platform, created_at`,
		userID, tokenStr, platform,
	).Scan(&dt.UserID, &dt.Token, &dt.Platform, &dt.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &dt, nil
}

// DeleteDeviceToken removes a token owned by the given user. It returns
// ErrNotFound if no matching row existed.
func (r *Repo) DeleteDeviceToken(ctx context.Context, userID, tokenStr string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM device_tokens WHERE token = $1 AND user_id = $2`,
		tokenStr, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TokensForUser returns every FCM token registered to a user.
func (r *Repo) TokensForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT token FROM device_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// InsertNotification persists a notification record for a user.
func (r *Repo) InsertNotification(ctx context.Context, userID, notiType string, payload any) (*model.Notification, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var n model.Notification
	err = r.pool.QueryRow(ctx,
		`INSERT INTO notifications (user_id, type, payload)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, type, payload, read, created_at`,
		userID, notiType, raw,
	).Scan(&n.ID, &n.UserID, &n.Type, &n.Payload, &n.Read, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
