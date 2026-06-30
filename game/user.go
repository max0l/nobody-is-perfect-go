package game

import (
	"unicode/utf8"

	"github.com/google/uuid"
)

type User struct {
	token    uuid.UUID
	username string
	userID   uuid.UUID
}

func (u *User) GetUserToken() *uuid.UUID {
	return &u.token
}

func (u *User) GetUserUsername() *string {
	return &u.username
}

func (u *User) GetUserID() *uuid.UUID {
	return &u.userID
}

func (s *Service) CreateUser(username *string) (*User, error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	if username == nil || *username == "" {
		return nil, ErrUsernameRequired
	}
	if utf8.RuneCountInString(*username) > MaxUsernameLength {
		return nil, ErrUsernameTooLong
	}

	userID := uuid.New()
	token := uuid.New()

	user := &User{
		token:    token,
		username: *username,
		userID:   userID,
	}

	s.users[token] = user
	s.usersByID[userID] = user

	return user, nil
}
