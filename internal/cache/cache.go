package cache

import (
	"context"
	"secretSanta/internal/storage/models"
)

type Cache interface {
	AddUser(ctx context.Context, userID int, value models.CacheNote) error
	User(ctx context.Context, userID int) (models.CacheNote, error)
	UpdateState(ctx context.Context, userID int, state string) error
	RoomWhereUserIsOrg(ctx context.Context, userID int) (int, error)
}
