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

func TestCreateUserRejectsUsernameLongerThanMax(t *testing.T) {
	service := NewService()
	valid := strings.Repeat("a", MaxUsernameLength)
	if _, err := service.CreateUser(&valid); err != nil {
		t.Fatalf("expected %d-character username to be valid, got %v", MaxUsernameLength, err)
	}

	tooLong := strings.Repeat("ä", MaxUsernameLength+1)
	if _, err := service.CreateUser(&tooLong); !errors.Is(err, ErrUsernameTooLong) {
		t.Fatalf("expected ErrUsernameTooLong, got %v", err)
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

func TestGetSystemStatusCountsGamesPlayersAndOnlinePlayers(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	service, _, _, _ := serviceWithTwoPlayersAt(t, base)

	secondOwnerName := "second-owner"
	secondOwner, err := service.CreateUser(&secondOwnerName)
	if err != nil {
		t.Fatalf("CreateUser second owner returned error: %v", err)
	}
	if _, err := service.CreateGame(*secondOwner.GetUserToken()); err != nil {
		t.Fatalf("CreateGame second returned error: %v", err)
	}

	status := service.GetSystemStatus()
	if status.Games != 2 {
		t.Fatalf("expected 2 games, got %d", status.Games)
	}
	if status.Players != 3 {
		t.Fatalf("expected 3 players, got %d", status.Players)
	}
	if status.OnlinePlayers != 3 {
		t.Fatalf("expected 3 online players, got %d", status.OnlinePlayers)
	}

	service.now = func() time.Time { return base.Add(PlayerOfflineAfter + time.Second) }
	status = service.GetSystemStatus()
	if status.OnlinePlayers != 0 {
		t.Fatalf("expected 0 online players after timeout, got %d", status.OnlinePlayers)
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

func TestStatusUpdatesPresenceAndPlayersBecomeOffline(t *testing.T) {
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
	for _, entry := range status.Players {
		if entry.UserUUID == *owner.GetUserID() && !entry.Online {
			t.Fatal("expected polling owner to be online")
		}
		if entry.UserUUID == *player.GetUserID() && entry.Online {
			t.Fatalf("expected inactive player offline after timeout: %+v", entry)
		}
	}

	status, err = service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error after player poll: %v", err)
	}
	foundPlayer := false
	for _, entry := range status.Players {
		if entry.UserUUID == *player.GetUserID() {
			foundPlayer = true
			if !entry.Online {
				t.Fatal("expected polling player to be online")
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
	if _, err := service.GetStatus(gameID, *player.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected kicked player status to be forbidden, got %v", err)
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

func TestLeaveGameRemovesPlayer(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	if err := service.LeaveGame(gameID, *player.GetUserToken()); err != nil {
		t.Fatalf("LeaveGame returned error: %v", err)
	}

	entries, err := service.GetPlayOrder(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetPlayOrder returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].UserUUID != *owner.GetUserID() {
		t.Fatalf("expected only owner after leave, got %+v", entries)
	}
	if _, err := service.GetStatus(gameID, *player.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected leaving player to be forbidden, got %v", err)
	}
}

func TestLeaveGameTransfersOwnershipWhenOwnerLeaves(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	if err := service.LeaveGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("LeaveGame owner returned error: %v", err)
	}

	status, err := service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}
	if status.GameOwnerUUID != *player.GetUserID() {
		t.Fatalf("expected ownership transferred to player, got %s", status.GameOwnerUUID)
	}
	if status.PlayerCount != 1 {
		t.Fatalf("expected one player after owner leaves, got %d", status.PlayerCount)
	}
}

func TestLeaveGameDiscardsGameWhenLastPlayerLeaves(t *testing.T) {
	service := NewService()
	ownerName := "owner"
	owner, err := service.CreateUser(&ownerName)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	gameID, err := service.CreateGame(*owner.GetUserToken())
	if err != nil {
		t.Fatalf("CreateGame returned error: %v", err)
	}

	if err := service.LeaveGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("LeaveGame returned error: %v", err)
	}
	if service.games[gameID] != nil {
		t.Fatal("expected game to be removed")
	}
	if _, err := service.GetStatus(gameID, *owner.GetUserToken()); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("expected ErrGameNotFound after last player leaves, got %v", err)
	}
}

func TestFinishGameDeletesGame(t *testing.T) {
	service, gameID, owner, _ := serviceWithTwoPlayers(t)

	if err := service.FinishGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("FinishGame returned error: %v", err)
	}
	if service.games[gameID] != nil {
		t.Fatal("expected game to be removed")
	}
	if _, err := service.GetStatus(gameID, *owner.GetUserToken()); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("expected ErrGameNotFound after finish, got %v", err)
	}
}

func TestFinishGameRequiresOwner(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)

	if err := service.FinishGame(gameID, *player.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if _, err := service.GetStatus(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("expected game to remain after forbidden finish, got %v", err)
	}
}

func TestKickCurrentRoundMasterRotatesMaster(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	third := addServicePlayer(t, service, gameID, "third")
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}
	ownerAnswer := "owner answer"
	playerAnswer := "player answer"
	thirdAnswer := "third answer"
	if err := service.SendAnswer(gameID, *owner.GetUserToken(), &ownerAnswer); err != nil {
		t.Fatalf("SendAnswer owner returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &playerAnswer); err != nil {
		t.Fatalf("SendAnswer player returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *third.GetUserToken(), &thirdAnswer); err != nil {
		t.Fatalf("SendAnswer third returned error: %v", err)
	}
	if err := service.StartVerification(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVerification returned error: %v", err)
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
	if err := service.VoteForAnswer(gameID, *third.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer third returned error: %v", err)
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
	if status.RoundMasterUUID != *third.GetUserID() {
		t.Fatalf("expected third player as round master after kick, got %s", status.RoundMasterUUID)
	}
}

func TestStartGameRequiresMinimumPlayers(t *testing.T) {
	service, gameID, owner, _ := serviceWithTwoPlayers(t)

	if err := service.StartGame(gameID, *owner.GetUserToken()); !errors.Is(err, ErrNotEnoughPlayers) {
		t.Fatalf("expected ErrNotEnoughPlayers, got %v", err)
	}
	status, err := service.GetStatus(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}
	if status.GameStatus != GameStatusCreated || status.Round != 0 {
		t.Fatalf("expected game to remain in lobby, got %+v", status)
	}
}

func TestStartGameInitializesRoundMasterAndStatus(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	addServicePlayer(t, service, gameID, "third")

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

func TestGetStatusIncludesCurrentUsersAnswer(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	addServicePlayer(t, service, gameID, "third")
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}

	answer := "player answer"
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &answer); err != nil {
		t.Fatalf("SendAnswer returned error: %v", err)
	}
	status, err := service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}
	if status.CurrentAnswer != answer {
		t.Fatalf("expected current answer %q, got %q", answer, status.CurrentAnswer)
	}
	if status.CurrentAnswerID == uuid.Nil {
		t.Fatal("expected current answer ID")
	}
	answerID := status.CurrentAnswerID
	if status.ReceivedAnswers != 1 {
		t.Fatalf("expected one received answer, got %d", status.ReceivedAnswers)
	}

	updatedAnswer := "updated player answer"
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &updatedAnswer); err != nil {
		t.Fatalf("SendAnswer update returned error: %v", err)
	}
	status, err = service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus after update returned error: %v", err)
	}
	if status.CurrentAnswer != updatedAnswer {
		t.Fatalf("expected updated current answer %q, got %q", updatedAnswer, status.CurrentAnswer)
	}
	if status.CurrentAnswerID != answerID {
		t.Fatalf("expected overwritten answer to keep ID %s, got %s", answerID, status.CurrentAnswerID)
	}
	if status.ReceivedAnswers != 1 {
		t.Fatalf("expected overwritten answer not to increase count, got %d", status.ReceivedAnswers)
	}
}

func TestVerificationAllowsRoundMasterToDeleteAnswersBeforeVoting(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	third := addServicePlayer(t, service, gameID, "third")
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}
	for _, entry := range []struct {
		user   *User
		answer string
	}{
		{user: owner, answer: "owner answer"},
		{user: player, answer: "player answer"},
		{user: third, answer: "third answer"},
	} {
		if err := service.SendAnswer(gameID, *entry.user.GetUserToken(), &entry.answer); err != nil {
			t.Fatalf("SendAnswer returned error: %v", err)
		}
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected ErrInvalidRound starting voting before verification, got %v", err)
	}
	if err := service.StartVerification(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVerification returned error: %v", err)
	}
	status, err := service.GetStatus(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}
	if status.RoundStatus != RoundStatusVerifying {
		t.Fatalf("expected verifying status, got %v", status.RoundStatus)
	}
	if _, err := service.GetAnswers(gameID, *player.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected non-master to be forbidden during verification, got %v", err)
	}
	answers, err := service.GetAnswers(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers round master returned error: %v", err)
	}
	if len(answers) != 3 {
		t.Fatalf("expected 3 answers before deletion, got %d", len(answers))
	}
	if err := service.DeleteAnswer(gameID, *player.GetUserToken(), answers[0].AnswerID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected non-master delete forbidden, got %v", err)
	}
	deletedAnswerID := answers[0].AnswerID
	if err := service.DeleteAnswer(gameID, *owner.GetUserToken(), deletedAnswerID); err != nil {
		t.Fatalf("DeleteAnswer returned error: %v", err)
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	}
	votingAnswers, err := service.GetAnswers(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers player returned error: %v", err)
	}
	if len(votingAnswers) != 2 {
		t.Fatalf("expected 2 answers after deletion, got %d", len(votingAnswers))
	}
	for _, answer := range votingAnswers {
		if answer.AnswerID == deletedAnswerID {
			t.Fatalf("deleted answer appeared during voting: %+v", answer)
		}
	}
	if err := service.DeleteAnswer(gameID, *owner.GetUserToken(), votingAnswers[0].AnswerID); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected delete after voting to fail with ErrInvalidRound, got %v", err)
	}
}

