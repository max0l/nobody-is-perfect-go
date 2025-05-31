package game

import (
	"github.com/google/uuid"
)

type User struct {
	token    uuid.UUID
	username string
	userID   uuid.UUID
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
}

func NewService() *Service {
	return &Service{
		users: make(map[uuid.UUID]*User),
		games: make(map[string]*Games),
	}
}
