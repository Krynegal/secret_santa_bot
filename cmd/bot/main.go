package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"secretSanta/internal/configs"
	"secretSanta/internal/storage"
	"secretSanta/internal/telegram"
)

func main() {
	cfg := configs.NewConfig()
	botApi, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatal(err)
	}
	botApi.Debug = true

	strg, err := storage.NewStorage(cfg)
	if err != nil {
		panic("Can not create the storage")
	}

	bot := telegram.NewBot(botApi, strg)
	if err = bot.Start(); err != nil {
		log.Fatal(err)
	}
}
