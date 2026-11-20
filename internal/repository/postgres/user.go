package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/temlaka/coinwatch/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		pool: pool,
	}
}

func (u *UserRepo) Create(user domain.User) error {
	_, err := u.pool.Exec(context.Background(), "INSERT INTO users (username, password_hash) VALUES ($1, $2)", user.Username, user.PasswordHash)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("create %s: %w", user.Username, domain.ErrUserExists)
	}
	if err != nil {
		return fmt.Errorf("create %s: %w", user.Username, err)
	}
	return nil
}

func (u *UserRepo) GetByUsername(username string) (domain.User, error) {
	var user domain.User
	err := u.pool.QueryRow(context.Background(), "SELECT username, password_hash FROM users WHERE username = $1", username).Scan(&user.Username, &user.PasswordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, fmt.Errorf("get %s: %w", username, domain.ErrUserNotFound)
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get %s: %w", username, err)
	}
	return user, nil

}
