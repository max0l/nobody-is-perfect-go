package game

import "github.com/google/uuid"

const maxScrambledAnswers = 6

var answerLabels = []string{"A", "B", "C", "D", "E", "F"}

type AnswerView struct {
	Label    string
	AnswerID uuid.UUID
	Answer   string
	UserUUID uuid.UUID
	Username string
}

type VoteView struct {
	UserUUID uuid.UUID
	Username string
}

type RevealedAnswerView struct {
	AnswerView
	Votes []VoteView
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
	state, err := game.activeRoundState()
	if err != nil {
		return err
	}
	if state.status != RoundStatusAnswering {
		return ErrInvalidRound
	}

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
	state, err := game.activeRoundState()
	if err != nil {
		return nil, err
	}
	if state.status == RoundStatusAnswering && state.roundMasterID != user.userID {
		return nil, ErrForbidden
	}

	ensureScrambled(s, state)

	includeAuthors := state.roundMasterID == user.userID
	count := displayedAnswerCount(state)

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

func (s *Service) VoteForAnswer(gameID string, token uuid.UUID, answerID uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if _, isPlayer := game.players[user.userID]; !isPlayer {
		return ErrForbidden
	}
	state, err := game.activeRoundState()
	if err != nil {
		return err
	}
	if state.status != RoundStatusVoting {
		return ErrInvalidRound
	}
	if state.roundMasterID == user.userID {
		return ErrForbidden
	}
	if !state.isDisplayedAnswer(answerID) {
		return ErrAnswerNotFound
	}

	state.votesByUser[user.userID] = answerID

	return nil
}

func ensureScrambled(s *Service, state *round) {
	if state.scrambled != nil {
		return
	}

	state.scrambled = make([]uuid.UUID, 0, len(state.answersByID))
	for answerID := range state.answersByID {
		state.scrambled = append(state.scrambled, answerID)
	}
	s.random.Shuffle(len(state.scrambled), func(i, j int) {
		state.scrambled[i], state.scrambled[j] = state.scrambled[j], state.scrambled[i]
	})
}

func displayedAnswerCount(state *round) int {
	count := len(state.scrambled)
	if count > maxScrambledAnswers {
		count = maxScrambledAnswers
	}

	return count
}

func (state *round) isDisplayedAnswer(answerID uuid.UUID) bool {
	for i := 0; i < displayedAnswerCount(state); i++ {
		if state.scrambled[i] == answerID {
			return true
		}
	}

	return false
}

func (g *Games) revealedAnswers(state *round) []RevealedAnswerView {
	answers := make([]RevealedAnswerView, 0, displayedAnswerCount(state))
	for i := 0; i < displayedAnswerCount(state); i++ {
		answer := state.answersByID[state.scrambled[i]]
		if answer == nil {
			continue
		}

		view := RevealedAnswerView{
			AnswerView: AnswerView{
				Label:    answerLabels[i],
				AnswerID: answer.answerID,
				Answer:   answer.answer,
				UserUUID: answer.userID,
			},
		}
		if answerUser := g.players[answer.userID]; answerUser != nil {
			view.Username = answerUser.username
		}

		for voterID, votedAnswerID := range state.votesByUser {
			if votedAnswerID != answer.answerID {
				continue
			}
			voter := g.players[voterID]
			if voter == nil {
				continue
			}

			view.Votes = append(view.Votes, VoteView{
				UserUUID: voter.userID,
				Username: voter.username,
			})
		}

		answers = append(answers, view)
	}

	return answers
}
