package storage

import (
	"errors"
	"secretSanta/internal/configs"
	"secretSanta/internal/storage/models"
)

var (
	RoomUserPairExists = errors.New("Room with this user is already exist")
)

type Storager interface {
	AddUser(user models.User) error
	CreateRoom(roomID int) error
	AssignRoomToUser(roomID, userID int, isOrganizer bool) error
	UsersFromRoom(roomID int) ([]models.User, error)
	RoomsWhereUserIsOrg(userID int) ([]int, error)
}

func NewStorage(cfg *configs.Config) (Storager, error) {
	db, err := NewDatabaseStorage(cfg.DB)
	if err != nil {
		return nil, err
	}
	return db, nil
}
