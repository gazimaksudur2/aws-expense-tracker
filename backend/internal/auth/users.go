package auth

import (
	"errors"
	"sync"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrUserNotFound = errors.New("user not found")
)

// User is a phase 1 account. Phase 2 (Cognito) drops PasswordHash.
type User struct {
	UserID       string
	Email        string
	Name         string
	PasswordHash string
}

// UserStore is implemented by the in-memory store (tests) and DynamoDB (Step 3).
type UserStore interface {
	CreateUser(u User) error
	GetUserByEmail(email string) (User, error)
}

type MemUserStore struct {
	mu      sync.RWMutex
	byEmail map[string]User
}

func NewMemUserStore() *MemUserStore {
	return &MemUserStore{byEmail: map[string]User{}}
}

func (m *MemUserStore) CreateUser(u User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byEmail[u.Email]; exists {
		return ErrEmailTaken
	}
	m.byEmail[u.Email] = u
	return nil
}

func (m *MemUserStore) GetUserByEmail(email string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.byEmail[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}
