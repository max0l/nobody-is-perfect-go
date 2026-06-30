package game

import (
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type StatusView struct {
	GameStatus      GameStatus
	Players         []PlayOrderEntry
	GameOwnerUUID   uuid.UUID
	ReceivedAnswers int
	ReceivedVotes   int
	CurrentAnswer   string
	CurrentAnswerID uuid.UUID
	PlayerCount     int
	Round           int
	RoundStatus     RoundStatus
	RoundMasterUUID uuid.UUID
}

func (s *Service) StartGame(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Msg("start game rejected: user is not owner")
		return ErrForbidden
	}
	if game.gameStatus != GameStatusCreated || len(game.usersByPlace) == 0 {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("game_status", int(game.gameStatus)).Msg("start game rejected: invalid game state")
		return ErrInvalidRound
	}
	if len(game.players) < MinPlayersToStart {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("players", len(game.players)).Int("min_players", MinPlayersToStart).Msg("start game rejected: not enough players")
		return ErrNotEnoughPlayers
	}

	game.gameStatus = GameStatusStarted
	game.startNextRound()
	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("players", len(game.players)).Msg("game started")

	return nil
}

func (s *Service) FinishGame(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Msg("finish game rejected: user is not owner")
		return ErrForbidden
	}

	delete(s.games, gameID)
	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("active_games", len(s.games)).Msg("game finished")

	return nil
}

func (s *Service) StartVoting(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	state, err := game.activeRoundState()
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID && state.roundMasterID != user.userID {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Msg("start voting rejected: user is not owner or round master")
		return ErrForbidden
	}
	if state.status != RoundStatusVerifying {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Str("round_status", state.status.String()).Msg("start voting rejected: invalid round status")
		return ErrInvalidRound
	}
	if game.gameOwner != user.userID && len(game.players) < MinPlayersToStart {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("players", len(game.players)).Int("min_players", MinPlayersToStart).Msg("start voting rejected: not enough players")
		return ErrInvalidRound
	}

	state.status = RoundStatusVoting
	ensureScrambled(s, state)
	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Int("answers", len(state.answersByUser)).Msg("voting started")

	return nil
}

func (s *Service) StartVerification(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	state, err := game.activeRoundState()
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID && state.roundMasterID != user.userID {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Msg("start verification rejected: user is not owner or round master")
		return ErrForbidden
	}
	if state.status != RoundStatusAnswering {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Str("round_status", state.status.String()).Msg("start verification rejected: invalid round status")
		return ErrInvalidRound
	}
	if game.gameOwner != user.userID && len(game.players) < MinPlayersToStart {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("players", len(game.players)).Int("min_players", MinPlayersToStart).Msg("start verification rejected: not enough players")
		return ErrInvalidRound
	}
	if game.gameOwner != user.userID && len(state.answersByUser) < len(game.players) {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("answers", len(state.answersByUser)).Int("players", len(game.players)).Msg("start verification rejected: missing answers")
		return ErrInvalidRound
	}

	state.status = RoundStatusVerifying
	ensureScrambled(s, state)
	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Int("answers", len(state.answersByUser)).Msg("verification started")

	return nil
}

func (s *Service) RevealRound(gameID string, token uuid.UUID) ([]RevealedAnswerView, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return nil, err
	}
	state, err := game.activeRoundState()
	if err != nil {
		return nil, err
	}
	if game.gameOwner != user.userID && state.roundMasterID != user.userID {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Msg("reveal round rejected: user is not owner or round master")
		return nil, ErrForbidden
	}
	if state.status != RoundStatusVoting && state.status != RoundStatusRevealed {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Str("round_status", state.status.String()).Msg("reveal round rejected: invalid round status")
		return nil, ErrInvalidRound
	}
	if state.status == RoundStatusVoting && game.gameOwner != user.userID && len(state.votesByUser) < game.requiredVoteCount(state) {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("votes", len(state.votesByUser)).Int("required_votes", game.requiredVoteCount(state)).Msg("reveal round rejected: missing votes")
		return nil, ErrInvalidRound
	}

	state.status = RoundStatusRevealed
	log.Info().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Int("votes", len(state.votesByUser)).Msg("round revealed")
	return game.revealedAnswers(state), nil
}

func (s *Service) GetRevealedVotes(gameID string, token uuid.UUID) ([]RevealedAnswerView, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return nil, err
	}
	if _, isPlayer := game.players[user.userID]; !isPlayer {
		return nil, ErrForbidden
	}
	state, err := game.activeRoundState()
	if err != nil {
		return nil, err
	}
	if state.status != RoundStatusRevealed {
		return nil, ErrInvalidRound
	}

	return game.revealedAnswers(state), nil
}

func (s *Service) NextRound(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	state, err := game.activeRoundState()
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID && state.roundMasterID != user.userID {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Msg("next round rejected: user is not owner or round master")
		return ErrForbidden
	}
	if state.status != RoundStatusRevealed {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("round", game.currentRound).Str("round_status", state.status.String()).Msg("next round rejected: invalid round status")
		return ErrInvalidRound
	}
	if game.gameOwner != user.userID && len(game.players) < MinPlayersToStart {
		log.Warn().Str("game_id", gameID).Str("user_id", user.userID.String()).Int("players", len(game.players)).Int("min_players", MinPlayersToStart).Msg("next round rejected: not enough players")
		return ErrInvalidRound
	}

	game.startNextRound()

	return nil
}

func (s *Service) GetStatus(gameID string, token uuid.UUID) (StatusView, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return StatusView{}, err
	}
	if _, isPlayer := game.players[user.userID]; !isPlayer {
		return StatusView{}, ErrForbidden
	}
	now := s.now()
	game.lastSeenByUser[user.userID] = now
	game.allOfflineSince = time.Time{}

	view := StatusView{
		GameStatus:    game.gameStatus,
		Players:       game.playOrderEntries(game.players, now),
		GameOwnerUUID: game.gameOwner,
		PlayerCount:   len(game.players),
		Round:         game.currentRound,
	}
	if game.currentRound > 0 {
		state := game.currentRoundState()
		view.RoundStatus = state.status
		view.RoundMasterUUID = state.roundMasterID
		view.ReceivedAnswers = len(state.answersByUser)
		view.ReceivedVotes = len(state.votesByUser)
		if answer := state.answersByUser[user.userID]; answer != nil {
			view.CurrentAnswer = answer.answer
			view.CurrentAnswerID = answer.answerID
		}
	}

	return view, nil
}

func (g *Games) activeRoundState() (*round, error) {
	if g.gameStatus != GameStatusStarted || g.currentRound == 0 {
		return nil, ErrInvalidRound
	}

	return g.currentRoundState(), nil
}

func (g *Games) requiredVoteCount(state *round) int {
	count := len(g.players)
	if _, hasRoundMaster := g.players[state.roundMasterID]; hasRoundMaster {
		count--
	}

	return count
}
