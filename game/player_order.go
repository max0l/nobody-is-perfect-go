package game

import "github.com/google/uuid"

type PlayOrderEntry struct {
	Place    int
	UserUUID uuid.UUID
	Username string
}

type SetPlayOrderEntry struct {
	Place    int
	UserUUID uuid.UUID
}

func (s *Service) JoinGame(gameID string, token uuid.UUID) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, exists := s.games[gameID]
	if !exists {
		return ErrGameNotFound
	}

	user, exists := s.users[token]
	if !exists {
		return ErrUserNotFound
	}

	if _, alreadyJoined := game.players[user.userID]; alreadyJoined {
		return nil
	}

	game.players[user.userID] = user
	game.appendPlayer(user.userID)

	return nil
}

func (s *Service) GetPlayOrder(gameID string, token uuid.UUID) ([]PlayOrderEntry, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return nil, err
	}
	if _, isPlayer := game.players[user.userID]; !isPlayer {
		return nil, ErrForbidden
	}

	return game.playOrderEntries(game.players), nil
}

func (s *Service) SetPlayOrder(gameID string, token uuid.UUID, entries []SetPlayOrderEntry) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	game, user, err := s.gameAndUser(gameID, token)
	if err != nil {
		return err
	}
	if game.gameOwner != user.userID {
		return ErrForbidden
	}

	return game.setPlayOrder(entries)
}

func (s *Service) gameAndUser(gameID string, token uuid.UUID) (*Games, *User, error) {
	game, exists := s.games[gameID]
	if !exists {
		return nil, nil, ErrGameNotFound
	}

	user, exists := s.users[token]
	if !exists {
		return nil, nil, ErrUserNotFound
	}

	return game, user, nil
}

func (g *Games) appendPlayer(userID uuid.UUID) {
	if _, exists := g.placeByUser[userID]; exists {
		return
	}

	g.usersByPlace = append(g.usersByPlace, userID)
	g.placeByUser[userID] = len(g.usersByPlace)
}

func (g *Games) playOrderEntries(players map[uuid.UUID]*User) []PlayOrderEntry {
	entries := make([]PlayOrderEntry, 0, len(g.usersByPlace))
	for place, userID := range g.usersByPlace {
		user := players[userID]
		if user == nil {
			continue
		}

		entries = append(entries, PlayOrderEntry{
			Place:    place + 1,
			UserUUID: user.userID,
			Username: user.username,
		})
	}

	return entries
}

func (g *Games) setPlayOrder(entries []SetPlayOrderEntry) error {
	if len(entries) != len(g.players) {
		return ErrInvalidPlayOrder
	}

	usersByPlace := make([]uuid.UUID, len(entries))
	placeByUser := make(map[uuid.UUID]int, len(entries))
	for _, entry := range entries {
		if entry.Place < 1 || entry.Place > len(entries) {
			return ErrInvalidPlayOrder
		}
		if _, isPlayer := g.players[entry.UserUUID]; !isPlayer {
			return ErrInvalidPlayOrder
		}
		if usersByPlace[entry.Place-1] != uuid.Nil {
			return ErrInvalidPlayOrder
		}
		if _, duplicateUser := placeByUser[entry.UserUUID]; duplicateUser {
			return ErrInvalidPlayOrder
		}

		usersByPlace[entry.Place-1] = entry.UserUUID
		placeByUser[entry.UserUUID] = entry.Place
	}

	g.usersByPlace = usersByPlace
	g.placeByUser = placeByUser

	return nil
}
