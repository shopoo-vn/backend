package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marketplace/auth-service/internal/model"
)

// ErrNotFound is returned when a user row does not exist.
var ErrNotFound = errors.New("user not found")

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

const userColumns = `id, email, phone, password_hash, display_name, avatar_url, role, status, created_at, updated_at`

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.DisplayName,
		&u.AvatarURL, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, phone, password_hash, display_name, avatar_url, role, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.ID, u.Email, u.Phone, u.PasswordHash, u.DisplayName, u.AvatarURL, u.Role, u.Status,
	)
	return err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *UserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

func (r *UserRepo) UpdateProfile(ctx context.Context, id, displayName string, avatarURL *string) (*model.User, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`UPDATE users SET display_name = $2, avatar_url = $3, updated_at = now()
		 WHERE id = $1
		 RETURNING `+userColumns,
		id, displayName, avatarURL,
	))
}

// List returns users newest-first for the admin console.
func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userColumns+` FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*model.User, 0, limit)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *UserRepo) UpdateStatus(ctx context.Context, id, status string) (*model.User, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`UPDATE users SET status = $2, updated_at = now() WHERE id = $1
		 RETURNING `+userColumns,
		id, status,
	))
}
