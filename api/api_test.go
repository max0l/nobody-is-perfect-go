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
