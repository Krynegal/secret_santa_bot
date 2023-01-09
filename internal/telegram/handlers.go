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
		_, err := b.bot.Send(replyMsg)
		if err != nil {
			return err
		}
		msg := tgbotapi.NewMessage(chatID, "Что вы хотите получить в качестве подарка?")
		_, err = b.bot.Send(msg)
		if err != nil {
			return err
		}
		//b.StateKeeper.update(int(chatID), "wishlist")
		if err = b.cache.UpdateState(ctx, int(chatID), "wishlist"); err != nil {
			return err
		}
	case "enterRoomID":
		msg := tgbotapi.NewMessage(chatID, "Введите roomID")
		_, err := b.bot.Send(msg)
		if err != nil {
			return err
		}
		//b.StateKeeper.update(int(chatID), "join")
		ctx := context.Background()
		if err = b.cache.UpdateState(ctx, int(chatID), "join"); err != nil {
			return err
		}
	case "showPlayers":
		//room, err := b.storage.RoomWhereUserIsOrg(int(chatID))
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
		k := tgbotapi.NewInlineKeyboardMarkup(btns)
		msgToOrg.ReplyMarkup = k

		_, err = b.bot.Send(msgToOrg)
		if err != nil {
			return err
		}
	case "distribute":
		// смотрим, в какой комнате юзер является организатором
		// roomID, err := b.storage.RoomWhereUserIsOrg(int(chatID))
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

//Основной обработчик диалоговых сообщений
func (b *Bot) handleMessage(message *tgbotapi.Message) error {
	log.Printf("[%s] %s", message.From.UserName, message.Text)
	msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)

	//state := b.StateKeeper.state(int(message.From.ID))
	ctx := context.Background()
	note, err := b.cache.State(ctx, int(message.From.ID))
	if err != nil {
		return err
	}
	log.Printf("\nstate: %s\n", note.State)

	switch note.State {
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
		//b.StateKeeper.update(int(message.From.ID), "wishlist")
		ctx = context.Background()
		if err = b.cache.UpdateState(ctx, int(message.From.ID), "wishlist"); err != nil {
			return err
		}
	case "wishlist":
		log.Printf("wishlist: %v", msg.Text)
		// пишем msg.Text в БД
		// нужно хранить roomID
		//if err := b.storage.AddWish(int(message.Chat.ID), msg.Text); err != nil {
		//	return err
		//}

		//_, err = b.storage.RoomWhereUserIsOrg(int(message.Chat.ID))
		ctx = context.Background()
		room, err := b.cache.RoomWhereUserIsOrg(ctx, int(message.Chat.ID))
		if err != nil {
			return err
		}
		log.Printf("\nroom: %v\n", room)
		if room == 0 {
			replyMsg := tgbotapi.NewMessage(message.Chat.ID, "Отлично! Теперь ждем, когда организатор начнет игру!")
			_, err = b.bot.Send(replyMsg)
			if err != nil {
				return err
			}
		} else {
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
		}
	}

	if msg.Text != message.Text {
		_, err = b.bot.Send(msg)
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

func generateRoomID() int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(100000)
}
