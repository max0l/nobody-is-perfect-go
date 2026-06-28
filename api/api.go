//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config server.cfg.yaml ../api.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config types.cfg.yaml ../api.yaml

package api

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/max0l/nobody-is-perfect-go/game"
	"github.com/rs/zerolog/log"
)

type StrictServer struct {
	gameService *game.Service
}

func (s *StrictServer) CreateGame(ctx context.Context, request CreateGameRequestObject) (CreateGameResponseObject, error) {

	log.Info().Interface("request", request).Interface("context", ctx).Msg("create game request")

	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return CreateGame401JSONResponse{Error: &msg}, nil
	}
	log.Info().Str("token", token.String()).Msg("create game for user")

	gameId, err := s.gameService.CreateGame(token)

	if err != nil {
		log.Error().Err(err).Msg("create game")
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return CreateGame401JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return CreateGame400JSONResponse{Error: &msg}, nil
	}
	return CreateGame201JSONResponse{
		GameId: &gameId,
	}, nil
}

func (s *StrictServer) GetAnswers(ctx context.Context, request GetAnswersRequestObject) (GetAnswersResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return GetAnswers401JSONResponse{Error: &msg}, nil
	}

	answers, err := s.gameService.GetAnswers(request.GameId, token)
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return GetAnswers404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return GetAnswers401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return GetAnswers403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return GetAnswers400JSONResponse{Error: &msg}, nil
	}

	response := make([]Answer, 0, len(answers))
	for _, answer := range answers {
		label := answer.Label
		answerID := answer.AnswerID
		answerText := answer.Answer
		responseAnswer := Answer{
			Label:      &label,
			AnswerUUID: &answerID,
			Answer:     &answerText,
		}
		if answer.UserUUID != uuid.Nil {
			userID := answer.UserUUID
			username := answer.Username
			responseAnswer.UserUUID = &userID
			responseAnswer.Username = &username
		}

		response = append(response, responseAnswer)
	}

	return GetAnswers200JSONResponse{Answers: &response}, nil
}

func (s *StrictServer) SendAnswer(ctx context.Context, request SendAnswerRequestObject) (SendAnswerResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return SendAnswer401JSONResponse{Error: &msg}, nil
	}
	if request.Body == nil {
		msg := BadRequestError
		return SendAnswer400JSONResponse{Error: &msg}, nil
	}

	if err := s.gameService.SendAnswer(request.GameId, token, request.Body.Answer); err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return SendAnswer404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return SendAnswer401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return SendAnswer403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return SendAnswer400JSONResponse{Error: &msg}, nil
	}

	msg := "received the answer"
	return SendAnswer200JSONResponse{Message: &msg}, nil
}

func (s *StrictServer) SelectValidAnswers(ctx context.Context, request SelectValidAnswersRequestObject) (SelectValidAnswersResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) FinishGame(ctx context.Context, request FinishGameRequestObject) (FinishGameResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) NextRound(ctx context.Context, request NextRoundRequestObject) (NextRoundResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return NextRound401JSONResponse{Error: &msg}, nil
	}

	if err := s.gameService.NextRound(request.GameId, token); err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return NextRound404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return NextRound401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return NextRound403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return NextRound400JSONResponse{Error: &msg}, nil
	}

	msg := "moved to the next round successfully"
	return NextRound200JSONResponse{Message: &msg}, nil
}

func (s *StrictServer) RevealVotes(ctx context.Context, request RevealVotesRequestObject) (RevealVotesResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return RevealVotes401JSONResponse{Error: &msg}, nil
	}

	answers, err := s.gameService.GetRevealedVotes(request.GameId, token)
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return RevealVotes404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return RevealVotes401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return RevealVotes403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return RevealVotes400JSONResponse{Error: &msg}, nil
	}

	response := revealedAnswersResponse(answers)
	return RevealVotes200JSONResponse{Answers: &response}, nil
}

func (s *StrictServer) StartGame(ctx context.Context, request StartGameRequestObject) (StartGameResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return StartGame401JSONResponse{Error: &msg}, nil
	}

	if err := s.gameService.StartGame(request.GameId, token); err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return StartGame404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return StartGame401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return StartGame403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return StartGame400JSONResponse{Error: &msg}, nil
	}

	msg := "game started successfully"
	return StartGame200JSONResponse{Message: &msg}, nil
}

