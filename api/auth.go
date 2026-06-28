package api

const (
	SessionCookieValueKey = "sessionCookie.Value"

	BadRequestError   = "bad request, missing required fields"
	UnauthorizedError = "unauthorized, invalid or missing token"
	ForbiddenError    = "forbidden, you do not have permission to access this resource"
	GameNotFoundError = "game not found, please check the game ID"
)
