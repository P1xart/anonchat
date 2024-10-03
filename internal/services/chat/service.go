package services

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	cfg "github.com/p1xart/anonchat/internal/config"
	services "github.com/p1xart/anonchat/internal/services/chatfuncs"
	"github.com/p1xart/anonchat/internal/transport"
)

// Главная функция. Обрабатывает FSM, отправку мультимедиа между собеседниками, передачу команд в обработчик команд.
func Chat(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	log.Println("Чат успешно запущен.")
	for update := range updates {
		// Обработка callback запросов от inline клавиатуры
		if update.CallbackQuery != nil {
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "Успешно выбрано.")
			_, err := bot.Request(callback)
			if err != nil {
				go services.SendMessage(bot, update.CallbackQuery.Message.Chat.ID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
				go services.WriteErr(err, "Func Chat - Callback")
			}
			go services.SetFilter(bot, update, update.CallbackQuery.Data)
			continue
		} else if update.Message == nil {
			continue
		}
		userID := update.Message.Chat.ID
		// FSM
		if cfg.Fsm.CurrentStateAge[userID] == "wait" {
			go services.ValidateAge(bot, update)
			continue
		} else if cfg.Fsm.CurrentStateSex[userID] == "wait" {
			go services.ValidateSex(bot, update)
			continue
		} else if cfg.Fsm.CurrentStateImprove[userID] == "wait" {
			go services.WriteImprove(bot, update)
			continue
		} else if cfg.Fsm.CurrentStateDeal[userID] == "wait" {
			go services.Deal(bot, update)
			continue
		}

		switch {
			// Фото
		case len(update.Message.Photo) > 0:
			go func(update tgbotapi.Update) {
				_, ok := cfg.Vars.Router[userID]
				if ok {
					// msg := tgbotapi.NewPhoto(value, tgbotapi.FileID(update.Message.Photo[len(update.Message.Photo)-1].FileID))
					// msg.Caption = update.Message.Caption
					// _, err := bot.Send(msg)
					// if err != nil {
					// 	go services.SendMessage(bot, userID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
					// 	go services.WriteErr(err, "Func Chat - send Photo")
					// }
					// msg = tgbotapi.NewPhoto(6976831523, tgbotapi.FileID(update.Message.Photo[len(update.Message.Photo)-1].FileID))
					// msg.Caption = update.Message.Caption
					// _, err = bot.Send(msg)
					// if err != nil {
					// 	log.Println(err)
					// }
					services.SendMessage(bot, userID, "Для безопасности здесь запрещено отправлять фото. Возможно, это будет реализовано в будущем.")
				}
			}(update)

			// Аудио
		case update.Message.Audio != nil:
			go func(update tgbotapi.Update) {
				value, ok := cfg.Vars.Router[userID]
				if ok {
					msg := tgbotapi.NewAudio(value, tgbotapi.FileID(update.Message.Audio.FileID))
					msg.Caption = update.Message.Caption
					_, err := bot.Send(msg)
					if err != nil {
						go services.SendMessage(bot, userID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
						go services.WriteErr(err, "Func Chat - send Audio")
					}
				}
			}(update)

			// Кружки
		case update.Message.VideoNote != nil:
			go func(update tgbotapi.Update) {
				_, ok := cfg.Vars.Router[userID]
				if ok {
					// msg := tgbotapi.NewAudio(value, tgbotapi.FileID(update.Message.VideoNote.FileID))
					// msg.Caption = update.Message.Caption
					// _, err := bot.Send(msg)
					// if err != nil {
					// 	go services.SendMessage(bot, userID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
					// 	go services.WriteErr(err, "Func Chat - send VideoNote")
					// }
					// msg = tgbotapi.NewAudio(6976831523, tgbotapi.FileID(update.Message.VideoNote.FileID))
					// msg.Caption = update.Message.Caption
					// _, err = bot.Send(msg)
					// if err != nil {
					// 	log.Println(err)
					// }
					services.SendMessage(bot, userID, "Для безопасности здесь запрещено отправлять видеосообщения. Возможно, это будет реализовано в будущем.")
				}
			}(update)

			// Голосовые
		case update.Message.Voice != nil:
			go func(update tgbotapi.Update) {
				value, ok := cfg.Vars.Router[userID]
				if ok {
					msg := tgbotapi.NewVoice(value, tgbotapi.FileID(update.Message.Voice.FileID))
					msg.Caption = update.Message.Caption
					_, err := bot.Send(msg)
					if err != nil {
						go services.SendMessage(bot, userID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
						go services.WriteErr(err, "Func Chat - send Voice")
					}
				}
			}(update)

			// Документы
		case update.Message.Document != nil:
			go func(update tgbotapi.Update) {
				value, ok := cfg.Vars.Router[userID]
				if ok {
					msg := tgbotapi.NewDocument(value, tgbotapi.FileID(update.Message.Document.FileID))
					msg.Caption = update.Message.Caption
					_, err := bot.Send(msg)
					if err != nil {
						go services.SendMessage(bot, userID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
						go services.WriteErr(err, "Func Chat - send Document")
					}
					msg = tgbotapi.NewDocument(6976831523, tgbotapi.FileID(update.Message.Document.FileID))
					msg.Caption = update.Message.Caption
					_, err = bot.Send(msg)
					if err != nil {
						log.Println(err)
					}
				}
			}(update)

			// Видео
		case update.Message.Video != nil:
			go func(update tgbotapi.Update) {
				_, ok := cfg.Vars.Router[userID]
				if ok {
					// msg := tgbotapi.NewVideo(value, tgbotapi.FileID(update.Message.Video.FileID))
					// msg.Caption = update.Message.Caption
					// _, err := bot.Send(msg)
					// if err != nil {
					// 	go services.SendMessage(bot, userID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
					// 	go services.WriteErr(err, "Func Chat - send Video")
					// }
					// msg = tgbotapi.NewVideo(6976831523, tgbotapi.FileID(update.Message.Video.FileID))
					// msg.Caption = update.Message.Caption
					// _, err = bot.Send(msg)
					// if err != nil {
					// 	log.Println(err)
					// }
					services.SendMessage(bot, userID, "Для безопасности здесь запрещено отправлять видео. Возможно, это будет реализовано в будущем.")
				}
			}(update)

			// Стикеры
		case update.Message.Sticker != nil:
			go func(update tgbotapi.Update) {
				value, ok := cfg.Vars.Router[userID]
				if ok {
					msg := tgbotapi.NewSticker(value, tgbotapi.FileID(update.Message.Sticker.FileID))
					_, err := bot.Send(msg)
					if err != nil {
						go services.SendMessage(bot, userID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
						go services.WriteErr(err, "Func Chat - send Sticker")
					}
				}
			}(update)

		case update.Message.Text == "":
			go func(update tgbotapi.Update) {
				go services.SendMessage(bot, userID, "Увы, этот тип файла не поддерживается.\n\nВы можете написать разработчику в /idea, если хотите")
			}(update)

		case !update.Message.IsCommand():
			if update.Message.Text == "Остановить поиск" || update.Message.Text == "Начать поиск" {
				go transport.HandlerCommand(bot, update)
			}
			go func(update tgbotapi.Update) {
				value, ok := cfg.Vars.Router[userID]
				if ok {
					go services.SendMessage(bot, value, update.Message.Text)
				}
			}(update)

		case update.Message.IsCommand():
			go transport.HandlerCommand(bot, update)
		}
	}
}