func (s *StrictServer) GetGameStatus(ctx context.Context, request GetGameStatusRequestObject) (GetGameStatusResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return GetGameStatus401JSONResponse{Error: &msg}, nil
	}

	status, err := s.gameService.GetStatus(request.GameId, token)
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return GetGameStatus404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return GetGameStatus401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return GetGameStatus403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return GetGameStatus400JSONResponse{Error: &msg}, nil
	}

	users := make([]User, 0, len(status.Players))
	for _, player := range status.Players {
		userID := player.UserUUID
		username := player.Username
		users = append(users, User{UserUUID: &userID, Username: &username})
	}
	gameStatus := int(status.GameStatus)
	receivedAnswers := status.ReceivedAnswers
	playerCount := status.PlayerCount
	round := status.Round
	gameOwner := status.GameOwnerUUID
	response := GetGameStatus200JSONResponse{
		Status:          &gameStatus,
		Users:           &users,
		GameMasterUUID:  &gameOwner,
		ReceivedAnswers: &receivedAnswers,
		PlayerCount:     &playerCount,
		Round:           &round,
	}
	if status.Round > 0 {
		roundStatus := status.RoundStatus.String()
		roundMaster := status.RoundMasterUUID
		response.RoundStatus = &roundStatus
		response.RoundMasterUUID = &roundMaster
	}

	return response, nil
}

func (s *StrictServer) VoteForAnswer(ctx context.Context, request VoteForAnswerRequestObject) (VoteForAnswerResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return VoteForAnswer401JSONResponse{Error: &msg}, nil
	}
	if request.Body == nil || request.Body.AnswerUUID == nil {
		msg := BadRequestError
		return VoteForAnswer400JSONResponse{Error: &msg}, nil
	}

	if err := s.gameService.VoteForAnswer(request.GameId, token, *request.Body.AnswerUUID); err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return VoteForAnswer404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return VoteForAnswer401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return VoteForAnswer403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return VoteForAnswer400JSONResponse{Error: &msg}, nil
	}

	msg := "vote recorded successfully"
	return VoteForAnswer200JSONResponse{Message: &msg}, nil
}

func (s *StrictServer) HealthCheck(ctx context.Context, request HealthCheckRequestObject) (HealthCheckResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) JoinGame(ctx context.Context, request JoinGameRequestObject) (JoinGameResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return JoinGame401JSONResponse{Error: &msg}, nil
	}

	if err := s.gameService.JoinGame(request.GameId, token); err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return JoinGame404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return JoinGame401JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return JoinGame400JSONResponse{Error: &msg}, nil
	}

	msg := "user joined game"
	return JoinGame200JSONResponse{Message: &msg}, nil
}

func (s *StrictServer) GetPlayOrder(ctx context.Context, request GetPlayOrderRequestObject) (GetPlayOrderResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return GetPlayOrder401JSONResponse{Error: &msg}, nil
	}

	playOrder, err := s.gameService.GetPlayOrder(request.GameId, token)
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return GetPlayOrder404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return GetPlayOrder401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return GetPlayOrder403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return GetPlayOrder400JSONResponse{Error: &msg}, nil
	}

	response := make([]PlayOrderUser, 0, len(playOrder))
	for _, entry := range playOrder {
		place := entry.Place
		userID := entry.UserUUID
		username := entry.Username
		response = append(response, PlayOrderUser{
			Place:    &place,
			UserUUID: &userID,
			Username: &username,
		})
	}

	return GetPlayOrder200JSONResponse{PlayOrder: &response}, nil
}

