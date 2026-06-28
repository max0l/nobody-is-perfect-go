package game

import "github.com/google/uuid"

const maxScrambledAnswers = 4

var answerLabels = []string{"A", "B", "C", "D"}

type AnswerView struct {
	Label    string
	AnswerID uuid.UUID
	Answer   string
	UserUUID uuid.UUID
	Username string
}

func (s *Service) SendAnswer(gameID string, token uuid.UUID, answerText *string) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	if answerText == nil || *answerText == "" {
		return ErrAnswerRequired
	}

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if _, isPlayer := game.players[user.userID]; !isPlayer {
		return ErrForbidden
	}

	state := game.currentRoundState()
	answer := state.answersByUser[user.userID]
	if answer == nil {
		answer = &Answer{
			answerID: uuid.New(),
			userID:   user.userID,
		}
		state.answersByUser[user.userID] = answer
		state.answersByID[answer.answerID] = answer
	}

	answer.answer = *answerText
	state.scrambled = nil

	return nil
}

func (s *Service) GetAnswers(gameID string, token uuid.UUID) ([]AnswerView, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return nil, err
	}
	if _, isPlayer := game.players[user.userID]; !isPlayer {
		return nil, ErrForbidden
	}

	state := game.currentRoundState()
	if state.scrambled == nil {
		state.scrambled = make([]uuid.UUID, 0, len(state.answersByID))
		for answerID := range state.answersByID {
			state.scrambled = append(state.scrambled, answerID)
		}
		s.random.Shuffle(len(state.scrambled), func(i, j int) {
			state.scrambled[i], state.scrambled[j] = state.scrambled[j], state.scrambled[i]
		})
	}

	includeAuthors := game.currentReaderID() == user.userID
	count := len(state.scrambled)
	if count > maxScrambledAnswers {
		count = maxScrambledAnswers
	}

	answers := make([]AnswerView, 0, count)
	for i := 0; i < count; i++ {
		answer := state.answersByID[state.scrambled[i]]
		if answer == nil {
			continue
		}

		view := AnswerView{
			Label:    answerLabels[i],
			AnswerID: answer.answerID,
			Answer:   answer.answer,
		}
		if includeAuthors {
			view.UserUUID = answer.userID
			if answerUser := game.players[answer.userID]; answerUser != nil {
				view.Username = answerUser.username
			}
		}

		answers = append(answers, view)
	}

	return answers, nil
}

func (s *Service) NextRound(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID {
		return ErrForbidden
	}

	game.currentRound++
	game.gameStatus = GameStatusStarted

	return nil
}
