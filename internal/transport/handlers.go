package transport

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	cfg "github.com/p1xart/anonchat/internal/config"
	services "github.com/p1xart/anonchat/internal/services/chatfuncs"
	"strconv"
	"sync"
)

// Хендлер команд
func HandlerCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := update.Message.Chat.ID
	wg := sync.WaitGroup{} // Костыль по сути. Вроде исправлял баг с преждевременной отправкой
	if len(cfg.Vars.Pool) == 2 {
		cfg.Vars.Lock.Lock()
		defer cfg.Vars.Lock.Unlock()
		cfg.Vars.Pool = []int64{}
	}

	switch update.Message.Text {
	case "/start":
		services.Deal(bot, update)
	case "/next", "/go", "/search", "Начать поиск":
		go services.SearchDialog(bot, update, false)

	case "/stop", "Остановить поиск":
		wg.Add(0)
		go services.StopDialog(bot, update.Message.Chat.ID, &wg, true)
		wg.Wait()

	case "/help":
		go services.SendMessage(bot, userID, "Этот бот сделан для анонимного общения между двумя неизвестными людьми.\nНа данный момент он находится в разработке и может время от времени умирать\n\nFAQ:\nВ: Я собираю что-то из того, что было отправлено сюда?\nО: Нет, но я понимаю, что не все поверят. Просто соблюдайте анонимность. Я не могу обещать, что вас, показавшего лицо, не найдет ваш собеседник.\n\nВ: Такие боты уже есть, зачем еще?\nО: В этом боте отсутствует реклама, разрабатывается он студентом для практики, никаких платных функций.\nВ будущем будут добавлены такие функции как:\n1) Режим без извращенцев (Отсутствие медиа-файлов, нельзя скинуть интим, следовательно извращуг в 2-3 раза меньше.)\n2) Балансный режим - на 1 девушку 1 парень\n3) Нейросеть для определения NSFW-контента\n\nВ: Вы можете сделать что-то с извращенцами и неадекватами?\nО: Увы, нет. Я не могу модерировать больше 1000 пользователей в день, но я подумаю над частичной реализацией. Потому и будет создан специальный режим.")

	case "/online":
		go services.SendMessage(bot, userID, fmt.Sprint("Сейчас общается ", strconv.Itoa(cfg.Vars.Online), " человек(а)"))

	case "/room":
		go services.SendMessage(bot, userID, "Эта функция пока в разработке.")

	case "/settings":
		go services.SendSettings(bot, update, userID)

	case "/idea":
		go services.SendMessage(bot, userID, "Напишите свое предложение.\nПишите чётко и без воды, как можно подробнее. Минимальное количество символов - 15.\n\n/cancel для отмены")
		go services.WriteImprove(bot, update)

	case "/setAge":
		go services.SendMessage(bot, userID, "Пожалуйста, введите возраст от 14 до 100\n\n/cancel - отменить")
		go services.ValidateAge(bot, update)

	case "/setGender":
		wg.Add(0)
		kb := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Мужчина"), tgbotapi.NewKeyboardButton("Женщина")))
		go services.SendMessageSetKeyboard(bot, userID, "Вы мужчина или женщина?\n\n/cancel - отменить", kb, &wg, false)
		wg.Wait()
		go services.ValidateSex(bot, update)

	case "/filters":
		go services.Filters(bot, update, false, false)
	}
}
