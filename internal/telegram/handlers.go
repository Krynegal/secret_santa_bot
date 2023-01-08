package telegram

import (
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"math/rand"
	"secretSanta/internal/storage"
	"secretSanta/internal/storage/models"
	"strconv"
	"time"
)

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

//Основной обработчик диалоговых сообщений
func (b *Bot) handleMessage(message *tgbotapi.Message) error {
	log.Printf("[%s] %s", message.From.UserName, message.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)

	state := b.StateKeeper.state(int(message.From.ID))
	log.Printf("\nstate: %s\n", state)

	switch state {
	case "join":
		log.Printf("entered roomID: %s", msg.Text)
		keyboard := tgbotapi.NewRemoveKeyboard(true)
		msg.ReplyMarkup = keyboard
		roomID, err := strconv.Atoi(msg.Text)
		if err != nil {
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
		b.StateKeeper.update(int(message.From.ID), "wishlist")
	case "wishlist":
		log.Printf("wishlist: %v", msg.Text)
		// пишем msg.Text в БД

		rooms, err := b.storage.RoomsWhereUserIsOrg(int(message.Chat.ID))
		if err != nil {
			return err
		}
		if len(rooms) != 0 {
			replyMsg := tgbotapi.NewMessage(message.Chat.ID, "Супер! Теперь ждем остальных участников")
			btns := []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("Показать участников", "showPlayers"),
			}
			k := tgbotapi.NewInlineKeyboardMarkup(btns)
			replyMsg.ReplyMarkup = k
			_, err = b.bot.Send(replyMsg)
			if err != nil {
				return err
			}
		} else {
			replyMsg := tgbotapi.NewMessage(message.Chat.ID, "Отлично! Теперь ждем, когда организатор начнет игру!")
			_, err = b.bot.Send(replyMsg)
			if err != nil {
				return err
			}
		}
	}

	if msg.Text != message.Text {
		_, err := b.bot.Send(msg)
		if err != nil {
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

	msg := tgbotapi.NewMessage(message.Chat.ID, "Добро пожаловать в Тайного Санту")
	b1 := tgbotapi.NewInlineKeyboardButtonData("Создать", "create")
	b2 := tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "enterRoomID")
	btns := []tgbotapi.InlineKeyboardButton{
		b1,
		b2,
	}
	k := tgbotapi.NewInlineKeyboardMarkup(btns)
	msg.ReplyMarkup = k

	_, err := b.bot.Send(msg)
	return err
}

// обработчик неизвестных команд
func (b *Bot) handleUnknownCommand(message *tgbotapi.Message) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, "Я не знаю такой команды")
	_, err := b.bot.Send(msg)
	return err
}

func generateRoomID() int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(100000)
}
