package game

import (
	"fmt"

	"github.com/google/uuid"
)

type Answer struct {
	answerID uuid.UUID
	userID   uuid.UUID
	answer   string
}

type round struct {
	status        RoundStatus
	roundMasterID uuid.UUID
	answersByUser map[uuid.UUID]*Answer
	answersByID   map[uuid.UUID]*Answer
	scrambled     []uuid.UUID
	votesByUser   map[uuid.UUID]uuid.UUID
}

type Games struct {
	gameID       string
	gameOwner    uuid.UUID
	gameStatus   GameStatus
	currentRound int
	players      map[uuid.UUID]*User
	rounds       map[int]*round
	usersByPlace []uuid.UUID
	placeByUser  map[uuid.UUID]int
}

func (s *Service) CreateGame(gameCreator uuid.UUID) (string, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	user, exists := s.users[gameCreator]
	if !exists {
		return "", ErrUserNotFound
	}

	gameID := s.generateNewGame()

	newGame := &Games{
		gameID:       gameID,
		gameOwner:    user.userID,
		gameStatus:   GameStatusCreated,
		currentRound: 0,
		players:      make(map[uuid.UUID]*User),
		rounds:       make(map[int]*round),
		placeByUser:  make(map[uuid.UUID]int),
	}

	s.games[gameID] = newGame
	newGame.players[user.userID] = user
	newGame.appendPlayer(user.userID)

	return gameID, nil
}

func (g *Games) currentReaderID() uuid.UUID {
	state := g.currentRoundState()
	if state.roundMasterID == uuid.Nil {
		return uuid.Nil
	}

	return state.roundMasterID
}

func (g *Games) currentRoundState() *round {
	state := g.rounds[g.currentRound]
	if state == nil {
		state = newRound(uuid.Nil)
		g.rounds[g.currentRound] = state
	}

	return state
}

func (g *Games) startNextRound() {
	g.currentRound++
	g.rounds[g.currentRound] = newRound(g.usersByPlace[(g.currentRound-1)%len(g.usersByPlace)])
}

func newRound(roundMasterID uuid.UUID) *round {
	return &round{
		status:        RoundStatusAnswering,
		roundMasterID: roundMasterID,
		answersByUser: make(map[uuid.UUID]*Answer),
		answersByID:   make(map[uuid.UUID]*Answer),
		votesByUser:   make(map[uuid.UUID]uuid.UUID),
	}
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
