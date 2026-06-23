package access

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) BootstrapAdmin(ctx context.Context, login, passwordHash string) error {
	_, err := repository.pool.Exec(ctx, `
		INSERT INTO users (id, role, login, password_hash, enabled)
		VALUES ($1, 'admin', $2, $3, TRUE)
		ON CONFLICT (login) DO NOTHING`, uuid.NewString(), login, passwordHash)
	if err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}
	return nil
}

func (repository *Repository) UserByLogin(ctx context.Context, login string) (User, error) {
	var user User
	err := repository.pool.QueryRow(ctx, `
		SELECT u.id, u.role, u.login, u.password_hash, u.enabled,
		       u.client_account_id, c.status
		FROM users u
		LEFT JOIN client_accounts c ON c.id = u.client_account_id
		WHERE u.login = $1`, login,
	).Scan(&user.ID, &user.Role, &user.Login, &user.PasswordHash, &user.Enabled, &user.ClientAccountID, &user.ClientStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}