func TestNonHostRoundMasterCannotStartVerificationBeforeAllAnswers(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	third := addServicePlayer(t, service, gameID, "third")
	if err := service.SetPlayOrder(gameID, *owner.GetUserToken(), []SetPlayOrderEntry{
		{UserUUID: *player.GetUserID(), Place: 1},
		{UserUUID: *owner.GetUserID(), Place: 2},
		{UserUUID: *third.GetUserID(), Place: 3},
	}); err != nil {
		t.Fatalf("SetPlayOrder returned error: %v", err)
	}
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
	if err := service.StartVerification(gameID, *player.GetUserToken()); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected ErrInvalidRound before all answers, got %v", err)
	}

	thirdAnswer := "third answer"
	if err := service.SendAnswer(gameID, *third.GetUserToken(), &thirdAnswer); err != nil {
		t.Fatalf("SendAnswer third returned error: %v", err)
	}
	if err := service.StartVerification(gameID, *player.GetUserToken()); err != nil {
		t.Fatalf("StartVerification after all answers returned error: %v", err)
	}
}

func TestHostCanStartVerificationAndRevealEarly(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	third := addServicePlayer(t, service, gameID, "third")
	if err := service.SetPlayOrder(gameID, *owner.GetUserToken(), []SetPlayOrderEntry{
		{UserUUID: *player.GetUserID(), Place: 1},
		{UserUUID: *owner.GetUserID(), Place: 2},
		{UserUUID: *third.GetUserID(), Place: 3},
	}); err != nil {
		t.Fatalf("SetPlayOrder returned error: %v", err)
	}
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
	if err := service.StartVerification(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("host StartVerification before all answers returned error: %v", err)
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("host StartVoting returned error: %v", err)
	}
	answers, err := service.GetAnswers(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	if err := service.VoteForAnswer(gameID, *owner.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer owner returned error: %v", err)
	}
	if _, err := service.RevealRound(gameID, *player.GetUserToken()); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected non-host round master reveal before all votes to fail, got %v", err)
	}
	if _, err := service.RevealRound(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("host RevealRound before all votes returned error: %v", err)
	}
}

