package game

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

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

func TestNewServiceUsesConfiguredWordlistPath(t *testing.T) {
	wordlistPath := writeTestWordlist(t)

	service := NewServiceWithOptions(ServiceOptions{WordlistPath: wordlistPath})

	if len(service.words) <= minimumWordCount {
		t.Fatalf("expected more than %d words, got %d", minimumWordCount, len(service.words))
	}
}

func writeTestWordlist(t *testing.T) string {
	t.Helper()

	var builder strings.Builder
	for i := 0; i <= minimumWordCount; i++ {
		builder.WriteString("word")
		builder.WriteString(string(rune('a' + i%26)))
		builder.WriteString("\n")
	}
	path := t.TempDir() + "/words.txt"
	if err := os.WriteFile(path, []byte(builder.String()), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	return path
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
	if createdGame.gameStatus != GameStatusCreated {
		t.Fatalf("expected game status created, got %q", createdGame.gameStatus)
	}
	if createdGame.players[userID] != user {
		t.Fatal("expected creator to be added as player")
	}
	if createdGame.placeByUser[userID] != 1 {
		t.Fatal("expected creator to be first in play order")
	}
}

func TestCreateGameRejectsMaxConcurrentGames(t *testing.T) {
	service := NewServiceWithOptions(ServiceOptions{MaxConcurrentGames: 1})
	firstName := "first"
	first, err := service.CreateUser(&firstName)
	if err != nil {
		t.Fatalf("CreateUser first returned error: %v", err)
	}
	if _, err := service.CreateGame(*first.GetUserToken()); err != nil {
		t.Fatalf("CreateGame first returned error: %v", err)
	}
	secondName := "second"
	second, err := service.CreateUser(&secondName)
	if err != nil {
		t.Fatalf("CreateUser second returned error: %v", err)
	}

	if _, err := service.CreateGame(*second.GetUserToken()); !errors.Is(err, ErrMaxGamesReached) {
		t.Fatalf("expected ErrMaxGamesReached, got %v", err)
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

func TestJoinGameMovesPlayerOutOfPreviousGame(t *testing.T) {
	service := NewService()
	ownerName := "owner"
	owner, err := service.CreateUser(&ownerName)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	firstGameID, err := service.CreateGame(*owner.GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame first returned error: %v", err)
	}
	playerName := "player"
	player, err := service.CreateUser(&playerName)
	if err != nil {
		t.Fatalf("CreateUser player returned error: %v", err)
	}
	if err := service.JoinGame(firstGameID, *player.GetUserToken()); err != nil {
		t.Fatalf("JoinGame first returned error: %v", err)
	}
	secondOwnerName := "second-owner"
	secondOwner, err := service.CreateUser(&secondOwnerName)
	if err != nil {
		t.Fatalf("CreateUser second owner returned error: %v", err)
	}
	secondGameID, err := service.CreateGame(*secondOwner.GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame second returned error: %v", err)
	}

	if err := service.JoinGame(secondGameID, *player.GetUserToken()); err != nil {
		t.Fatalf("JoinGame second returned error: %v", err)
	}

	if _, err := service.GetStatus(firstGameID, *player.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected moved player to be forbidden in first game, got %v", err)
	}
	firstEntries, err := service.GetPlayOrder(firstGameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetPlayOrder first returned error: %v", err)
	}
	if len(firstEntries) != 1 || firstEntries[0].UserUUID != *owner.GetUserID() {
		t.Fatalf("expected only owner in first game, got %+v", firstEntries)
	}
	secondEntries, err := service.GetPlayOrder(secondGameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetPlayOrder second returned error: %v", err)
	}
	if len(secondEntries) != 2 {
		t.Fatalf("expected player in second game, got %+v", secondEntries)
	}
}

func TestCreateGameMovesOwnerOutOfPreviousGameAndTransfersOwnership(t *testing.T) {
	service := NewService()
	ownerName := "owner"
	owner, err := service.CreateUser(&ownerName)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	firstGameID, err := service.CreateGame(*owner.GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame first returned error: %v", err)
	}
	playerName := "player"
	player, err := service.CreateUser(&playerName)
	if err != nil {
		t.Fatalf("CreateUser player returned error: %v", err)
	}
	if err := service.JoinGame(firstGameID, *player.GetUserToken()); err != nil {
		t.Fatalf("JoinGame returned error: %v", err)
	}

	secondGameID, err := service.CreateGame(*owner.GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame second returned error: %v", err)
	}

	status, err := service.GetStatus(firstGameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus first returned error: %v", err)
	}
	if status.GameOwnerUUID != *player.GetUserID() {
		t.Fatalf("expected ownership transferred to remaining player, got %s", status.GameOwnerUUID)
	}
	if _, err := service.GetStatus(firstGameID, *owner.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected moved owner to be forbidden in first game, got %v", err)
	}
	if _, err := service.GetStatus(secondGameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("expected owner in second game, got %v", err)
	}
}

func TestCreateGameDeletesPreviousGameWhenNoPlayersRemain(t *testing.T) {
	service := NewService()
	ownerName := "owner"
	owner, err := service.CreateUser(&ownerName)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	firstGameID, err := service.CreateGame(*owner.GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame first returned error: %v", err)
	}

	if _, err := service.CreateGame(*owner.GetUserToken()); err != nil {
		t.Fatalf("CreateGame second returned error: %v", err)
	}

	if _, err := service.GetStatus(firstGameID, *owner.GetUserToken()); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("expected previous single-player game to be deleted, got %v", err)
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

func TestPingUpdatesPresenceAndPlayersBecomeOffline(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	service, gameID, owner, player := serviceWithTwoPlayersAt(t, base)

	status, err := service.GetStatus(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}
	for _, player := range status.Players {
		if !player.Online {
			t.Fatalf("expected player online immediately after join: %+v", player)
		}
	}

	service.now = func() time.Time { return base.Add(PlayerOfflineAfter + time.Second) }
	status, err = service.GetStatus(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error after timeout: %v", err)
	}
	for _, player := range status.Players {
		if player.Online {
			t.Fatalf("expected player offline after timeout: %+v", player)
		}
	}

	if err := service.PingGame(gameID, *player.GetUserToken()); err != nil {
		t.Fatalf("PingGame returned error: %v", err)
	}
	status, err = service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error after ping: %v", err)
	}
	foundPlayer := false
	for _, entry := range status.Players {
		if entry.UserUUID == *player.GetUserID() {
			foundPlayer = true
			if !entry.Online {
				t.Fatal("expected pinged player to be online")
			}
		}
	}
	if !foundPlayer {
		t.Fatal("expected player in status")
	}
}

func TestGameIsDiscardedAfterAllPlayersOfflineForGracePeriod(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	service, gameID, owner, _ := serviceWithTwoPlayersAt(t, base)

	service.now = func() time.Time { return base.Add(PlayerOfflineAfter + GameDiscardAfter + time.Second) }
	if _, err := service.GetStatus(gameID, *owner.GetUserToken()); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("expected ErrGameNotFound after all players offline grace period, got %v", err)
	}
	if service.games[gameID] != nil {
		t.Fatal("expected game to be removed")
	}
}

func TestKickPlayerRemovesPlayerAndPreventsFurtherMutation(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	if err := service.KickPlayer(gameID, *owner.GetUserToken(), *player.GetUserID()); err != nil {
		t.Fatalf("KickPlayer returned error: %v", err)
	}
	entries, err := service.GetPlayOrder(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetPlayOrder returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].UserUUID != *owner.GetUserID() {
		t.Fatalf("expected only owner after kick, got %+v", entries)
	}
	if err := service.PingGame(gameID, *player.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected kicked player ping to be forbidden, got %v", err)
	}
}

func TestKickPlayerRequiresOwnerAndCannotKickOwner(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	if err := service.KickPlayer(gameID, *player.GetUserToken(), *owner.GetUserID()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected non-owner kick to be forbidden, got %v", err)
	}
	if err := service.KickPlayer(gameID, *owner.GetUserToken(), *owner.GetUserID()); !errors.Is(err, ErrCannotKickOwner) {
		t.Fatalf("expected ErrCannotKickOwner, got %v", err)
	}
}

func TestKickCurrentRoundMasterRotatesMaster(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}
	ownerAnswer := "owner answer"
	playerAnswer := "player answer"
	if err := service.SendAnswer(gameID, *owner.GetUserToken(), &ownerAnswer); err != nil {
		t.Fatalf("SendAnswer owner returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &playerAnswer); err != nil {
		t.Fatalf("SendAnswer player returned error: %v", err)
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	}
	answers, err := service.GetAnswers(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	if err := service.VoteForAnswer(gameID, *player.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer returned error: %v", err)
	}
	if _, err := service.RevealRound(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("RevealRound returned error: %v", err)
	}
	if err := service.NextRound(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("NextRound returned error: %v", err)
	}
	status, err := service.GetStatus(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}
	if status.RoundMasterUUID != *player.GetUserID() {
		t.Fatalf("expected player as round master before kick, got %s", status.RoundMasterUUID)
	}

	if err := service.KickPlayer(gameID, *owner.GetUserToken(), *player.GetUserID()); err != nil {
		t.Fatalf("KickPlayer returned error: %v", err)
	}
	status, err = service.GetStatus(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error after kick: %v", err)
	}
	if status.RoundMasterUUID != *owner.GetUserID() {
		t.Fatalf("expected owner as round master after kick, got %s", status.RoundMasterUUID)
	}
}

func TestStartGameInitializesRoundMasterAndStatus(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}

	status, err := service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}
	if status.GameStatus != GameStatusStarted {
		t.Fatalf("expected game status started, got %v", status.GameStatus)
	}
	if status.Round != 1 {
		t.Fatalf("expected round 1, got %d", status.Round)
	}
	if status.RoundStatus != RoundStatusAnswering {
		t.Fatalf("expected round status answering, got %v", status.RoundStatus)
	}
	if status.RoundMasterUUID != *owner.GetUserID() {
		t.Fatalf("expected owner as first round master, got %s", status.RoundMasterUUID)
	}
}

func TestVotingRevealAndNextRoundFlow(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}
	ownerAnswer := "owner answer"
	playerAnswer := "player answer"
	if err := service.SendAnswer(gameID, *owner.GetUserToken(), &ownerAnswer); err != nil {
		t.Fatalf("SendAnswer owner returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &playerAnswer); err != nil {
		t.Fatalf("SendAnswer player returned error: %v", err)
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	}

	changedAnswer := "changed answer"
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &changedAnswer); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected ErrInvalidRound when changing answer during voting, got %v", err)
	}
	answers, err := service.GetAnswers(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	if len(answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(answers))
	}
	if err := service.VoteForAnswer(gameID, *owner.GetUserToken(), answers[0].AnswerID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for round master vote, got %v", err)
	}
	if err := service.VoteForAnswer(gameID, *player.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer first returned error: %v", err)
	}
	status, err := service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus voting returned error: %v", err)
	}
	if status.ReceivedVotes != 1 {
		t.Fatalf("expected one received vote, got %d", status.ReceivedVotes)
	}
	if err := service.VoteForAnswer(gameID, *player.GetUserToken(), answers[1].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer change returned error: %v", err)
	}
	status, err = service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus changed vote returned error: %v", err)
	}
	if status.ReceivedVotes != 1 {
		t.Fatalf("expected changed vote not to increase received votes, got %d", status.ReceivedVotes)
	}

	revealed, err := service.RevealRound(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("RevealRound returned error: %v", err)
	}
	voteCount := 0
	for _, answer := range revealed {
		voteCount += len(answer.Votes)
		if answer.AnswerID == answers[1].AnswerID && len(answer.Votes) != 1 {
			t.Fatalf("expected changed vote on second answer, got %+v", answer.Votes)
		}
	}
	if voteCount != 1 {
		t.Fatalf("expected one revealed vote, got %d", voteCount)
	}
	if err := service.NextRound(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("NextRound returned error: %v", err)
	}
	status, err = service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}
	if status.Round != 2 || status.RoundMasterUUID != *player.GetUserID() || status.RoundStatus != RoundStatusAnswering {
		t.Fatalf("unexpected next round status: %+v", status)
	}
}

