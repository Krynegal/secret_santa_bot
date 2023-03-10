package telegram

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"math/rand"
	"secretSanta/internal/storage"
	"secretSanta/internal/storage/models"
	"strconv"
	"time"
)

func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) error {
	chatID := query.Message.Chat.ID
	switch query.Data {
	case "create":
		//roomID := generateRoomID()
		roomID := 1234
		if err := b.storage.CreateRoom(roomID); err != nil {
			return err
		}
		if err := b.storage.AssignRoomToUser(roomID, int(chatID), true); err != nil {
			return err
		}
		ctx := context.Background()
		if err := b.cache.AddUser(ctx, int(chatID), models.CacheNote{RoomID: roomID, IsOrganizer: true}); err != nil {
			return err
		}

		replyText := fmt.Sprintf("Вот ваш roomID: %s\n скиньте его другим игрокам", strconv.Itoa(roomID))
		replyMsg := tgbotapi.NewMessage(chatID, replyText)
		if _, err := b.bot.Send(replyMsg); err != nil {
			return err
		}
		msg := tgbotapi.NewMessage(chatID, "Что вы хотите получить в качестве подарка?")
		if _, err := b.bot.Send(msg); err != nil {
			return err
		}
		if err := b.cache.UpdateState(ctx, int(chatID), "wishlist"); err != nil {
			return err
		}
	case "enterRoomID":
		msg := tgbotapi.NewMessage(chatID, "Введите roomID")
		_, err := b.bot.Send(msg)
		if err != nil {
			return err
		}
		ctx := context.Background()
		if err = b.cache.UpdateState(ctx, int(chatID), "join"); err != nil {
			return err
		}
	case "showPlayers":
		ctx := context.Background()
		roomID, err := b.cache.RoomWhereUserIsOrg(ctx, int(chatID))
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		roomUsers, err := b.storage.UsersFromRoom(roomID)
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
		msgToOrg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(btns)

		if _, err = b.bot.Send(msgToOrg); err != nil {
			return err
		}
	case "distribute":
		ctx := context.Background()
		roomID, err := b.cache.RoomWhereUserIsOrg(ctx, int(chatID))
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
				var wishlist string
				wishlist, err = b.storage.Wish(roomID, getter.ID)
				if err != nil {
					return err
				}
				text := fmt.Sprintf("Вы дарите подарок: %s\n\nВот что было написано в пожелании к подарку:\n%s", getter.Username, wishlist)
				if giver.ID == 966098933 || giver.ID == 253141599 {
					msgToGiver := tgbotapi.NewMessage(int64(giver.ID), text)
					if _, err = b.bot.Send(msgToGiver); err != nil {
						return err
					}
				}
			}
		} else {
			msg := tgbotapi.NewMessage(chatID, "В комнате недостаточно игроков")
			if _, err = b.bot.Send(msg); err != nil {
				return err
			}
		}
	}
	return nil
}

const commandStart = "start"

//Обработчик "команд" - сообщений формата /<any text>
func (b *Bot) handleCommand(message *tgbotapi.Message) error {
	switch message.Command() {
	case commandStart:
		return b.handleStartCommand(message)
	default:
		return b.handleUnknownCommand(message)
	}
}

/*
Обработчик команды /start
Отправляет кнопку "Поиск" при запуске бота
*/
func (b *Bot) handleStartCommand(message *tgbotapi.Message) error {

	user := models.User{
		ID:        int(message.Chat.ID),
		Firstname: message.From.FirstName,
		Lastname:  message.From.LastName,
		Username:  message.From.UserName,
	}

	if err := b.storage.AddUser(user); err != nil {
		return err
	}
	ctx := context.Background()
	if err := b.cache.AddUser(ctx, user.ID, models.CacheNote{IsOrganizer: false}); err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Добро пожаловать в Тайного Санту")
	btns := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Создать", "create"),
		tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "enterRoomID"),
	}
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(btns)
	_, err := b.bot.Send(msg)
	return err
}

// обработчик неизвестных команд
func (b *Bot) handleUnknownCommand(message *tgbotapi.Message) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, "Я не знаю такой команды")
	_, err := b.bot.Send(msg)
	return err
}

//Основной обработчик диалоговых сообщений
func (b *Bot) handleMessage(message *tgbotapi.Message) error {
	log.Printf("[%s] %s", message.From.UserName, message.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)

	ctx := context.Background()
	user, err := b.cache.User(ctx, int(message.From.ID))
	if err != nil {
		return err
	}
	log.Printf("\nstate: %s\n", user.State)

	switch user.State {
	case "join":
		log.Printf("entered roomID: %s", msg.Text)
		keyboard := tgbotapi.NewRemoveKeyboard(true)
		msg.ReplyMarkup = keyboard
		var roomID int
		roomID, err = strconv.Atoi(msg.Text)
		if err != nil {
			return err
		}
		if err = b.cache.AddUser(ctx, int(message.Chat.ID), models.CacheNote{RoomID: roomID, IsOrganizer: false}); err != nil {
			return err
		}
		err = b.storage.AssignRoomToUser(roomID, int(message.Chat.ID), false)
		if err != nil {
			if errors.Is(err, storage.RoomUserPairExists) {
				msg.Text = "Вы уже присоединились к этой комнате"
				return nil
			}
			return err
		}
		msg.Text = "Что вы хотите получить в качестве подарка?"
		ctx = context.Background()
		if err = b.cache.UpdateState(ctx, int(message.From.ID), "wishlist"); err != nil {
			return err
		}
	case "wishlist":
		log.Printf("\nuserID: %d\n", message.Chat.ID)
		ctx = context.Background()
		user, err = b.cache.User(ctx, int(message.Chat.ID))
		if err != nil {
			return err
		}
		if err = b.storage.AddWish(user.RoomID, int(message.Chat.ID), msg.Text); err != nil {
			return err
		}

		ctx = context.Background()
		user, err = b.cache.User(ctx, int(message.Chat.ID))
		if err != nil {
			return err
		}
		//log.Printf("\nroom: %v\n", room)
		if !user.IsOrganizer {
			replyMsg := tgbotapi.NewMessage(message.Chat.ID, "Отлично! Теперь ждем, когда организатор начнет игру!")
			if _, err = b.bot.Send(replyMsg); err != nil {
				return err
			}
		} else {
			replyMsg := tgbotapi.NewMessage(message.Chat.ID, "Супер! Теперь ждем остальных участников")
			btns := []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("Показать участников", "showPlayers"),
			}
			k := tgbotapi.NewInlineKeyboardMarkup(btns)
			replyMsg.ReplyMarkup = k
			if _, err = b.bot.Send(replyMsg); err != nil {
				return err
			}
		}
	}

	if msg.Text != message.Text {
		if _, err = b.bot.Send(msg); err != nil {
			return err
		}
	}

	return nil
}

func shuffleUsers(users []models.User) map[models.User]models.User {
	shuffledUsers := make(map[models.User]models.User, len(users))
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })
	for i := 0; i < len(users)-1; i++ {
		shuffledUsers[users[i]] = users[i+1]
	}
	shuffledUsers[users[len(users)-1]] = users[0]
	return shuffledUsers
}

func generateRoomID() int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(100000)
}
