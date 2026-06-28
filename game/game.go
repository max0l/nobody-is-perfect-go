package game

import (
	"fmt"

	"github.com/google/uuid"
)

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

func (s *Service) CreateGame(gameCreator uuid.UUID) (string, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	if _, exists := s.users[gameCreator]; !exists {
		return "", ErrUserNotFound
	}

	gameID := s.generateNewGame()

	newGame := &Games{
		gameID:     gameID,
		gameOwner:  gameCreator,
		gameStatus: "created",
		players:    make(map[uuid.UUID]*User),
		answers:    make(map[uuid.UUID]*Answer),
	}

	s.games[gameID] = newGame
	newGame.players[gameCreator] = s.users[gameCreator]

	return gameID, nil
}

func (s *Service) generateNewGame() string {
	gameID := s.getThreeWords()
	for gameAlreadyExists := true; gameAlreadyExists; {
		if s.games[gameID] != nil {
			gameID = s.getThreeWords()
			continue
		}
		gameAlreadyExists = false
	}
	return gameID
}

func (s *Service) getThreeWords() string {
	selected := make([]string, 3)
	for i := 0; i < 3; i++ {
		selected[i] = s.words[s.random.Intn(len(s.words))]
	}

	return fmt.Sprintf("%s.%s.%s", selected[0], selected[1], selected[2])
}
