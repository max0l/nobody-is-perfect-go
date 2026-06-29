package api

const (
	SessionCookieName     = "session"
	SessionCookieValueKey = "sessionCookie.Value"
	SessionCookieMaxAge   = 60 * 60 * 24 * 30

	BadRequestError       = "bad request, missing required fields"
	UnauthorizedError     = "unauthorized, invalid or missing token"
	ForbiddenError        = "forbidden, you do not have permission to access this resource"
	GameNotFoundError     = "game not found, please check the game ID"
	NotEnoughPlayersError = "at least 3 players are required to start a game"
)