func (s *StrictServer) SetPlayOrder(ctx context.Context, request SetPlayOrderRequestObject) (SetPlayOrderResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return SetPlayOrder401JSONResponse{Error: &msg}, nil
	}
	if request.Body == nil || request.Body.PlayOrder == nil {
		msg := BadRequestError
		return SetPlayOrder400JSONResponse{Error: &msg}, nil
	}

	entries := make([]game.SetPlayOrderEntry, 0, len(*request.Body.PlayOrder))
	for _, entry := range *request.Body.PlayOrder {
		if entry.Place == nil || entry.UserUUID == nil {
			msg := BadRequestError
			return SetPlayOrder400JSONResponse{Error: &msg}, nil
		}

		entries = append(entries, game.SetPlayOrderEntry{
			Place:    *entry.Place,
			UserUUID: *entry.UserUUID,
		})
	}

	if err := s.gameService.SetPlayOrder(request.GameId, token, entries); err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return SetPlayOrder404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return SetPlayOrder401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return SetPlayOrder403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return SetPlayOrder400JSONResponse{Error: &msg}, nil
	}

	msg := "play order set"
	return SetPlayOrder200JSONResponse{Message: &msg}, nil
}

func (s *StrictServer) TriggerReveal(ctx context.Context, request TriggerRevealRequestObject) (TriggerRevealResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return TriggerReveal401JSONResponse{Error: &msg}, nil
	}

	answers, err := s.gameService.RevealRound(request.GameId, token)
	if err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return TriggerReveal404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return TriggerReveal401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return TriggerReveal403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return TriggerReveal400JSONResponse{Error: &msg}, nil
	}

	response := revealedAnswersResponse(answers)
	return TriggerReveal200JSONResponse{Answers: &response}, nil
}

func (s *StrictServer) StartVoting(ctx context.Context, request StartVotingRequestObject) (StartVotingResponseObject, error) {
	token, ok := sessionToken(ctx)
	if !ok {
		msg := UnauthorizedError
		return StartVoting401JSONResponse{Error: &msg}, nil
	}

	if err := s.gameService.StartVoting(request.GameId, token); err != nil {
		if errors.Is(err, game.ErrGameNotFound) {
			msg := GameNotFoundError
			return StartVoting404JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrUserNotFound) {
			msg := UnauthorizedError
			return StartVoting401JSONResponse{Error: &msg}, nil
		}
		if errors.Is(err, game.ErrForbidden) {
			msg := ForbiddenError
			return StartVoting403JSONResponse{Error: &msg}, nil
		}

		msg := BadRequestError
		return StartVoting400JSONResponse{Error: &msg}, nil
	}

	msg := "voting started successfully"
	return StartVoting200JSONResponse{Message: &msg}, nil
}

func NewServer() *StrictServer {
	return &StrictServer{
		gameService: game.NewService(),
	}
}

func (s *StrictServer) CreateUser(ctx context.Context, request CreateUserRequestObject) (CreateUserResponseObject, error) {
	user, err := s.gameService.CreateUser(request.Body.Username)

	if err == nil {
		return CreateUser201JSONResponse{
			UserToken: user.GetUserToken(),
			UserUUID:  user.GetUserID(),
		}, err
	}

	log.Error().Err(err).Msg("create user")
	if errors.Is(err, game.ErrUsernameRequired) {
		msg := BadRequestError
		return CreateUser400JSONResponse{Error: &msg}, nil
	}

	msg := ForbiddenError
	return CreateUser403JSONResponse{Error: &msg}, nil
}

func sessionToken(ctx context.Context) (uuid.UUID, bool) {
	token, ok := ctx.Value(SessionCookieValueKey).(string)
	if !ok || token == "" {
		return uuid.Nil, false
	}

	parsed, err := uuid.Parse(token)
	if err != nil {
		log.Error().Err(err).Msg("parse session token")
		return uuid.Nil, false
	}

	return parsed, true
}

func revealedAnswersResponse(answers []game.RevealedAnswerView) []AnswerWithVotes {
	response := make([]AnswerWithVotes, 0, len(answers))
	for _, answer := range answers {
		label := answer.Label
		answerID := answer.AnswerID
		answerText := answer.Answer
		userID := answer.UserUUID
		username := answer.Username
		votes := make([]VoteUser, 0, len(answer.Votes))
		for _, vote := range answer.Votes {
			voterID := vote.UserUUID
			voterName := vote.Username
			votes = append(votes, VoteUser{UserUUID: &voterID, Username: &voterName})
		}

		response = append(response, AnswerWithVotes{
			Label:      &label,
			AnswerUUID: &answerID,
			Answer:     &answerText,
			UserUUID:   &userID,
			Username:   &username,
			Votes:      &votes,
		})
	}

	return response
}
