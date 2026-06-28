package game

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCreateUserStoresUserByToken(t *testing.T) {
	service := NewService()
	username := "alice"

	user, err := service.CreateUser(&username)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if user.GetUserToken() == nil || *user.GetUserToken() == uuid.Nil {
		t.Fatal("expected user token to be set")
	}
	if user.GetUserID() == nil || *user.GetUserID() == uuid.Nil {
		t.Fatal("expected user ID to be set")
	}
	if got := user.GetUserUsername(); got == nil || *got != username {
		t.Fatalf("expected username %q, got %v", username, got)
	}
	if service.users[*user.GetUserToken()] != user {
		t.Fatal("expected service to store user by token")
	}
}

func TestNewServiceLoadsWordsAtStartup(t *testing.T) {
	service := NewService()

	if len(service.words) <= minimumWordCount {
		t.Fatalf("expected more than %d words, got %d", minimumWordCount, len(service.words))
	}
	if service.random == nil {
		t.Fatal("expected random source to be initialized")
	}
}

func TestCreateUserRequiresUsername(t *testing.T) {
	service := NewService()

	if _, err := service.CreateUser(nil); !errors.Is(err, ErrUsernameRequired) {
		t.Fatalf("expected ErrUsernameRequired for nil username, got %v", err)
	}

	empty := ""
	if _, err := service.CreateUser(&empty); !errors.Is(err, ErrUsernameRequired) {
		t.Fatalf("expected ErrUsernameRequired for empty username, got %v", err)
	}
}

func TestCreateGameRequiresKnownUserToken(t *testing.T) {
	service := NewService()

	if _, err := service.CreateGame(uuid.New()); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestCreateGameStoresCreatedGame(t *testing.T) {
	service := NewService()
	username := "alice"
	user, err := service.CreateUser(&username)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	token := *user.GetUserToken()
	gameID, err := service.CreateGame(token)
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}

	if gameID == "" {
		t.Fatal("expected game ID to be set")
	}
	if strings.Count(gameID, ".") != 2 {
		t.Fatalf("expected game ID to contain three words, got %q", gameID)
	}

	createdGame := service.games[gameID]
	if createdGame == nil {
		t.Fatal("expected created game to be stored")
	}
	if createdGame.gameOwner != token {
		t.Fatalf("expected game owner %s, got %s", token, createdGame.gameOwner)
	}
	if createdGame.gameStatus != "created" {
		t.Fatalf("expected game status created, got %q", createdGame.gameStatus)
	}
	if createdGame.players[token] != user {
		t.Fatal("expected creator to be added as player")
	}
}