func TestBelowMinimumPlayersBlocksNonHostProgressionExceptVotingReveal(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	third := addServicePlayer(t, service, gameID, "third")
	if err := service.SetPlayOrder(gameID, *owner.GetUserToken(), []SetPlayOrderEntry{
		{UserUUID: *player.GetUserID(), Place: 1},
		{UserUUID: *owner.GetUserID(), Place: 2},
		{UserUUID: *third.GetUserID(), Place: 3},
	}); err != nil {
		t.Fatalf("SetPlayOrder returned error: %v", err)
	}
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
	if err := service.LeaveGame(gameID, *third.GetUserToken()); err != nil {
		t.Fatalf("LeaveGame third returned error: %v", err)
	}
	if err := service.StartVerification(gameID, *player.GetUserToken()); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected non-host verification below minimum to fail, got %v", err)
	}
	if err := service.StartVerification(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("host StartVerification below minimum returned error: %v", err)
	}
	if err := service.StartVoting(gameID, *player.GetUserToken()); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected non-host voting below minimum to fail, got %v", err)
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("host StartVoting below minimum returned error: %v", err)
	}

	answers, err := service.GetAnswers(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	if err := service.VoteForAnswer(gameID, *owner.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer owner returned error: %v", err)
	}
	if _, err := service.RevealRound(gameID, *player.GetUserToken()); err != nil {
		t.Fatalf("non-host round master RevealRound during voting below minimum returned error: %v", err)
	}
	if err := service.NextRound(gameID, *player.GetUserToken()); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected non-host next round below minimum to fail, got %v", err)
	}
	if err := service.NextRound(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("host NextRound below minimum returned error: %v", err)
	}
}