func TestRevealRequiresAllEligiblePlayersToVote(t *testing.T) {
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
		t.Fatalf("JoinGame player returned error: %v", err)
	}
	secondPlayerName := "second"
	secondPlayer, err := service.CreateUser(&secondPlayerName)
	if err != nil {
		t.Fatalf("CreateUser second player returned error: %v", err)
	}
	if err := service.JoinGame(gameID, *secondPlayer.GetUserToken()); err != nil {
		t.Fatalf("JoinGame second player returned error: %v", err)
	}
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}
	for _, user := range []*User{owner, player, secondPlayer} {
		answer := *user.GetUserUsername() + " answer"
		if err := service.SendAnswer(gameID, *user.GetUserToken(), &answer); err != nil {
			t.Fatalf("SendAnswer returned error: %v", err)
		}
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	}
	answers, err := service.GetAnswers(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	if err := service.VoteForAnswer(gameID, *player.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer player returned error: %v", err)
	}
	if _, err := service.RevealRound(gameID, *owner.GetUserToken()); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected ErrInvalidRound before all votes, got %v", err)
	}
	if err := service.VoteForAnswer(gameID, *secondPlayer.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer second player returned error: %v", err)
	}
	if _, err := service.RevealRound(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("RevealRound returned error after all votes: %v", err)
	}
}

