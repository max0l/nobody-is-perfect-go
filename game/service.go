package game

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const minimumWordCount = 1000

var (
	ErrUsernameRequired = errors.New("username is required")
	ErrUserNotFound     = errors.New("user does not exist")
	ErrGameNotFound     = errors.New("game does not exist")
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidPlayOrder = errors.New("invalid play order")
)

type Service struct {
	users     map[uuid.UUID]*User
	usersByID map[uuid.UUID]*User
	games     map[string]*Games
	words     []string
	random    *rand.Rand
	Lock      sync.Mutex
}

func NewService() *Service {
	words, err := loadWords()
	if err != nil {
		panic(err)
	}

	return &Service{
		users:     make(map[uuid.UUID]*User),
		usersByID: make(map[uuid.UUID]*User),
		games:     make(map[string]*Games),
		words:     words,
		random:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func loadWords() ([]string, error) {
	file, err := openWordsFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read words file: %w", err)
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := scanner.Text()
		if word != "" {
			words = append(words, word)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan words file: %w", err)
	}
	if len(words) <= minimumWordCount {
		return nil, fmt.Errorf("words file must contain more than %d words", minimumWordCount)
	}

	return words, nil
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
