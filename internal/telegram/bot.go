package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"secretSanta/internal/cache"
	"secretSanta/internal/storage"
)

type Bot struct {
	bot     *tgbotapi.BotAPI
	storage storage.Storager
	cache   cache.Cache
}

func NewBot(bot *tgbotapi.BotAPI, storage storage.Storager, cache cache.Cache) *Bot {
	return &Bot{
		bot:     bot,
		storage: storage,
		cache:   cache,
	}
}

func (b *Bot) Start() error {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		// Handle callback for buttons
		if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
			if err := b.handleCallbackQuery(update.CallbackQuery); err != nil {
				fmt.Println(err)
			}
		}

		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		// Handle commands
		if update.Message.IsCommand() {
			if err := b.handleCommand(update.Message); err != nil {
				fmt.Println(err)
			}
			continue
		}

		// Handle other messages
		if err := b.handleMessage(update.Message); err != nil {
			fmt.Println(err)
		}
	}
	return nil
}
