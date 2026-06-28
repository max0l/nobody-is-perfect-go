package game

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrUsernameRequired = errors.New("username is required")
	ErrUserNotFound     = errors.New("user does not exist")
)

type Service struct {
	users map[uuid.UUID]*User
	games map[string]*Games
	Lock  sync.Mutex
}

func NewService() *Service {
	return &Service{
		users: make(map[uuid.UUID]*User),
		games: make(map[string]*Games),
	}
}
