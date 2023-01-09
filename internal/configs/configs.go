package configs

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	BotToken  string
	CacheAddr string
	DB        string
}

func NewConfig() *Config {
	cfg := &Config{}
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
	cfg.BotToken = os.Getenv("TOKEN")
	cfg.CacheAddr = os.Getenv("CACHE_ADDRESS")
	cfg.DB = os.Getenv("DATA_SOURCE_NAME")
	log.Printf("configs: %v", *cfg)
	return cfg
}
