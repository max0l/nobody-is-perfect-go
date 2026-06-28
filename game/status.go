package game

type GameStatus int

const (
	GameStatusCreated GameStatus = iota
	GameStatusStarted
	GameStatusFinished
)

func (s GameStatus) String() string {
	switch s {
	case GameStatusCreated:
		return "created"
	case GameStatusStarted:
		return "started"
	case GameStatusFinished:
		return "finished"
	default:
		return "unknown"
	}
}

func (s GameStatus) IsValid() bool {
	return s >= GameStatusCreated && s <= GameStatusFinished
}
