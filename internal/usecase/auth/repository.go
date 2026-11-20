package auth

import "github.com/temlaka/coinwatch/internal/domain"

type UserRepository interface {
	Create(domain.User) error
	GetByUsername(username string) (domain.User, error)
}