func TestVotingRevealAndNextRoundFlow(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	third := addServicePlayer(t, service, gameID, "third")
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}
	ownerAnswer := "owner answer"
	playerAnswer := "player answer"
	thirdAnswer := "third answer"
	if err := service.SendAnswer(gameID, *owner.GetUserToken(), &ownerAnswer); err != nil {
		t.Fatalf("SendAnswer owner returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &playerAnswer); err != nil {
		t.Fatalf("SendAnswer player returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *third.GetUserToken(), &thirdAnswer); err != nil {
		t.Fatalf("SendAnswer third returned error: %v", err)
	}
	if err := service.StartVerification(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVerification returned error: %v", err)
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
	if len(answers) != 3 {
		t.Fatalf("expected 3 answers, got %d", len(answers))
	}
	if err := service.VoteForAnswer(gameID, *owner.GetUserToken(), answers[0].AnswerID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for round master vote, got %v", err)
	}
	if err := service.VoteForAnswer(gameID, *player.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer first returned error: %v", err)
	}
	if err := service.VoteForAnswer(gameID, *third.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer third returned error: %v", err)
	}
	status, err := service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus voting returned error: %v", err)
	}
	if status.ReceivedVotes != 2 {
		t.Fatalf("expected two received votes, got %d", status.ReceivedVotes)
	}
	if err := service.VoteForAnswer(gameID, *player.GetUserToken(), answers[1].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer change returned error: %v", err)
	}
	status, err = service.GetStatus(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetStatus changed vote returned error: %v", err)
	}
	if status.ReceivedVotes != 2 {
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
	if voteCount != 2 {
		t.Fatalf("expected two revealed votes, got %d", voteCount)
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
	if err := service.SetPlayOrder(gameID, *owner.GetUserToken(), []SetPlayOrderEntry{
		{UserUUID: *player.GetUserID(), Place: 1},
		{UserUUID: *owner.GetUserID(), Place: 2},
		{UserUUID: *secondPlayer.GetUserID(), Place: 3},
	}); err != nil {
		t.Fatalf("SetPlayOrder returned error: %v", err)
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
	if err := service.StartVerification(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVerification returned error: %v", err)
	}
	if err := service.StartVoting(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVoting returned error: %v", err)
	}
	answers, err := service.GetAnswers(gameID, *player.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers returned error: %v", err)
	}
	if err := service.VoteForAnswer(gameID, *owner.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer owner returned error: %v", err)
	}
	if _, err := service.RevealRound(gameID, *player.GetUserToken()); !errors.Is(err, ErrInvalidRound) {
		t.Fatalf("expected ErrInvalidRound before all votes, got %v", err)
	}
	if err := service.VoteForAnswer(gameID, *secondPlayer.GetUserToken(), answers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer second player returned error: %v", err)
	}
	if _, err := service.RevealRound(gameID, *player.GetUserToken()); err != nil {
		t.Fatalf("RevealRound returned error after all votes: %v", err)
	}
}

func TestAnswersAreRoundScopedAndReaderSeesAuthors(t *testing.T) {
	service, gameID, owner, player := serviceWithTwoPlayers(t)
	third := addServicePlayer(t, service, gameID, "third")
	ownerAnswer := "owner answer"
	playerAnswer := "player answer"
	thirdAnswer := "third answer"
	if err := service.StartGame(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartGame returned error: %v", err)
	}

	if err := service.SendAnswer(gameID, *owner.GetUserToken(), &ownerAnswer); err != nil {
		t.Fatalf("SendAnswer owner returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *player.GetUserToken(), &playerAnswer); err != nil {
		t.Fatalf("SendAnswer player returned error: %v", err)
	}
	if err := service.SendAnswer(gameID, *third.GetUserToken(), &thirdAnswer); err != nil {
		t.Fatalf("SendAnswer third returned error: %v", err)
	}

	readerAnswers, err := service.GetAnswers(gameID, *owner.GetUserToken())
	if err != nil {
		t.Fatalf("GetAnswers reader returned error: %v", err)
	}
	if len(readerAnswers) != 3 {
		t.Fatalf("expected 3 reader answers, got %d", len(readerAnswers))
	}
	for _, answer := range readerAnswers {
		if answer.Label == "" || answer.AnswerID == uuid.Nil || answer.UserUUID == uuid.Nil || answer.Username == "" {
			t.Fatalf("expected reader answer to include label, ID, and author: %+v", answer)
		}
	}

	if _, err := service.GetAnswers(gameID, *player.GetUserToken()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected non-master to wait during answering, got %v", err)
	}
	if err := service.StartVerification(gameID, *owner.GetUserToken()); err != nil {
		t.Fatalf("StartVerification returned error: %v", err)
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
	if err := service.VoteForAnswer(gameID, *third.GetUserToken(), playerAnswers[0].AnswerID); err != nil {
		t.Fatalf("VoteForAnswer third returned error: %v", err)
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

func TestAnswersAreLimitedToLabelsAThroughF(t *testing.T) {
	service := NewService()
	users := make([]*User, 0, 7)
	for i := 0; i < 7; i++ {
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
	if len(answers) != 6 {
		t.Fatalf("expected 6 displayed answers, got %d", len(answers))
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

func addServicePlayer(t *testing.T, service *Service, gameID string, username string) *User {
	t.Helper()

	user, err := service.CreateUser(&username)
	if err != nil {
		t.Fatalf("CreateUser %s returned error: %v", username, err)
	}
	if err := service.JoinGame(gameID, *user.GetUserToken()); err != nil {
		t.Fatalf("JoinGame %s returned error: %v", username, err)
	}

	return user
}
