package game

import (
	"errors"
	"github.com/google/uuid"
	"sync"
)

type User struct {
	token    uuid.UUID
	username string
	userID   uuid.UUID
}

func (u *User) GetUserToken() *uuid.UUID {
	return &u.token
}

func (u *User) GetUserUsername() *string {
	return &u.username
}

func (u *User) GetUserID() *uuid.UUID {
	return &u.userID
}

type Answer struct {
	answer     string
	answerUUID uuid.UUID
}

type Games struct {
	gameID     string
	gameOwner  uuid.UUID
	gameStatus string
	players    map[uuid.UUID]*User
	answers    map[uuid.UUID]*Answer
}

type Service struct {
	users map[uuid.UUID]*User
	games map[string]*Games
	Lock  sync.Mutex
}

func (s *Service) CreateUser(username *string) (*User, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if username == nil || *username == "" {
		return nil, errors.New("no Username") // Handle error for empty username
	}

	userID := uuid.New()
	token := uuid.New()

	user := &User{
		token:    token,
		username: *username,
		userID:   userID,
	}

	s.users[token] = user

	return user, nil
}

func NewService() *Service {
	return &Service{
		users: make(map[uuid.UUID]*User),
		games: make(map[string]*Games),
	}
}
