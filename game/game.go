package game

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

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
	gameID := getThreeWords()
	for gameAlreadyExists := true; gameAlreadyExists; {
		if s.games[gameID] != nil {
			gameID = getThreeWords()
			continue
		}
		gameAlreadyExists = false
	}
	return gameID
}

// getThreeWords is chosen 3 words from a file randomly.
func getThreeWords() string {
	file, err := openWordsFile()
	if err != nil {
		panic("Failed to read words file: " + err.Error())
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic("Failed to close words file: " + err.Error())
		}
	}(file)

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic("Error reading words file: " + err.Error())
	}

	if len(words) < 3 {
		panic("Words file must contain at least 3 words")
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	selected := make([]string, 3)
	for i := 0; i < 3; i++ {
		selected[i] = words[r.Intn(len(words))]
	}

	return fmt.Sprintf("%s.%s.%s", selected[0], selected[1], selected[2])

}

func openWordsFile() (*os.File, error) {
	for _, path := range []string{"words.txt", filepath.Join("..", "words.txt")} {
		file, err := os.Open(path)
		if err == nil {
			return file, nil
		}
	}

	return nil, os.ErrNotExist
}
