package game

import "github.com/google/uuid"

type StatusView struct {
	GameStatus      GameStatus
	Players         []PlayOrderEntry
	GameOwnerUUID   uuid.UUID
	ReceivedAnswers int
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
		return ErrForbidden
	}
	if game.gameStatus != GameStatusCreated || len(game.usersByPlace) == 0 {
		return ErrInvalidRound
	}

	game.gameStatus = GameStatusStarted
	game.startNextRound()

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
		return ErrForbidden
	}
	if state.status != RoundStatusAnswering {
		return ErrInvalidRound
	}

	state.status = RoundStatusVoting
	ensureScrambled(s, state)

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
		return nil, ErrForbidden
	}
	if state.status != RoundStatusVoting && state.status != RoundStatusRevealed {
		return nil, ErrInvalidRound
	}

	state.status = RoundStatusRevealed
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
		return ErrForbidden
	}
	if state.status != RoundStatusRevealed {
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

	view := StatusView{
		GameStatus:    game.gameStatus,
		Players:       game.playOrderEntries(game.players, s.now()),
		GameOwnerUUID: game.gameOwner,
		PlayerCount:   len(game.players),
		Round:         game.currentRound,
	}
	if game.currentRound > 0 {
		state := game.currentRoundState()
		view.RoundStatus = state.status
		view.RoundMasterUUID = state.roundMasterID
		view.ReceivedAnswers = len(state.answersByUser)
	}

	return view, nil
}

func (g *Games) activeRoundState() (*round, error) {
	if g.gameStatus != GameStatusStarted || g.currentRound == 0 {
		return nil, ErrInvalidRound
	}

	return g.currentRoundState(), nil
}
