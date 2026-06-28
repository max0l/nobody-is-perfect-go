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

	token, ok := ctx.Value(SessionCookieValueKey).(string)
	if !ok || token == "" {
		msg := UnauthorizedError
		return CreateGame401JSONResponse{Error: &msg}, nil
	}
	log.Info().Str("token", token).Msg("create game for user")

	userUuid, err := uuid.Parse(token)

	if err != nil {
		log.Error().Err(err).Msg("parse user UUID")
		msg := UnauthorizedError
		return CreateGame401JSONResponse{Error: &msg}, nil
	}

	gameId, err := s.gameService.CreateGame(userUuid)

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
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) SendAnswer(ctx context.Context, request SendAnswerRequestObject) (SendAnswerResponseObject, error) {
	//TODO implement me
	panic("implement me")
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
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) RevealVotes(ctx context.Context, request RevealVotesRequestObject) (RevealVotesResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) StartGame(ctx context.Context, request StartGameRequestObject) (StartGameResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) GetGameStatus(ctx context.Context, request GetGameStatusRequestObject) (GetGameStatusResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) VoteForAnswer(ctx context.Context, request VoteForAnswerRequestObject) (VoteForAnswerResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) HealthCheck(ctx context.Context, request HealthCheckRequestObject) (HealthCheckResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) JoinGame(ctx context.Context, request JoinGameRequestObject) (JoinGameResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) GetPlayOrder(ctx context.Context, request GetPlayOrderRequestObject) (GetPlayOrderResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (s *StrictServer) SetPlayOrder(ctx context.Context, request SetPlayOrderRequestObject) (SetPlayOrderResponseObject, error) {
	//TODO implement me
	panic("implement me")
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
