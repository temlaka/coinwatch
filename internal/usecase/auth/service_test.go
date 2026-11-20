package auth_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/temlaka/coinwatch/internal/domain"
	"github.com/temlaka/coinwatch/internal/repository/memory"
	"github.com/temlaka/coinwatch/internal/usecase/auth"
	"github.com/temlaka/coinwatch/pkg/token"

	"golang.org/x/crypto/bcrypt"
)

const secret = "test-secret"

func TestRegister(t *testing.T) {
	repo := memory.NewUserRepo()
	svc := auth.NewService(repo, secret)

	tok, err := svc.Register("alice", "secret")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	sub, err := token.ParseToken(tok, secret)
	if err != nil || sub != "alice" {
		t.Errorf("token sub = %q, err = %v; want alice", sub, err)
	}

	user, err := repo.GetByUsername("alice")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if bytes.Equal(user.PasswordHash, []byte("secret")) {
		t.Error("password stored in plain text")
	}
	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte("secret")); err != nil {
		t.Errorf("stored hash does not match password: %v", err)
	}
}

func TestRegister_Errors(t *testing.T) {
	cases := []struct {
		name     string
		username string
		password string
		want     error
	}{
		{"empty username", "", "secret", domain.ErrInvalidInput},
		{"empty password", "bob", "", domain.ErrInvalidInput},
		{"password longer than 72 bytes", "bob", strings.Repeat("a", 73), domain.ErrInvalidInput},
		{"duplicate", "alice", "other", domain.ErrUserExists},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := auth.NewService(memory.NewUserRepo(), secret)
			if _, err := svc.Register("alice", "secret"); err != nil {
				t.Fatalf("setup Register: %v", err)
			}

			tok, err := svc.Register(tc.username, tc.password)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if tok != "" {
				t.Errorf("token = %q, want empty", tok)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	svc := auth.NewService(memory.NewUserRepo(), secret)
	if _, err := svc.Register("alice", "secret"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	cases := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{"ok", "alice", "secret", nil},
		{"wrong password", "alice", "wrong", domain.ErrInvalidCredentials},
		{"unknown user", "nobody", "secret", domain.ErrInvalidCredentials},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tok, err := svc.Login(tc.username, tc.password)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			sub, err := token.ParseToken(tok, secret)
			if err != nil || sub != tc.username {
				t.Errorf("token sub = %q, err = %v; want %s", sub, err, tc.username)
			}
		})
	}
}
