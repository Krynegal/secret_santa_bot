package telegram

import (
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
	case "main":
		log.Printf("\nmsg.Text: %s\n", msg.Text)
		if msg.Text == "Создать" {
			roomID := generateRoomID()
			if err := b.storage.CreateRoom(roomID); err != nil {
				return err
			}
			if err := b.storage.AssignRoomToUser(roomID, int(message.Chat.ID), true); err != nil {
				return err
			}
			replyText := fmt.Sprintf("Вот ваш roomID: %s", strconv.Itoa(roomID))
			replyMsg := tgbotapi.NewMessage(message.Chat.ID, replyText)
			buttons := []tgbotapi.KeyboardButton{
				tgbotapi.NewKeyboardButton("Распределить"),
			}
			keyboard := tgbotapi.NewReplyKeyboard(buttons)

			//var keyboard = tgbotapi.NewInlineKeyboardMarkup(
			//	tgbotapi.NewInlineKeyboardRow(
			//		tgbotapi.NewInlineKeyboardButtonData("Распределить", "distribute"),
			//	),
			//)

			replyMsg.ReplyMarkup = keyboard
			_, err := b.bot.Send(replyMsg)
			if err != nil {
				return err
			}
		} else {
			msg.Text = "Введите roomID"
			b.StateKeeper.update(int(message.From.ID), "join")
		}
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

		roomUsers, err := b.storage.UsersFromRoom(roomID)
		if err != nil {
			return err
		}
		log.Printf("roomUsers: %v", roomUsers)
		for _, user := range roomUsers {
			if int(message.Chat.ID) == user.ChatID {
				continue
			}
			msgToOrg := tgbotapi.NewMessage(int64(user.ChatID), message.Text)
			msgToOrg.Text = fmt.Sprintf("новое подключение: %s", message.From.FirstName)
			_, err = b.bot.Send(msgToOrg)
			if err != nil {
				return err
			}
		}
	case "Распределить":
		// смотрим, в каких группах юзер - организатор
		rooms, err := b.storage.RoomsWhereUserIsOrg(int(message.Chat.ID))
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
				for giver, getter := range shuffledUsers {
					msgToGiver := tgbotapi.NewMessage(int64(giver.ChatID), message.Text)
					msgToGiver.Text = fmt.Sprintf("Вы дарите подарок %s", getter.Username)
					_, err = b.bot.Send(msgToGiver)
					if err != nil {
						return err
					}
				}
			} else {
				msg.Text = "В комнате недостаточно игроков("
				_, err = b.bot.Send(msg)
				if err != nil {
					return err
				}
			}

		} else {
			msg.Text = "Вы не являетесь организатором ни одной группы"
			_, err = b.bot.Send(msg)
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
		ChatID:    int(message.Chat.ID),
		Firstname: message.From.FirstName,
		Lastname:  message.From.LastName,
		Username:  message.From.UserName,
	}

	if err := b.storage.AddUser(user); err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Добро пожаловать в Тайного Санту")
	btnCreate := tgbotapi.NewKeyboardButton("Создать")
	btnJoin := tgbotapi.NewKeyboardButton("Присоединиться")
	buttons := []tgbotapi.KeyboardButton{
		btnCreate,
		btnJoin,
	}
	keyboard := tgbotapi.NewReplyKeyboard(buttons)
	msg.ReplyMarkup = keyboard

	b.StateKeeper.update(int(message.From.ID), "main")
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
