package auth

import (
	"crypto/subtle"
	"fmt"
	"sync"
)

type User struct {
	Username string
	Password string
}

type Store struct {
	mu    sync.RWMutex
	users map[string]string
}

func NewStore() *Store {
	return &Store{users: make(map[string]string)}
}

func (s *Store) AddUser(username, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[username] = password
}

func (s *Store) Check(username, password string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	expected, ok := s.users[username]
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(password), []byte(expected)) == 1
}

func (s *Store) Users() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.users))
	for u := range s.users {
		names = append(names, u)
	}
	return names
}

func (s *Store) String() string {
	return fmt.Sprintf("auth store with %d user(s)", len(s.Users()))
}
