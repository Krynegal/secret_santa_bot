package configs

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	BotToken string
	DB       string
}

func NewConfig() *Config {
	cfg := &Config{}
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
	cfg.BotToken = os.Getenv("TOKEN")
	cfg.DB = os.Getenv("DATA_SOURCE_NAME")
	log.Printf("configs: %v", *cfg)
	return cfg
}
