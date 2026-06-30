package game

import (
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type PlayOrderEntry struct {
	Place    int
	UserUUID uuid.UUID
	Username string
	Online   bool
}

type SetPlayOrderEntry struct {
	Place    int
	UserUUID uuid.UUID
}

func (s *Service) JoinGame(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, exists := s.games[gameID]
	if !exists {
		log.Warn().Str("game_id", gameID).Msg("join game rejected: game not found")
		return ErrGameNotFound
	}

	user, exists := s.users[token]
	if !exists {
		log.Warn().Str("game_id", gameID).Msg("join game rejected: user not found")
		return ErrUserNotFound
	}

	if _, alreadyJoined := game.players[user.userID]; alreadyJoined {
		s.leaveOtherGamesLocked(user.userID, gameID)
		log.Debug().Str("game_id", gameID).Str("user_id", user.userID.String()).Msg("join game ignored: user already joined")
		return nil
	}

	s.leaveOtherGamesLocked(user.userID, gameID)
	game.players[user.userID] = user
	game.lastSeenByUser[user.userID] = s.now()
	game.allOfflineSince = time.Time{}
	game.appendPlayer(user.userID)
	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("players", len(game.players)).Msg("player joined game")

	return nil
}

func (s *Service) GetPlayOrder(gameID string, token uuid.UUID) ([]PlayOrderEntry, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return nil, err
	}
	if _, isPlayer := game.players[user.userID]; !isPlayer {
		return nil, ErrForbidden
	}

	return game.playOrderEntries(game.players, s.now()), nil
}

func (s *Service) SetPlayOrder(gameID string, token uuid.UUID, entries []SetPlayOrderEntry) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Msg("set play order rejected: user is not owner")
		return ErrForbidden
	}

	if err := game.setPlayOrder(entries); err != nil {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Err(err).Msg("set play order rejected")
		return err
	}

	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("players", len(entries)).Msg("play order set")
	return nil
}

func (s *Service) gameAndUser(gameID string, token uuid.UUID) (*Games, *User, error) {
	if s.discardExpiredGameLocked(gameID, s.now()) {
		log.Warn().Str("game_id", gameID).Msg("game lookup failed: game expired")
		return nil, nil, ErrGameNotFound
	}

	game, exists := s.games[gameID]
	if !exists {
		log.Warn().Str("game_id", gameID).Msg("game lookup failed: game not found")
		return nil, nil, ErrGameNotFound
	}

	user, exists := s.users[token]
	if !exists {
		log.Warn().Str("game_id", gameID).Msg("game lookup failed: user not found")
		return nil, nil, ErrUserNotFound
	}

	return game, user, nil
}

func (g *Games) appendPlayer(userID uuid.UUID) {
	if _, exists := g.placeByUser[userID]; exists {
		return
	}

	g.usersByPlace = append(g.usersByPlace, userID)
	g.placeByUser[userID] = len(g.usersByPlace)
}

func (g *Games) playOrderEntries(players map[uuid.UUID]*User, now time.Time) []PlayOrderEntry {
	entries := make([]PlayOrderEntry, 0, len(g.usersByPlace))
	for place, userID := range g.usersByPlace {
		user := players[userID]
		if user == nil {
			continue
		}

		entries = append(entries, PlayOrderEntry{
			Place:    place + 1,
			UserUUID: user.userID,
			Username: user.username,
			Online:   g.onlineAt(userID, now),
		})
	}

	return entries
}

func (g *Games) setPlayOrder(entries []SetPlayOrderEntry) error {
	if len(entries) != len(g.players) {
		return ErrInvalidPlayOrder
	}

	usersByPlace := make([]uuid.UUID, len(entries))
	placeByUser := make(map[uuid.UUID]int, len(entries))
	for _, entry := range entries {
		if entry.Place < 1 || entry.Place > len(entries) {
			return ErrInvalidPlayOrder
		}
		if _, isPlayer := g.players[entry.UserUUID]; !isPlayer {
			return ErrInvalidPlayOrder
		}
		if usersByPlace[entry.Place-1] != uuid.Nil {
			return ErrInvalidPlayOrder
		}
		if _, duplicateUser := placeByUser[entry.UserUUID]; duplicateUser {
			return ErrInvalidPlayOrder
		}

		usersByPlace[entry.Place-1] = entry.UserUUID
		placeByUser[entry.UserUUID] = entry.Place
	}

	g.usersByPlace = usersByPlace
	g.placeByUser = placeByUser

	return nil
}
