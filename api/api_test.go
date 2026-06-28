package api

import (
	"context"
	"testing"
)

func TestCreateUserReturnsToken(t *testing.T) {
	server := NewServer()
	username := "alice"

	response, err := server.CreateUser(context.Background(), CreateUserRequestObject{Body: &CreateUserJSONRequestBody{Username: &username}})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	created, ok := response.(CreateUser201JSONResponse)
	if !ok {
		t.Fatalf("expected CreateUser201JSONResponse, got %T", response)
	}
	if created.UserToken == nil {
		t.Fatal("expected user token to be set")
	}
	if created.UserUUID == nil {
		t.Fatal("expected user UUID to be set")
	}
}

func TestCreateUserReturnsBadRequestForMissingUsername(t *testing.T) {
	server := NewServer()

	response, err := server.CreateUser(context.Background(), CreateUserRequestObject{Body: &CreateUserJSONRequestBody{}})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	badRequest, ok := response.(CreateUser400JSONResponse)
	if !ok {
		t.Fatalf("expected CreateUser400JSONResponse, got %T", response)
	}
	if badRequest.Error == nil || *badRequest.Error != BadRequestError {
		t.Fatalf("expected error %q, got %v", BadRequestError, badRequest.Error)
	}
}

func TestCreateGameReturnsUnauthorizedWithoutSession(t *testing.T) {
	server := NewServer()

	response, err := server.CreateGame(context.Background(), CreateGameRequestObject{})
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}

	unauthorized, ok := response.(CreateGame401JSONResponse)
	if !ok {
		t.Fatalf("expected CreateGame401JSONResponse, got %T", response)
	}
	if unauthorized.Error == nil || *unauthorized.Error != UnauthorizedError {
		t.Fatalf("expected error %q, got %v", UnauthorizedError, unauthorized.Error)
	}
}

func TestCreateGameReturnsUnauthorizedForInvalidSession(t *testing.T) {
	server := NewServer()
	ctx := context.WithValue(context.Background(), SessionCookieValueKey, "not-a-uuid")

	response, err := server.CreateGame(ctx, CreateGameRequestObject{})
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}

	if _, ok := response.(CreateGame401JSONResponse); !ok {
		t.Fatalf("expected CreateGame401JSONResponse, got %T", response)
	}
}

func TestCreateGameReturnsCreatedForValidSession(t *testing.T) {
	server := NewServer()
	username := "alice"
	createUserResponse, err := server.CreateUser(context.Background(), CreateUserRequestObject{Body: &CreateUserJSONRequestBody{Username: &username}})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	createdUser := createUserResponse.(CreateUser201JSONResponse)

	ctx := context.WithValue(context.Background(), SessionCookieValueKey, createdUser.UserToken.String())
	response, err := server.CreateGame(ctx, CreateGameRequestObject{})
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}

	createdGame, ok := response.(CreateGame201JSONResponse)
	if !ok {
		t.Fatalf("expected CreateGame201JSONResponse, got %T", response)
	}
	if createdGame.GameId == nil || *createdGame.GameId == "" {
		t.Fatal("expected game ID to be set")
	}
}

func TestJoinGameAndGetPlayOrder(t *testing.T) {
	server, gameID, owner, player := serverWithGameAndJoinedPlayer(t)
	ctx := context.WithValue(context.Background(), SessionCookieValueKey, owner.UserToken.String())

	response, err := server.GetPlayOrder(ctx, GetPlayOrderRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetPlayOrder returned error: %v", err)
	}

	playOrder, ok := response.(GetPlayOrder200JSONResponse)
	if !ok {
		t.Fatalf("expected GetPlayOrder200JSONResponse, got %T", response)
	}
	if playOrder.PlayOrder == nil || len(*playOrder.PlayOrder) != 2 {
		t.Fatalf("expected 2 play order entries, got %v", playOrder.PlayOrder)
	}
	if *(*playOrder.PlayOrder)[0].UserUUID != *owner.UserUUID {
		t.Fatalf("expected owner first, got %+v", (*playOrder.PlayOrder)[0])
	}
	if *(*playOrder.PlayOrder)[1].UserUUID != *player.UserUUID {
		t.Fatalf("expected player second, got %+v", (*playOrder.PlayOrder)[1])
	}
}

