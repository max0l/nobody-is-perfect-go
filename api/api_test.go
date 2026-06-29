package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestCreateUserSetsSecureHTTPOnlySessionCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := NewServer()
	username := "alice"
	response := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(response)

	createdResponse, err := server.CreateUser(ginContext, CreateUserRequestObject{Body: &CreateUserJSONRequestBody{Username: &username}})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	created, ok := createdResponse.(CreateUser201JSONResponse)
	if !ok {
		t.Fatalf("expected CreateUser201JSONResponse, got %T", createdResponse)
	}

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != SessionCookieName {
		t.Fatalf("expected cookie name %q, got %q", SessionCookieName, cookie.Name)
	}
	if cookie.Value != created.UserToken.String() {
		t.Fatalf("expected session cookie value to match user token")
	}
	if !cookie.HttpOnly {
		t.Fatal("expected session cookie to be HttpOnly")
	}
	if !cookie.Secure {
		t.Fatal("expected session cookie to be Secure")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("expected SameSite=Strict, got %v", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Fatalf("expected cookie path /, got %q", cookie.Path)
	}
	if cookie.MaxAge != SessionCookieMaxAge {
		t.Fatalf("expected cookie max age %d, got %d", SessionCookieMaxAge, cookie.MaxAge)
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

func TestAnswerEndpointShowsAuthorsOnlyToCurrentReader(t *testing.T) {
	server, gameID, owner, player := serverWithGameAndJoinedPlayer(t)
	ownerCtx := context.WithValue(context.Background(), SessionCookieValueKey, owner.UserToken.String())
	playerCtx := context.WithValue(context.Background(), SessionCookieValueKey, player.UserToken.String())
	ownerAnswer := "owner answer"
	playerAnswer := "player answer"
	if response, err := server.StartGame(ownerCtx, StartGameRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	} else if _, ok := response.(StartGame200JSONResponse); !ok {
		t.Fatalf("expected StartGame200JSONResponse, got %T", response)
	}

	if response, err := server.SendAnswer(ownerCtx, SendAnswerRequestObject{GameId: gameID, Body: &SendAnswerJSONRequestBody{Answer: &ownerAnswer}}); err != nil {
		t.Fatalf("SendAnswer owner returned error: %v", err)
	} else if _, ok := response.(SendAnswer200JSONResponse); !ok {
		t.Fatalf("expected SendAnswer200JSONResponse, got %T", response)
	}
	if response, err := server.SendAnswer(playerCtx, SendAnswerRequestObject{GameId: gameID, Body: &SendAnswerJSONRequestBody{Answer: &playerAnswer}}); err != nil {
		t.Fatalf("SendAnswer player returned error: %v", err)
	} else if _, ok := response.(SendAnswer200JSONResponse); !ok {
		t.Fatalf("expected SendAnswer200JSONResponse, got %T", response)
	}

	readerResponse, err := server.GetAnswers(ownerCtx, GetAnswersRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetAnswers reader returned error: %v", err)
	}
	readerAnswers := readerResponse.(GetAnswers200JSONResponse)
	if readerAnswers.Answers == nil || len(*readerAnswers.Answers) != 2 {
		t.Fatalf("expected 2 reader answers, got %v", readerAnswers.Answers)
	}
	for _, answer := range *readerAnswers.Answers {
		if answer.Label == nil || answer.AnswerUUID == nil || answer.UserUUID == nil || answer.Username == nil {
			t.Fatalf("expected reader answer to include author fields: %+v", answer)
		}
	}

	waitResponse, err := server.GetAnswers(playerCtx, GetAnswersRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetAnswers waiting player returned error: %v", err)
	}
	if _, ok := waitResponse.(GetAnswers403JSONResponse); !ok {
		t.Fatalf("expected GetAnswers403JSONResponse while answering, got %T", waitResponse)
	}
	if response, err := server.StartVoting(ownerCtx, StartVotingRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	} else if _, ok := response.(StartVoting200JSONResponse); !ok {
		t.Fatalf("expected StartVoting200JSONResponse, got %T", response)
	}

	playerResponse, err := server.GetAnswers(playerCtx, GetAnswersRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetAnswers player returned error: %v", err)
	}
	playerAnswers := playerResponse.(GetAnswers200JSONResponse)
	for _, answer := range *playerAnswers.Answers {
		if answer.UserUUID != nil || answer.Username != nil {
			t.Fatalf("expected player answer to hide author fields: %+v", answer)
		}
	}
}

func TestVotingRevealAndStatusEndpoints(t *testing.T) {
	server, gameID, owner, player := serverWithGameAndJoinedPlayer(t)
	ownerCtx := context.WithValue(context.Background(), SessionCookieValueKey, owner.UserToken.String())
	playerCtx := context.WithValue(context.Background(), SessionCookieValueKey, player.UserToken.String())
	if response, err := server.StartGame(ownerCtx, StartGameRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	} else if _, ok := response.(StartGame200JSONResponse); !ok {
		t.Fatalf("expected StartGame200JSONResponse, got %T", response)
	}
	ownerAnswer := "owner answer"
	playerAnswer := "player answer"
	if _, err := server.SendAnswer(ownerCtx, SendAnswerRequestObject{GameId: gameID, Body: &SendAnswerJSONRequestBody{Answer: &ownerAnswer}}); err != nil {
		t.Fatalf("SendAnswer owner returned error: %v", err)
	}
	if _, err := server.SendAnswer(playerCtx, SendAnswerRequestObject{GameId: gameID, Body: &SendAnswerJSONRequestBody{Answer: &playerAnswer}}); err != nil {
		t.Fatalf("SendAnswer player returned error: %v", err)
	}
	statusResponse, err := server.GetGameStatus(playerCtx, GetGameStatusRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetGameStatus returned error: %v", err)
	}
	status := statusResponse.(GetGameStatus200JSONResponse)
	if status.Round == nil || *status.Round != 1 || status.RoundStatus == nil || *status.RoundStatus != "answering" {
		t.Fatalf("unexpected status response: %+v", status)
	}
	if status.RoundMasterUUID == nil || *status.RoundMasterUUID != *owner.UserUUID {
		t.Fatalf("expected owner as round master, got %+v", status.RoundMasterUUID)
	}

	if response, err := server.StartVoting(ownerCtx, StartVotingRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	} else if _, ok := response.(StartVoting200JSONResponse); !ok {
		t.Fatalf("expected StartVoting200JSONResponse, got %T", response)
	}
	answersResponse, err := server.GetAnswers(playerCtx, GetAnswersRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	answers := answersResponse.(GetAnswers200JSONResponse)
	if answers.Answers == nil || len(*answers.Answers) != 2 {
		t.Fatalf("expected 2 answers, got %+v", answers.Answers)
	}
	masterVoteResponse, err := server.VoteForAnswer(ownerCtx, VoteForAnswerRequestObject{GameId: gameID, Body: &VoteForAnswerJSONRequestBody{AnswerUUID: (*answers.Answers)[0].AnswerUUID}})
	if err != nil {
		t.Fatalf("VoteForAnswer master returned error: %v", err)
	}
	if _, ok := masterVoteResponse.(VoteForAnswer403JSONResponse); !ok {
		t.Fatalf("expected VoteForAnswer403JSONResponse, got %T", masterVoteResponse)
	}
	if response, err := server.VoteForAnswer(playerCtx, VoteForAnswerRequestObject{GameId: gameID, Body: &VoteForAnswerJSONRequestBody{AnswerUUID: (*answers.Answers)[0].AnswerUUID}}); err != nil {
		t.Fatalf("VoteForAnswer returned error: %v", err)
	} else if _, ok := response.(VoteForAnswer200JSONResponse); !ok {
		t.Fatalf("expected VoteForAnswer200JSONResponse, got %T", response)
	}
	statusResponse, err = server.GetGameStatus(playerCtx, GetGameStatusRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetGameStatus after vote returned error: %v", err)
	}
	status = statusResponse.(GetGameStatus200JSONResponse)
	if status.ReceivedVotes == nil || *status.ReceivedVotes != 1 {
		t.Fatalf("expected one received vote, got %+v", status.ReceivedVotes)
	}
	triggerResponse, err := server.TriggerReveal(ownerCtx, TriggerRevealRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("TriggerReveal returned error: %v", err)
	}
	revealed := triggerResponse.(TriggerReveal200JSONResponse)
	if revealed.Answers == nil || len(*revealed.Answers) != 2 {
		t.Fatalf("expected 2 revealed answers, got %+v", revealed.Answers)
	}
	voteCount := 0
	for _, answer := range *revealed.Answers {
		if answer.Votes != nil {
			voteCount += len(*answer.Votes)
		}
	}
	if voteCount != 1 {
		t.Fatalf("expected one revealed vote, got %d", voteCount)
	}
	if response, err := server.NextRound(ownerCtx, NextRoundRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("NextRound returned error: %v", err)
	} else if _, ok := response.(NextRound200JSONResponse); !ok {
		t.Fatalf("expected NextRound200JSONResponse, got %T", response)
	}
}

func TestRevealEndpointRequiresAllEligiblePlayersToVote(t *testing.T) {
	server, gameID, owner, player := serverWithGameAndJoinedPlayer(t)
	ownerCtx := context.WithValue(context.Background(), SessionCookieValueKey, owner.UserToken.String())
	playerCtx := context.WithValue(context.Background(), SessionCookieValueKey, player.UserToken.String())
	secondName := "second"
	secondResponse, err := server.CreateUser(context.Background(), CreateUserRequestObject{Body: &CreateUserJSONRequestBody{Username: &secondName}})
	if err != nil {
		t.Fatalf("CreateUser second returned error: %v", err)
	}
	second := secondResponse.(CreateUser201JSONResponse)
	secondCtx := context.WithValue(context.Background(), SessionCookieValueKey, second.UserToken.String())
	if response, err := server.JoinGame(secondCtx, JoinGameRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("JoinGame second returned error: %v", err)
	} else if _, ok := response.(JoinGame200JSONResponse); !ok {
		t.Fatalf("expected JoinGame200JSONResponse, got %T", response)
	}
	if response, err := server.StartGame(ownerCtx, StartGameRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	} else if _, ok := response.(StartGame200JSONResponse); !ok {
		t.Fatalf("expected StartGame200JSONResponse, got %T", response)
	}
	for _, entry := range []struct {
		ctx    context.Context
		answer string
	}{
		{ctx: ownerCtx, answer: "owner answer"},
		{ctx: playerCtx, answer: "player answer"},
		{ctx: secondCtx, answer: "second answer"},
	} {
		if response, err := server.SendAnswer(entry.ctx, SendAnswerRequestObject{GameId: gameID, Body: &SendAnswerJSONRequestBody{Answer: &entry.answer}}); err != nil {
			t.Fatalf("SendAnswer returned error: %v", err)
		} else if _, ok := response.(SendAnswer200JSONResponse); !ok {
			t.Fatalf("expected SendAnswer200JSONResponse, got %T", response)
		}
	}
	if response, err := server.StartVoting(ownerCtx, StartVotingRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	} else if _, ok := response.(StartVoting200JSONResponse); !ok {
		t.Fatalf("expected StartVoting200JSONResponse, got %T", response)
	}
	answersResponse, err := server.GetAnswers(playerCtx, GetAnswersRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	answers := answersResponse.(GetAnswers200JSONResponse)
	if answers.Answers == nil || len(*answers.Answers) == 0 {
		t.Fatalf("expected answers, got %+v", answers.Answers)
	}
	if response, err := server.VoteForAnswer(playerCtx, VoteForAnswerRequestObject{GameId: gameID, Body: &VoteForAnswerJSONRequestBody{AnswerUUID: (*answers.Answers)[0].AnswerUUID}}); err != nil {
		t.Fatalf("VoteForAnswer player returned error: %v", err)
	} else if _, ok := response.(VoteForAnswer200JSONResponse); !ok {
		t.Fatalf("expected VoteForAnswer200JSONResponse, got %T", response)
	}
	if response, err := server.TriggerReveal(ownerCtx, TriggerRevealRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("TriggerReveal before all votes returned error: %v", err)
	} else if _, ok := response.(TriggerReveal400JSONResponse); !ok {
		t.Fatalf("expected TriggerReveal400JSONResponse before all votes, got %T", response)
	}
	if response, err := server.VoteForAnswer(secondCtx, VoteForAnswerRequestObject{GameId: gameID, Body: &VoteForAnswerJSONRequestBody{AnswerUUID: (*answers.Answers)[0].AnswerUUID}}); err != nil {
		t.Fatalf("VoteForAnswer second returned error: %v", err)
	} else if _, ok := response.(VoteForAnswer200JSONResponse); !ok {
		t.Fatalf("expected VoteForAnswer200JSONResponse, got %T", response)
	}
	if response, err := server.TriggerReveal(ownerCtx, TriggerRevealRequestObject{GameId: gameID}); err != nil {
		t.Fatalf("TriggerReveal after all votes returned error: %v", err)
	} else if _, ok := response.(TriggerReveal200JSONResponse); !ok {
		t.Fatalf("expected TriggerReveal200JSONResponse after all votes, got %T", response)
	}
}

func TestPingGameEndpointUpdatesOnlineStatus(t *testing.T) {
	server, gameID, owner, _ := serverWithGameAndJoinedPlayer(t)
	ownerCtx := context.WithValue(context.Background(), SessionCookieValueKey, owner.UserToken.String())

	response, err := server.PingGame(ownerCtx, PingGameRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("PingGame returned error: %v", err)
	}
	if _, ok := response.(PingGame200JSONResponse); !ok {
		t.Fatalf("expected PingGame200JSONResponse, got %T", response)
	}

	statusResponse, err := server.GetGameStatus(ownerCtx, GetGameStatusRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetGameStatus returned error: %v", err)
	}
	status := statusResponse.(GetGameStatus200JSONResponse)
	if status.Users == nil || len(*status.Users) == 0 {
		t.Fatal("expected users in status")
	}
	for _, user := range *status.Users {
		if user.UserUUID != nil && *user.UserUUID == *owner.UserUUID {
			if user.Online == nil || !*user.Online {
				t.Fatal("expected pinged owner to be online")
			}
			return
		}
	}
	t.Fatal("expected owner in status")
}

func TestKickPlayerEndpointRequiresCreator(t *testing.T) {
	server, gameID, owner, player := serverWithGameAndJoinedPlayer(t)
	ownerCtx := context.WithValue(context.Background(), SessionCookieValueKey, owner.UserToken.String())
	playerCtx := context.WithValue(context.Background(), SessionCookieValueKey, player.UserToken.String())

	forbiddenResponse, err := server.KickPlayer(playerCtx, KickPlayerRequestObject{GameId: gameID, UserUUID: *owner.UserUUID})
	if err != nil {
		t.Fatalf("KickPlayer non-owner returned error: %v", err)
	}
	if _, ok := forbiddenResponse.(KickPlayer403JSONResponse); !ok {
		t.Fatalf("expected KickPlayer403JSONResponse, got %T", forbiddenResponse)
	}

	response, err := server.KickPlayer(ownerCtx, KickPlayerRequestObject{GameId: gameID, UserUUID: *player.UserUUID})
	if err != nil {
		t.Fatalf("KickPlayer owner returned error: %v", err)
	}
	if _, ok := response.(KickPlayer200JSONResponse); !ok {
		t.Fatalf("expected KickPlayer200JSONResponse, got %T", response)
	}

	playOrderResponse, err := server.GetPlayOrder(ownerCtx, GetPlayOrderRequestObject{GameId: gameID})
	if err != nil {
		t.Fatalf("GetPlayOrder returned error: %v", err)
	}
	playOrder := playOrderResponse.(GetPlayOrder200JSONResponse)
	if playOrder.PlayOrder == nil || len(*playOrder.PlayOrder) != 1 || *(*playOrder.PlayOrder)[0].UserUUID != *owner.UserUUID {
		t.Fatalf("expected only owner after kick, got %+v", playOrder.PlayOrder)
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