func TestAnswersAreRoundScopedAndReaderSeesAuthors(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	ownerAnswer := "owner answer"
	playerAnswer := "player answer"
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}

	if err := service.SendAnswer(gameID, *owner.GetUserToken(), &ownerAnswer); err != nil {
		t.Fatalf("SendAnswer owner returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &playerAnswer); err != nil {
		t.Fatalf("SendAnswer player returned error: %v", err)
	}

	readerAnswers, err := service.GetAnswers(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers reader returned error: %v", err)
	}
	if len(readerAnswers) != 2 {
		t.Fatalf("expected 2 reader answers, got %d", len(readerAnswers))
	}
	for _, answer := range readerAnswers {
		if answer.Label == "" || answer.AnswerID == uuid.Nil || answer.UserUUID == uuid.Nil || answer.Username == "" {
			t.Fatalf("expected reader answer to include label, ID, and author: %+v", answer)
		}
	}

	if _, err := service.GetAnswers(gameID, *player.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected non-master to wait during answering, got %v", err)
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	}

	playerAnswers, err := service.GetAnswers(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers player returned error: %v", err)
	}
	for _, answer := range playerAnswers {
		if answer.UserUUID != uuid.Nil || answer.Username != "" {
			t.Fatalf("expected non-reader answer to hide author: %+v", answer)
		}
	}
	if err := service.VoteForAnswer(gameID, *player.GetUserToken(), playerAnswers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer returned error: %v", err)
	}

	if _, err := service.RevealRound(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("RevealRound returned error: %v", err)
	}
	if err := service.NextRound(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("NextRound returned error: %v", err)
	}
	nextRoundAnswers, err := service.GetAnswers(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers next round returned error: %v", err)
	}
	if len(nextRoundAnswers) != 0 {
		t.Fatalf("expected next round answers to start empty, got %+v", nextRoundAnswers)
	}
}