func TestSetPlayOrderReordersPlayers(t *testing.T) {
	server, gameID, owner, player := serverWithGameAndJoinedPlayer(t)
	ctx := context.WithValue(context.Background(), SessionCookieValueKey, owner.UserToken.String())
	one := 1
	two := 2
	body := SetPlayOrderJSONRequestBody{
		PlayOrder: &[]struct {
			Place    *int  `json:"place,omitempty"`
			UserUUID *UUID `json:"userUUID,omitempty"`
		}{
			{Place: &one, UserUUID: player.UserUUID},
			{Place: &two, UserUUID: owner.UserUUID},
		},
	}

	response, err := server.SetPlayOrder(ctx, SetPlayOrderRequestObject{GameId: gameID, Body: &body})
	if err != nil {
		t.Fatalf("SetPlayOrder returned error: %v", err)
	}
	if _, ok := response.(SetPlayOrder200JSONResponse); !ok {
		t.Fatalf("expected SetPlayOrder200JSONResponse, got %T", response)
	}

	getResponse, err := server.GetPlayOrder(ctx, GetPlayOrderRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetPlayOrder returned error: %v", err)
	}
	playOrder := getResponse.(GetPlayOrder200JSONResponse)
	if *(*playOrder.PlayOrder)[0].UserUUID != *player.UserUUID {
		t.Fatalf("expected player first after reorder, got %+v", (*playOrder.PlayOrder)[0])
	}
}

func TestSetPlayOrderRequiresOwner(t *testing.T) {
	server, gameID, owner, player := serverWithGameAndJoinedPlayer(t)
	ctx := context.WithValue(context.Background(), SessionCookieValueKey, player.UserToken.String())
	one := 1
	two := 2
	body := SetPlayOrderJSONRequestBody{
		PlayOrder: &[]struct {
			Place    *int  `json:"place,omitempty"`
			UserUUID *UUID `json:"userUUID,omitempty"`
		}{
			{Place: &one, UserUUID: player.UserUUID},
			{Place: &two, UserUUID: owner.UserUUID},
		},
	}

	response, err := server.SetPlayOrder(ctx, SetPlayOrderRequestObject{GameId: gameID, Body: &body})
	if err != nil {
		t.Fatalf("SetPlayOrder returned error: %v", err)
	}
	if _, ok := response.(SetPlayOrder403JSONResponse); !ok {
		t.Fatalf("expected SetPlayOrder403JSONResponse, got %T", response)
	}
}

func serverWithGameAndJoinedPlayer(t *testing.T) (*StrictServer, string, CreateUser201JSONResponse, CreateUser201JSONResponse) {
	t.Helper()

	server := NewServer()
	ownerName := "owner"
	ownerResponse, err := server.CreateUser(context.Background(), CreateUserRequestObject{Body: &CreateUserJSONRequestBody{Username: &ownerName}})
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	owner := ownerResponse.(CreateUser201JSONResponse)
	ownerCtx := context.WithValue(context.Background(), SessionCookieValueKey, owner.UserToken.String())
	gameResponse, err := server.CreateGame(ownerCtx, CreateGameRequestObject{})
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}
	gameID := *gameResponse.(CreateGame201JSONResponse).GameId

	playerName := "player"
	playerResponse, err := server.CreateUser(context.Background(), CreateUserRequestObject{Body: &CreateUserJSONRequestBody{Username: &playerName}})
	if err != nil {
		t.Fatalf("CreateUser player returned error: %v", err)
	}
	player := playerResponse.(CreateUser201JSONResponse)
	playerCtx := context.WithValue(context.Background(), SessionCookieValueKey, player.UserToken.String())
	joinResponse, err := server.JoinGame(playerCtx, JoinGameRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("JoinGame returned error: %v", err)
	}
	if _, ok := joinResponse.(JoinGame200JSONResponse); !ok {
		t.Fatalf("expected JoinGame200JSONResponse, got %T", joinResponse)
	}

	return server, gameID, owner, player
}
