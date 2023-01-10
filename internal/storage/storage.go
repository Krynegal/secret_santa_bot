package storage

import (
	"errors"
	"secretSanta/internal/configs"
	"secretSanta/internal/storage/models"
)

var (
	RoomUserPairExists = errors.New("room with this user is already exist")
	UserIsNotOrg       = errors.New("there are no rooms where user is an organizer")
)

type Storager interface {
	AddUser(user models.User) error
	CreateRoom(roomID int) error
	AssignRoomToUser(roomID, userID int, isOrganizer bool) error
	UsersFromRoom(roomID int) ([]models.User, error)
	RoomWhereUserIsOrg(userID int) (int, error)
	AddWish(roomID, userID int, wish string) error
	Wish(roomID, userID int) (string, error)
}

func NewStorage(cfg *configs.Config) (Storager, error) {
	db, err := NewDatabaseStorage(cfg.DB)
	if err != nil {
		return nil, err
	}
	return db, nil
}
