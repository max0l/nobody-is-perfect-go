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

type RoundStatus int

const (
	RoundStatusAnswering RoundStatus = iota
	RoundStatusVerifying
	RoundStatusVoting
	RoundStatusRevealed
)

func (s RoundStatus) String() string {
	switch s {
	case RoundStatusAnswering:
		return "answering"
	case RoundStatusVerifying:
		return "verifying"
	case RoundStatusVoting:
		return "voting"
	case RoundStatusRevealed:
		return "revealed"
	default:
		return "unknown"
	}
}

func (s RoundStatus) IsValid() bool {
	return s >= RoundStatusAnswering && s <= RoundStatusRevealed
}
