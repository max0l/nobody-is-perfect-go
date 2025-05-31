//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config server.cfg.yaml ../api.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config types.cfg.yaml ../api.yaml

package api

import (
	"github.com/gin-gonic/gin"
	"github.com/max0l/nobody-is-perfect-go/game"
	"net/http"
)

type Server struct {
	gameService *game.Service
}

func NewServer() *Server {
	return &Server{
		gameService: game.NewService(),
	}
}

func (s Server) PostApiCreateUser(ctx *gin.Context) {
	var body CreateUserRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Status(http.StatusBadRequest)
		ctx.Error(err)
		return
	}

	userToken, userUUID = s.gameService.CreateUser(body.Username)

	ctx.JSON(http.StatusOK, UserToken{
		UserToken:    userToken.String(),
		UserUUID: userUUID.String(),
	}
}
