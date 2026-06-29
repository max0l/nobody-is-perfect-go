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

const (
	minimumWordCount    = 1000
	PlayerOfflineAfter  = 15 * time.Second
	GameDiscardAfter    = 60 * time.Second
	DefaultWordlistPath = "words.txt"
	DefaultMaxGames     = 100
)

var (
	ErrUsernameRequired = errors.New("username is required")
	ErrUserNotFound     = errors.New("user does not exist")
	ErrGameNotFound     = errors.New("game does not exist")
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidPlayOrder = errors.New("invalid play order")
	ErrAnswerRequired   = errors.New("answer is required")
	ErrInvalidRound     = errors.New("invalid round state")
	ErrAnswerNotFound   = errors.New("answer does not exist")
	ErrCannotKickOwner  = errors.New("cannot kick game owner")
	ErrPlayerNotFound   = errors.New("player does not exist")
	ErrMaxGamesReached  = errors.New("max concurrent games reached")
)

type ServiceOptions struct {
	WordlistPath       string
	MaxConcurrentGames int
}

type Service struct {
	users              map[uuid.UUID]*User
	usersByID          map[uuid.UUID]*User
	games              map[string]*Games
	words              []string
	maxConcurrentGames int
	random             *rand.Rand
	now                func() time.Time
	Lock               sync.Mutex
}

func NewService() *Service {
	return NewServiceWithOptions(ServiceOptions{})
}

func NewServiceWithOptions(options ServiceOptions) *Service {
	if options.WordlistPath == "" {
		options.WordlistPath = DefaultWordlistPath
	}
	if options.MaxConcurrentGames == 0 {
		options.MaxConcurrentGames = DefaultMaxGames
	}

	words, err := loadWords(options.WordlistPath)
	if err != nil {
		panic(err)
	}

	return &Service{
		users:              make(map[uuid.UUID]*User),
		usersByID:          make(map[uuid.UUID]*User),
		games:              make(map[string]*Games),
		words:              words,
		maxConcurrentGames: options.MaxConcurrentGames,
		random:             rand.New(rand.NewSource(time.Now().UnixNano())),
		now:                time.Now,
	}
}

func loadWords(path string) ([]string, error) {
	file, err := openWordsFile(path)
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

func openWordsFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err == nil {
		return file, nil
	}
	if path == DefaultWordlistPath {
		return os.Open(filepath.Join("..", path))
	}

	return nil, err
}
