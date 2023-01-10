package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v9"
	"secretSanta/internal/configs"
	"secretSanta/internal/storage/models"
	"strconv"
	"time"
)

type cache struct {
	client *redis.Client
}

func dial(cfg *configs.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        cfg.CacheAddr,
		DB:          0,
		DialTimeout: 600 * time.Second,
		ReadTimeout: 600 * time.Second,
	})
}

func New(cfg *configs.Config) (*cache, error) {
	client := dial(cfg)
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}
	return &cache{
		client: client,
	}, nil
}

func (c *cache) AddUser(ctx context.Context, userID int, value models.CacheNote) error {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(value); err != nil {
		return err
	}
	uid := strconv.Itoa(userID)
	return c.client.Set(ctx, uid, buffer.Bytes(), 0).Err()
}

func (c *cache) User(ctx context.Context, userID int) (models.CacheNote, error) {
	uid := strconv.Itoa(userID)
	cmd := c.client.Get(ctx, uid)
	cmdb, err := cmd.Bytes()
	if err != nil {
		return models.CacheNote{}, err
	}
	b := bytes.NewReader(cmdb)
	var res models.CacheNote
	if err = json.NewDecoder(b).Decode(&res); err != nil {
		return models.CacheNote{}, err
	}
	return res, nil
}

func (c *cache) UpdateState(ctx context.Context, userID int, state string) error {
	uid := strconv.Itoa(userID)
	user, err := c.User(ctx, userID)
	if err != nil {
		return err
	}
	user.State = state
	var buffer bytes.Buffer
	if err = json.NewEncoder(&buffer).Encode(user); err != nil {
		return err
	}
	if err = c.client.Set(ctx, uid, buffer.Bytes(), 0).Err(); err != nil {
		return err
	}
	return nil
}

func (c *cache) RoomWhereUserIsOrg(ctx context.Context, userID int) (int, error) {
	uid := strconv.Itoa(userID)
	cmd := c.client.Get(ctx, uid)

	cmdb, err := cmd.Bytes()
	if err != nil {
		return -1, err
	}

	b := bytes.NewReader(cmdb)
	var res models.CacheNote
	if err = json.NewDecoder(b).Decode(&res); err != nil {
		return -1, err
	}
	return res.RoomID, nil
}
