package game

import (
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func (s *Service) KickPlayer(gameID string, token uuid.UUID, targetUserID uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Str("target_user_id", targetUserID.String()).Msg("kick player rejected: user is not owner")
		return ErrForbidden
	}
	if targetUserID == game.gameOwner {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Str("target_user_id", targetUserID.String()).Msg("kick player rejected: cannot kick owner")
		return ErrCannotKickOwner
	}
	if _, isPlayer := game.players[targetUserID]; !isPlayer {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Str("target_user_id", targetUserID.String()).Msg("kick player rejected: target is not player")
		return ErrPlayerNotFound
	}

	game.removePlayer(targetUserID)
	if len(game.players) == 0 {
		delete(s.games, gameID)
		log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Str("target_user_id", targetUserID.String()).Msg("player kicked and empty game removed")
		return nil
	}
	game.ensureRoundMaster()
	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Str("target_user_id", targetUserID.String()).Int("players", len(game.players)).Msg("player kicked")

	return nil
}

func (s *Service) LeaveGame(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if _, isPlayer := game.players[user.userID]; !isPlayer {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Msg("leave game rejected: user is not player")
		return ErrForbidden
	}

	wasOwner := game.gameOwner == user.userID
	game.removePlayer(user.userID)
	if len(game.players) == 0 {
		delete(s.games, gameID)
		log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Msg("player left and empty game removed")
		return nil
	}
	if wasOwner {
		game.gameOwner = game.usersByPlace[0]
		log.Info().Str("game_id", gameID).Str("old_owner_user_id", user.userID.String()).Str("new_owner_user_id", game.gameOwner.String()).Msg("game owner transferred")
	}
	game.ensureRoundMaster()
	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("players", len(game.players)).Msg("player left game")

	return nil
}

func (s *Service) discardExpiredGameLocked(gameID string, now time.Time) bool {
	game := s.games[gameID]
	if game == nil {
		return false
	}
	if len(game.players) == 0 {
		delete(s.games, gameID)
		log.Info().Str("game_id", gameID).Msg("empty game discarded")
		return true
	}

	if !game.allPlayersOffline(now) {
		game.allOfflineSince = time.Time{}
		return false
	}

	if game.allOfflineSince.IsZero() {
		game.allOfflineSince = game.offlineSince(now)
	}
	if now.Sub(game.allOfflineSince) >= GameDiscardAfter {
		delete(s.games, gameID)
		log.Info().Str("game_id", gameID).Dur("offline_for", now.Sub(game.allOfflineSince)).Msg("offline game discarded")
		return true
	}

	return false
}

func (g *Games) onlineAt(userID uuid.UUID, now time.Time) bool {
	lastSeen, ok := g.lastSeenByUser[userID]
	return ok && now.Sub(lastSeen) <= PlayerOfflineAfter
}

func (g *Games) allPlayersOffline(now time.Time) bool {
	for userID := range g.players {
		if g.onlineAt(userID, now) {
			return false
		}
	}

	return true
}

func (g *Games) offlineSince(now time.Time) time.Time {
	var latest time.Time
	for userID := range g.players {
		if lastSeen := g.lastSeenByUser[userID]; lastSeen.After(latest) {
			latest = lastSeen
		}
	}
	if latest.IsZero() {
		return now
	}

	return latest.Add(PlayerOfflineAfter)
}

func (g *Games) removePlayer(userID uuid.UUID) {
	delete(g.players, userID)
	delete(g.lastSeenByUser, userID)
	delete(g.placeByUser, userID)

	usersByPlace := g.usersByPlace[:0]
	for _, existingUserID := range g.usersByPlace {
		if existingUserID != userID {
			usersByPlace = append(usersByPlace, existingUserID)
		}
	}
	g.usersByPlace = usersByPlace
	g.rebuildPlaces()

	for _, state := range g.rounds {
		answer := state.answersByUser[userID]
		if answer != nil {
			delete(state.answersByID, answer.answerID)
			state.scrambled = removeUUID(state.scrambled, answer.answerID)
		}
		delete(state.answersByUser, userID)
		delete(state.votesByUser, userID)
		if answer != nil {
			for voterID, answerID := range state.votesByUser {
				if answerID == answer.answerID {
					delete(state.votesByUser, voterID)
				}
			}
		}
	}
}

func (s *Service) leaveOtherGamesLocked(userID uuid.UUID, exceptGameID string) {
	for gameID, game := range s.games {
		if gameID == exceptGameID {
			continue
		}
		if _, isPlayer := game.players[userID]; !isPlayer {
			continue
		}

		wasOwner := game.gameOwner == userID
		game.removePlayer(userID)
		if len(game.players) == 0 {
			delete(s.games, gameID)
			continue
		}
		if wasOwner {
			game.gameOwner = game.usersByPlace[0]
			log.Info().Str("game_id", gameID).Str("old_owner_user_id", userID.String()).Str("new_owner_user_id", game.gameOwner.String()).Msg("game owner transferred after joining another game")
		}
		game.ensureRoundMaster()
		log.Info().Str("game_id", gameID).Str("user_id", userID.String()).Int("players", len(game.players)).Msg("player removed after joining another game")
	}
}

func (g *Games) rebuildPlaces() {
	g.placeByUser = make(map[uuid.UUID]int, len(g.usersByPlace))
	for index, userID := range g.usersByPlace {
		g.placeByUser[userID] = index + 1
	}
}

func (g *Games) ensureRoundMaster() {
	if g.currentRound == 0 || len(g.usersByPlace) == 0 {
		return
	}
	state := g.currentRoundState()
	if _, exists := g.players[state.roundMasterID]; exists {
		return
	}

	state.roundMasterID = g.usersByPlace[(g.currentRound-1)%len(g.usersByPlace)]
	log.Info().Str("game_id", g.gameID).Int("round", g.currentRound).Str("round_master_user_id", state.roundMasterID.String()).Msg("round master changed")
}

func removeUUID(values []uuid.UUID, value uuid.UUID) []uuid.UUID {
	filtered := values[:0]
	for _, existing := range values {
		if existing != value {
			filtered = append(filtered, existing)
		}
	}

	return filtered
}
