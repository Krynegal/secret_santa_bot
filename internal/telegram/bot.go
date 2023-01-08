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
			switch update.CallbackQuery.Data {
			case "create":
				//roomID := generateRoomID()
				roomID := 1234
				if err := b.storage.CreateRoom(roomID); err != nil {
					return err
				}
				if err := b.storage.AssignRoomToUser(roomID, int(update.CallbackQuery.Message.Chat.ID), true); err != nil {
					return err
				}

				replyText := fmt.Sprintf("Вот ваш roomID: %s\n скиньте его другим игрокам", strconv.Itoa(roomID))
				replyMsg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, replyText)
				_, err := b.bot.Send(replyMsg)
				if err != nil {
					return err
				}
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Что вы хотите получить в качестве подарка?")
				_, err = b.bot.Send(msg)
				if err != nil {
					return err
				}
				b.StateKeeper.update(int(update.CallbackQuery.Message.Chat.ID), "wishlist")
			case "enterRoomID":
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Введите roomID")
				_, err := b.bot.Send(msg)
				if err != nil {
					return err
				}
				b.StateKeeper.update(int(update.CallbackQuery.Message.Chat.ID), "join")
			case "showPlayers":
				rooms, err := b.storage.RoomsWhereUserIsOrg(int(update.CallbackQuery.Message.Chat.ID))
				if err != nil {
					return err
				}
				roomUsers, err := b.storage.UsersFromRoom(rooms[0])
				if err != nil {
					return err
				}
				text := "Участники: \n\n"
				for i, user := range roomUsers {
					if int64(user.ID) == update.CallbackQuery.Message.Chat.ID {
						continue
					}
					text += fmt.Sprintf("%d. %s %s\n", i, user.Firstname, user.Lastname)
				}
				msgToOrg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
				distrBtn := tgbotapi.NewInlineKeyboardButtonData("Распределить", "distribute")
				btns := []tgbotapi.InlineKeyboardButton{
					distrBtn,
				}
				k := tgbotapi.NewInlineKeyboardMarkup(btns)
				msgToOrg.ReplyMarkup = k

				_, err = b.bot.Send(msgToOrg)
				if err != nil {
					return err
				}
			case "distribute":
				// смотрим, в каких группах юзер - организатор
				rooms, err := b.storage.RoomsWhereUserIsOrg(int(update.CallbackQuery.Message.Chat.ID))
				if err != nil {
					return err
				}
				// TODO: если групп >1, спршиваем, про какую идет речь
				// после выбора группы, находим всех ее участников и перемешиваем
				var roomID int
				if len(rooms) != 0 {
					roomID = rooms[0]
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
						msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "В комнате недостаточно игроков")
						_, err = b.bot.Send(msg)
						if err != nil {
							return err
						}
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
