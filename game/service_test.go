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
	if service.usersByID[*user.GetUserID()] != user {
		t.Fatal("expected service to store user by ID")
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
	userID := *user.GetUserID()
	if createdGame.gameOwner != userID {
		t.Fatalf("expected game owner %s, got %s", userID, createdGame.gameOwner)
	}
	if createdGame.gameStatus != "created" {
		t.Fatalf("expected game status created, got %q", createdGame.gameStatus)
	}
	if createdGame.players[userID] != user {
		t.Fatal("expected creator to be added as player")
	}
	if createdGame.placeByUser[userID] != 1 {
		t.Fatal("expected creator to be first in play order")
	}
}

func TestJoinGameAppendsPlayerToOrder(t *testing.T) {
	service := NewService()
	ownerName := "owner"
	owner, err := service.CreateUser(&ownerName)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	gameID, err := service.CreateGame(*owner.GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}

	playerName := "player"
	player, err := service.CreateUser(&playerName)
	if err != nil {
		t.Fatalf("CreateUser player returned error: %v", err)
	}
	if err := service.JoinGame(gameID, *player.GetUserToken()); err != nil {
		t.Fatalf("JoinGame returned error: %v", err)
	}

	entries, err := service.GetPlayOrder(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetPlayOrder returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 play order entries, got %d", len(entries))
	}
	if entries[0].UserUUID != *owner.GetUserID() || entries[0].Place != 1 {
		t.Fatalf("expected owner first, got %+v", entries[0])
	}
	if entries[1].UserUUID != *player.GetUserID() || entries[1].Place != 2 {
		t.Fatalf("expected player second, got %+v", entries[1])
	}
}

func TestSetPlayOrderReordersPlayers(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	err := service.SetPlayOrder(gameID, *owner.GetUserToken(), []SetPlayOrderEntry{
		{Place: 1, UserUUID: *player.GetUserID()},
		{Place: 2, UserUUID: *owner.GetUserID()},
	})
	if err != nil {
		t.Fatalf("SetPlayOrder returned error: %v", err)
	}

	entries, err := service.GetPlayOrder(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetPlayOrder returned error: %v", err)
	}
	if entries[0].UserUUID != *player.GetUserID() || entries[1].UserUUID != *owner.GetUserID() {
		t.Fatalf("unexpected play order: %+v", entries)
	}
}

func TestSetPlayOrderRequiresOwner(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	err := service.SetPlayOrder(gameID, *player.GetUserToken(), []SetPlayOrderEntry{
		{Place: 1, UserUUID: *player.GetUserID()},
		{Place: 2, UserUUID: *owner.GetUserID()},
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSetPlayOrderRejectsInvalidOrder(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	err := service.SetPlayOrder(gameID, *owner.GetUserToken(), []SetPlayOrderEntry{
		{Place: 1, UserUUID: *player.GetUserID()},
		{Place: 1, UserUUID: *owner.GetUserID()},
	})
	if !errors.Is(err, ErrInvalidPlayOrder) {
		t.Fatalf("expected ErrInvalidPlayOrder, got %v", err)
	}
}

func serviceWithTwoPlayers(t *testing.T) (*Service, string, *User, *User) {
	t.Helper()

	service := NewService()
	ownerName := "owner"
	owner, err := service.CreateUser(&ownerName)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	gameID, err := service.CreateGame(*owner.GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}
	playerName := "player"
	player, err := service.CreateUser(&playerName)
	if err != nil {
		t.Fatalf("CreateUser player returned error: %v", err)
	}
	if err := service.JoinGame(gameID, *player.GetUserToken()); err != nil {
		t.Fatalf("JoinGame returned error: %v", err)
	}

	return service, gameID, owner, player
}