func TestAnswersAreLimitedToLabelsAThroughD(t *testing.T) {
	service := NewService()
	users := make([]*User, 0, 5)
	for i := 0; i < 5; i++ {
		username := string(rune('a' + i))
		user, err := service.CreateUser(&username)
		if err != nil {
			t.Fatalf("CreateUser returned error: %v", err)
		}
		users = append(users, user)
	}

	gameID, err := service.CreateGame(*users[0].GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}
	for _, user := range users[1:] {
		if err := service.JoinGame(gameID, *user.GetUserToken()); err != nil {
			t.Fatalf("JoinGame returned error: %v", err)
		}
	}
	if err := service.StartGame(gameID, *users[0].GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}
	for i, user := range users {
		answer := string(rune('A' + i))
		if err := service.SendAnswer(gameID, *user.GetUserToken(), &answer); err != nil {
			t.Fatalf("SendAnswer returned error: %v", err)
		}
	}

	answers, err := service.GetAnswers(gameID, *users[0].GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	if len(answers) != 4 {
		t.Fatalf("expected 4 displayed answers, got %d", len(answers))
	}
	for i, answer := range answers {
		if answer.Label != answerLabels[i] {
			t.Fatalf("expected label %q, got %q", answerLabels[i], answer.Label)
		}
	}
}

func serviceWithTwoPlayers(t *testing.T) (*Service, string, *User, *User) {
	t.Helper()

	return serviceWithTwoPlayersAt(t, time.Now())
}

func serviceWithTwoPlayersAt(t *testing.T, now time.Time) (*Service, string, *User, *User) {
	t.Helper()

	service := NewService()
	service.now = func() time.Time { return now }
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
