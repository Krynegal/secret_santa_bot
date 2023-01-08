package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"secretSanta/internal/storage"
	"strconv"
	"sync"
)

type Bot struct {
	bot         *tgbotapi.BotAPI
	storage     storage.Storager
	StateKeeper StateKeeper
}

func NewBot(bot *tgbotapi.BotAPI, storage storage.Storager) *Bot {
	return &Bot{
		bot:     bot,
		storage: storage,
		StateKeeper: StateKeeper{
			mu:     sync.RWMutex{},
			states: map[int]string{},
		},
	}
}

func (b *Bot) Start() error {
	log.Printf("Authorized on account %s", b.bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		log.Printf("Callbackquery: %v", update.CallbackQuery)

		if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
			chatID := update.CallbackQuery.Message.Chat.ID
			switch update.CallbackQuery.Data {
			case "create":
				//roomID := generateRoomID()
				roomID := 1234
				if err := b.storage.CreateRoom(roomID); err != nil {
					return err
				}
				if err := b.storage.AssignRoomToUser(roomID, int(chatID), true); err != nil {
					return err
				}

				replyText := fmt.Sprintf("Вот ваш roomID: %s\n скиньте его другим игрокам", strconv.Itoa(roomID))
				replyMsg := tgbotapi.NewMessage(chatID, replyText)
				_, err := b.bot.Send(replyMsg)
				if err != nil {
					return err
				}
				msg := tgbotapi.NewMessage(chatID, "Что вы хотите получить в качестве подарка?")
				_, err = b.bot.Send(msg)
				if err != nil {
					return err
				}
				b.StateKeeper.update(int(chatID), "wishlist")
			case "enterRoomID":
				msg := tgbotapi.NewMessage(chatID, "Введите roomID")
				_, err := b.bot.Send(msg)
				if err != nil {
					return err
				}
				b.StateKeeper.update(int(chatID), "join")
			case "showPlayers":
				room, err := b.storage.RoomWhereUserIsOrg(int(chatID))
				if err != nil {
					return err
				}
				roomUsers, err := b.storage.UsersFromRoom(room)
				if err != nil {
					return err
				}
				text := "Участники: \n\n"
				for i, user := range roomUsers {
					if int64(user.ID) == chatID {
						continue
					}
					text += fmt.Sprintf("%d. %s %s\n", i, user.Firstname, user.Lastname)
				}
				msgToOrg := tgbotapi.NewMessage(chatID, text)
				btns := []tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardButtonData("Распределить", "distribute"),
				}
				k := tgbotapi.NewInlineKeyboardMarkup(btns)
				msgToOrg.ReplyMarkup = k

				_, err = b.bot.Send(msgToOrg)
				if err != nil {
					return err
				}
			case "distribute":
				// смотрим, в какой комнате юзер является организатором
				roomID, err := b.storage.RoomWhereUserIsOrg(int(chatID))
				if err != nil {
					return err
				}
				roomUsers, err := b.storage.UsersFromRoom(roomID)
				if err != nil {
					return err
				}
				if len(roomUsers) > 2 {
					shuffledUsers := shuffleUsers(roomUsers)
					log.Printf("shuffledUsers: %v", shuffledUsers)
					for giver, getter := range shuffledUsers {
						msgToGiver := tgbotapi.NewMessage(int64(giver.ID), fmt.Sprintf("Вы дарите подарок %s", getter.Username))
						_, err = b.bot.Send(msgToGiver)
						if err != nil {
							return err
						}
					}
				} else {
					msg := tgbotapi.NewMessage(chatID, "В комнате недостаточно игроков")
					_, err = b.bot.Send(msg)
					if err != nil {
						return err
					}
				}
			}
		}

		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		//Handle commands
		if update.Message.IsCommand() {
			if err := b.handleCommand(update.Message); err != nil {
				fmt.Println(err)
			}
			continue
		}

		//Handle other messages
		if err := b.handleMessage(update.Message); err != nil {
			fmt.Println(err)
		}
	}
	return nil
}
