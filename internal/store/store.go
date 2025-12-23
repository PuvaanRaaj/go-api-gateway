package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials indicates the user record does not match the provided credentials.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrAPIKeyNotFound signals that the provided API key is missing or revoked.
	ErrAPIKeyNotFound = errors.New("api key not found")
)

// Identity represents the authenticated user.
type Identity struct {
	UserID uuid.UUID
	Email  string
}

// Store wraps database operations for users and API keys.
type Store struct {
	db *sql.DB
}

// New creates a Store backed by db.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// AuthenticateUser verifies the supplied credentials.
func (s *Store) AuthenticateUser(ctx context.Context, email, password string) (*Identity, error) {
	const query = `SELECT id, email, password_hash FROM users WHERE email = $1 LIMIT 1`

	var (
		id           uuid.UUID
		storedEmail  string
		passwordHash string
	)

	err := s.db.QueryRowContext(ctx, query, email).Scan(&id, &storedEmail, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &Identity{UserID: id, Email: storedEmail}, nil
}

// LookupAPIKey fetches a user identity by API key, ensuring the key is active.
func (s *Store) LookupAPIKey(ctx context.Context, key string) (*Identity, error) {
	const query = `
SELECT ak.user_id, u.email
FROM api_keys ak
JOIN users u ON ak.user_id = u.id
WHERE ak.key = $1
  AND ak.revoked = false
LIMIT 1`

	var (
		id    uuid.UUID
		email string
	)
	err := s.db.QueryRowContext(ctx, query, key).Scan(&id, &email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &Identity{UserID: id, Email: email}, nil
}
