package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marketplace/media-service/internal/model"
)

// ErrNotFound is returned when a media row does not exist.
var ErrNotFound = errors.New("media not found")

type MediaRepo struct {
	pool *pgxpool.Pool
}

func NewMediaRepo(pool *pgxpool.Pool) *MediaRepo { return &MediaRepo{pool: pool} }

const mediaColumns = `id, owner_id, original_name, content_type, sizes, created_at`

func scanMedia(row pgx.Row) (*model.Media, error) {
	var (
		m         model.Media
		original  *string
		contentTy *string
		sizesRaw  []byte
	)
	err := row.Scan(&m.ID, &m.OwnerID, &original, &contentTy, &sizesRaw, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if original != nil {
		m.OriginalName = *original
	}
	if contentTy != nil {
		m.ContentType = *contentTy
	}
	m.Sizes = map[string]string{}
	if len(sizesRaw) > 0 {
		if err := json.Unmarshal(sizesRaw, &m.Sizes); err != nil {
			return nil, err
		}
	}
	return &m, nil
}

// Create inserts a new media row. Sizes is marshalled to JSONB.
func (r *MediaRepo) Create(ctx context.Context, m *model.Media) error {
	sizesJSON, err := json.Marshal(m.Sizes)
	if err != nil {
		return err
	}
	// Pass the JSON as text with an explicit ::jsonb cast: pgx encodes a Go
	// string as text (which Postgres casts to jsonb), whereas a raw []byte would
	// be sent as bytea and rejected by the jsonb column.
	_, err = r.pool.Exec(ctx,
		`INSERT INTO media (id, owner_id, original_name, content_type, sizes)
		 VALUES ($1, $2, $3, $4, $5::jsonb)`,
		m.ID, m.OwnerID, nullable(m.OriginalName), nullable(m.ContentType), string(sizesJSON),
	)
	return err
}

// GetByID returns a single media row or ErrNotFound.
func (r *MediaRepo) GetByID(ctx context.Context, id string) (*model.Media, error) {
	return scanMedia(r.pool.QueryRow(ctx, `SELECT `+mediaColumns+` FROM media WHERE id = $1`, id))
}

// Delete removes a media row by id. It returns the number of rows affected so
// the caller can distinguish a missing row from a successful delete.
func (r *MediaRepo) Delete(ctx context.Context, id string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM media WHERE id = $1`, id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// nullable maps an empty string to a SQL NULL so optional TEXT columns stay null.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
