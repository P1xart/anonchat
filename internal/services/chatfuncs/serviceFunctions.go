package services

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	cfg "github.com/p1xart/anonchat/internal/config"
	db "github.com/p1xart/anonchat/internal/database"
)

// Валидация возраста.
func ValidateAge(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := update.Message.Chat.ID
	if (update.Message.IsCommand() || update.Message.Text == "Начать поиск" || update.Message.Text == "Остановить поиск") && update.Message.Text != "/setAge" && update.Message.Text != "/cancel" {
		go SendMessage(bot, userID, "Сначала отмените текущую операцию с помощью /cancel")
		return
	}
	if update.Message.Text == "/setAge" {
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateAge[userID] = "wait"
		return
	}
	if update.Message.Command() == "cancel" {
		go SendMessage(bot, userID, "Вы отменили выбор возраста")
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateAge[userID] = "done"
		return
	}
	age, err := strconv.Atoi(update.Message.Text)
	if err != nil {
		go SendMessage(bot, userID, "Ошибка при валидации\nПроверьте, нет ли букв.")
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateAge[userID] = "wait"
		return
	}
	if age >= 14 && age <= 100 {
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateAge[userID] = "done"
		db.SetAge(userID, age, cfg.Vars.DB)
		SendSettings(bot, update, userID)
	} else {
		go SendMessage(bot, userID, "Неправильный возраст.\nВы должны быть старше 14 и младше 100 лет.")
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateAge[userID] = "wait"
	}
}

// Валидация пола
func ValidateSex(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := update.Message.Chat.ID
	word := []rune(string(strings.ToLower(update.Message.Text)))

	if (update.Message.IsCommand() || update.Message.Text == "Начать поиск" || update.Message.Text == "Остановить поиск") && update.Message.Text != "/setGender" && update.Message.Text != "/cancel" {
		go SendMessage(bot, userID, "Сначала отмените текущую операцию с помощью /cancel")
		return
	}

	if update.Message.Text == "/setGender" {
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateSex[userID] = "wait"
		return
	}

	if update.Message.Command() == "cancel" {
		go SendMessageRemoveKeyboard(bot, userID, "Вы отменили выбор пола")
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateSex[userID] = "done"
		return
	}

	if strings.EqualFold(string(word[0]), "м") {
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateSex[userID] = "done"
		db.SetSex(userID, "m", cfg.Vars.DB)
		SendSettings(bot, update, userID)
	} else if strings.EqualFold(string(word[0]), "ж") {
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateSex[userID] = "done"
		db.SetSex(userID, "f", cfg.Vars.DB)
		SendSettings(bot, update, userID)
	} else {
		wg := sync.WaitGroup{}
		kb := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Мужчина"), tgbotapi.NewKeyboardButton("Женщина")))
		go SendMessageSetKeyboard(bot, userID, "Я вас не понимаю...\nПроверьте слово на ошибку.", kb, &wg, false)
		cfg.Fsm.Lock.Lock()
		defer cfg.Fsm.Lock.Unlock()
		cfg.Fsm.CurrentStateSex[userID] = "wait"
	}
}

// Функция остановки диалога
func StopDialog(bot *tgbotapi.BotAPI, userID int64, wg *sync.WaitGroup, send bool) {
	delCompanion, ok := cfg.Vars.Router[userID] // В паре с кем либо?

	if db.IsSearch(bot, userID, cfg.Vars.DB)  { // Если is_search=true, то останавливаем поиск. Иначе завершаем диалог.
		db.StopSearch(bot, userID, cfg.Vars.DB)
		go SendMessageSetKeyboard(bot, userID, "Вы остановили поиск собеседника.\n\n/go для поиска следующего", cfg.SearchKb, wg, false)
	} else if ok {
		if send {
			go SendMessageSetKeyboard(bot, userID, "Вы завершили диалог!\n\n/go для поиска следующего", cfg.SearchKb, wg, false)
		}
		go SendMessageSetKeyboard(bot, delCompanion, "Собеседник завершил диалог...\n\n/go для поиска следующего", cfg.SearchKb, wg, false)
		cfg.Vars.Lock.Lock() // Блокируем переменные для защиты от одновременной записи
		delete(cfg.Vars.Router, userID)
		delete(cfg.Vars.Router, delCompanion)
		cfg.Vars.Online -= 2
		cfg.Vars.Lock.Unlock()
	} else {
		go SendMessageSetKeyboard(bot, userID, "У вас нет собеседника!\n\n/go для его поиска.", cfg.SearchKb, wg, false)
	}
	if !send {
		wg.Done()
	}
}

// Функция поиска собеседника
func SearchDialog(bot *tgbotapi.BotAPI, update tgbotapi.Update, isCallback bool) {
	var userID int64
	var resSex string
	wg := sync.WaitGroup{}
	resAgeCategories := make([]string, 0, 5)
	kb := cfg.StopKb // Кнопка "Остановить диалог"
	if isCallback {
		userID = update.CallbackQuery.Message.Chat.ID
	} else {
		userID = update.Message.Chat.ID
	}
	
	_, ok := cfg.Vars.Router[userID]
	if ok {
		wg.Add(1)
		go StopDialog(bot, userID, &wg, false)
		wg.Wait()
	}
	sexArr, ageCategoryArr, err := db.GetSexAgeFilter(bot, userID, cfg.Vars.DB)
	pair := db.SearchCompanion(bot, userID, sexArr, ageCategoryArr, cfg.Vars.DB)
	if pair[0] != 0 { // Если есть хотя бы 1 user_id, то в pair гарантированно 2 элемента. Связываем собеседников.
		for _, value := range pair {
			go SendMessageRemoveKeyboard(bot, value, "Собеседник найден!\n\n/go - следующий\n/stop - остановить общение")
		}
		cfg.Vars.Lock.Lock() // Блокируем переменные для защиты от одновременной записи
		cfg.Vars.Router[pair[0]] = pair[1]
		cfg.Vars.Router[pair[1]] = pair[0]
		cfg.Vars.Online += 2
		cfg.Vars.Lock.Unlock()
	}
	
	// BEGIN Перевод сокращений из записей базы данных в понятный пользователю вид
	// Обработка категорий
	if (len(ageCategoryArr) == 1 && ageCategoryArr[0] == 0) || len(ageCategoryArr) == 5  {
		resAgeCategories = append(resAgeCategories, "Любая категория")
	} else {
		for _, v := range ageCategoryArr {
			switch v {
			case 1:
				resAgeCategories = append(resAgeCategories, "До 17")
			case 2:
				resAgeCategories = append(resAgeCategories, "18-25")
			case 3:
				resAgeCategories = append(resAgeCategories, "26-33")
			case 4:
				resAgeCategories = append(resAgeCategories, "34-41")
			case 5:
				resAgeCategories = append(resAgeCategories, "Больше 42")
			}
		}
	}

	if len(sexArr) == 0 || len(sexArr) == 2 { // Обработка пола
			resSex = "Любой пол"
		} else if sexArr[0] == "m" {
			resSex = "Мужчина"
		} else {
			resSex = "Женщина"
		}
		resAge := strings.Join(resAgeCategories, ", ")
		// END

		if err != nil {
			go SendMessage(bot, userID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
			go WriteErr(err, "Func SearchDialog")
		}
		wg.Add(1)
		if userID != pair[1] {
			SendMessageSetKeyboard(bot, userID, fmt.Sprintf("Идёт поиск собеседника...\nИскомый пол: %s\nИскомая возрастная категория: %s\n\nДля отмены введите /stop", resSex, resAge), kb, &wg, true)
		}
		wg.Wait()
}

// Отправка сообщения пользователю
func SendMessage(bot *tgbotapi.BotAPI, userID int64, textMessage string) {
	msg := tgbotapi.NewMessage(userID, textMessage)
	_, err := bot.Send(msg)
	if err != nil {
		if err.Error() == "Forbidden: bot was blocked by the user" {
			delCompanion, ok := cfg.Vars.Router[userID]
			if ok {
				cfg.Vars.Lock.Lock() // Блокируем переменные для защиты от одновременной записи
				delete(cfg.Vars.Router, userID)
				delete(cfg.Vars.Router, delCompanion)
				cfg.Vars.Online -= 2
				cfg.Vars.Lock.Unlock()
				wg := sync.WaitGroup{}
				go SendMessageSetKeyboard(bot, delCompanion, "Собеседник заблокировал чат.\n\n/go для поиска следующего", cfg.SearchKb, &wg, false)
			}
		}
		go WriteErr(err, "Func SendMessage")
	}
}

// Отправка сообщения пользователю и установка inline клавиатуры одновременно.
func SendMessageSetKeyboard(bot *tgbotapi.BotAPI, userID int64, textMessage string, kb tgbotapi.ReplyKeyboardMarkup, wg *sync.WaitGroup, wait bool) {
	msg := tgbotapi.NewMessage(userID, textMessage)
	msg.ReplyMarkup = kb
	_, err := bot.Send(msg)
	if err != nil {
		if err.Error() == "Forbidden: bot was blocked by the user" {
			delCompanion, ok := cfg.Vars.Router[userID]
			if ok {
				cfg.Vars.Lock.Lock()
				delete(cfg.Vars.Router, userID)
				delete(cfg.Vars.Router, delCompanion)
				cfg.Vars.Online -= 2
				cfg.Vars.Lock.Unlock()
				wg := sync.WaitGroup{}
				go SendMessageSetKeyboard(bot, delCompanion, "Собеседник заблокировал чат.\n\n/go для поиска следующего", cfg.SearchKb, &wg, false)
			}
		}
		go WriteErr(err, "Func SendMessageSetKeyboard")
	}
	if wait {
		wg.Done()
	}
}

// Отправка сообщения пользователю и удаление inline клавиатуры одновременно.
func SendMessageRemoveKeyboard(bot *tgbotapi.BotAPI, receiver int64, textMessage string) {
	msg := tgbotapi.NewMessage(receiver, textMessage)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	_, err := bot.Send(msg)
	if err != nil {
		go SendMessage(bot, cfg.Vars.Router[receiver], "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
		go WriteErr(err, "Func SendMessageRemoveKeyboard")
	}
}

// Функция записи предложений улучшений
func WriteImprove(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := update.Message.Chat.ID
	wg := sync.WaitGroup{}

	if (update.Message.IsCommand() || update.Message.Text == "Начать поиск" || update.Message.Text == "Остановить поиск") && update.Message.Text != "/idea" && update.Message.Text != "/cancel" {
		go SendMessage(bot, userID, "Сначала отмените текущую операцию с помощью /cancel")
		return
	}
	cfg.Vars.Lock.Lock()
	defer cfg.Vars.Lock.Unlock()
	switch update.Message.Text {
	case "/cancel":
		go SendMessageSetKeyboard(bot, userID, "Вы отменили предложение идеи.", cfg.SearchKb, &wg, false)
		cfg.Fsm.CurrentStateImprove[userID] = "done"
		return
	case "/idea":
		cfg.Fsm.CurrentStateImprove[userID] = "wait"
		return
	default:
		if len(update.Message.Text) < 15 {
			go SendMessage(bot, userID, "Текст должен состоять минимум из 15 символов.")
			return
		}
		go SendMessage(bot, userID, "Улучшение записано. Спасибо за ваше содействие.")
		cfg.Fsm.CurrentStateImprove[userID] = "done"
		var err error
		file, err := os.Create(fmt.Sprint("/home/pixart/go/src/github.com/p1xart/anonchat/improve/", update.Message.Text[:10], ".txt"))
		if err != nil {
			go SendMessage(bot, update.Message.Chat.ID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
			go WriteErr(err, "Func WriteImprove")
			return
		}
		defer file.Close()
		file.WriteString(fmt.Sprint(update.Message.Chat.ID, "\n", update.Message.Text))
	}
}

// Функция записи ошибок
func WriteErr(E error, location string) {
	var err error

	file, err := os.Create(fmt.Sprint("/home/pixart/go/src/github.com/p1xart/anonchat/errs/", E.Error()[:10], ".txt"))
	if err != nil {
		log.Println(E, "Func WriteErr")
		return
	}
	defer file.Close()
	file.WriteString(fmt.Sprint(time.Now(), "\n", E, "\n", cfg.Vars.Router, "\n", cfg.Vars.Pool))
	log.Println(fmt.Sprintf("%s:", location), E)
}

// Отправка пользователю настроек
func SendSettings(bot *tgbotapi.BotAPI, update tgbotapi.Update, userID int64) {
	var ageASCII string
	wg := sync.WaitGroup{}

	age_category, sex, err := db.GetSexAge(bot, userID, cfg.Vars.DB)
	if err != nil {
		go SendMessage(bot, update.Message.Chat.ID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
		go WriteErr(err, "Func SendSettings")
		return
	}

	switch age_category {
	case 1:
		ageASCII = "До 17"
	case 2:
		ageASCII = "18-25"
	case 3:
		ageASCII = "26-33"
	case 4:
		ageASCII = "34-41"
	case 5:
		ageASCII = "Больше 42"
	default:
		ageASCII = "Не известно"
	}

	if sex == "m" {
		sex = "Мужчина"
	} else if sex == "f" {
		sex = "Женщина"
	} else {
		sex = "Не известно"
	}
	
	go SendMessageSetKeyboard(bot, userID, fmt.Sprint("Ваш пол - ", sex, "\nВаша возрастная категория - ", ageASCII, "\n\n/setAge - указать возраст\n/setGender - указать пол"), cfg.SearchKb, &wg, false)
}

// Начальное подтверждение положения и правил.
func Deal(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userID := update.Message.Chat.ID
	wg := sync.WaitGroup{}
	kb := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Подтверждаю")))
	if update.Message.IsCommand() && update.Message.Text != "/start" && update.Message.Text != "Подтверждаю" {
		go SendMessageSetKeyboard(bot, userID, "Для продолжения прочтите документ выше и при согласии нажмите кнопку ниже 'Подтверждаю'", kb, &wg, false)
		return
	}
	if update.Message.Text == "Подтверждаю" {
		userID = update.Message.Chat.ID
		cfg.Vars.Lock.Lock()
		db.CreateUser(userID, cfg.Vars.DB)
		cfg.Fsm.CurrentStateDeal[userID] = "done"
		cfg.Vars.Lock.Unlock()
		SendMessage(bot, userID, "/go - искать следующего собеседника\n/stop - остановить поиск или диалог\n/filters - поиск по полу и возрасту\n/settings - настройки\n/online - онлайн\n/idea - предложить идею\n/help - справка")
		SearchDialog(bot, update, false)
		return
	}
	cfg.Fsm.CurrentStateDeal[userID] = "wait"
	go SendMessageSetKeyboard(bot, userID, "Приветствую! Это сервис для анонимного общения между двумя неизвестными людьми.\n\nВо избежание проблем, сомнительных моментов и вреда - прочтите положение https://telegra.ph/Polozhenie-Anonimnogo-chata-06-11, и, если вы согласны, нажмите кнопку 'Подтверждаю', в противном случае покиньте сервис!", kb, &wg, false)
}

// Отправка фильтров для поиска
func Filters(bot *tgbotapi.BotAPI, update tgbotapi.Update, isCallback bool, EditKeyboard bool) {
	var userID int64
	// Если callback, то юзаем Chat ID из запроса Callback. Иначе, при использовании из Message, будет nil.
	if isCallback {
		userID = update.CallbackQuery.Message.Chat.ID
	} else {
		userID = update.Message.Chat.ID
	}

	age, sex, err := db.GetSexAge(bot, userID, cfg.Vars.DB)
	if err != nil {
		go SendMessage(bot, update.Message.Chat.ID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
		go WriteErr(err, "Func Filters - GetSexAgeFilter func")
		return
	}

	if (age == 0 || sex == "unknown") && !EditKeyboard {
		wg := sync.WaitGroup{}
		wg.Add(0)
		go SendMessageSetKeyboard(bot, userID, "Чтобы использовать эту функцию вы должны указать свой пол и возраст.\n/settings для настройки", cfg.SearchKb, &wg, false)
		wg.Wait()
		return
	}

	sexArr, ageArr, err := db.GetSexAgeFilter(bot, userID, cfg.Vars.DB)
	if err != nil {
		go SendMessage(bot, update.Message.Chat.ID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
		go WriteErr(err, "Func Filters - GetSexAgeFilter func")
		return
	}
	// Всего 5 категорий возрастов и 2 пола.
	// Из БД переменная ageArr поступает примерно как [cat1, cat2, cat3, cat4, cat5]
	// Где cat1/2/3/4/5 - число (категория)
	c17 := "До 17"
	c25 := "18-25"
	c33 := "26-33"
	c41 := "34-41"
	c42 := "После 42"
	w := "Девушку"
	m := "Парня"
	a := "Неважно"

	for _, v := range sexArr {
		if len(sexArr) == 2 {
			a = "Неважно ✅"
			break
		}
		if v == "f" {
			w = "Девушку ✅"
		} else if v == "m" {
			m = "Парня ✅"
		}
	}

	for _, value := range ageArr {
		if value == 1 {
			c17 = "До 17  ✅"
		}
		if value == 2 {
			c25 = "18-25  ✅"
		}
		if value == 3 {
			c33 = "26-33  ✅"
		}
		if value == 4 {
			c41 = "34-41  ✅"
		}
		if value == 5 {
			c42 = "После 42  ✅"
		}
	}
	// Разметка inline клавиатуры фильтра.
	// c17, c25, c33... - переменные с названиями категорий.
	// "0", "1", "2"... - Callback идентификаторы
	var ageKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c17, "0"),
			tgbotapi.NewInlineKeyboardButtonData(c25, "1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c33, "2"),
			tgbotapi.NewInlineKeyboardButtonData(c41, "3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c42, "4"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(w, "5"),
			tgbotapi.NewInlineKeyboardButtonData(m, "6"),
			tgbotapi.NewInlineKeyboardButtonData(a, "7"),
		),
	)

	if !EditKeyboard {
		msg := tgbotapi.NewMessage(userID, "Кого вы ищите?\n\nКак только определитесь - введите /go для поиска собеседника")
		msg.ReplyMarkup = ageKeyboard
		msgOut, err := bot.Send(msg)
		if err != nil {
			go SendMessage(bot, update.Message.Chat.ID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
			go WriteErr(err, "FuncFilters - !EditKeyboard")
			return
		}
		cfg.Vars.Lock.Lock()
		cfg.Vars.EditReplyMarkup[userID] = msgOut.MessageID
		cfg.Vars.Lock.Unlock()
	}
	cfg.Vars.Lock.Lock()
	defer cfg.Vars.Lock.Unlock()
	cfg.Vars.UpdReplyMarkup[userID] = ageKeyboard
}

// Запись фильтров для поиска
func SetFilter(bot *tgbotapi.BotAPI, update tgbotapi.Update, data string) {
	userID := update.CallbackQuery.Message.Chat.ID
	_, ageArr, err := db.GetSexAgeFilter(bot, userID, cfg.Vars.DB)
	if err != nil {
		go SendMessage(bot, update.Message.Chat.ID, "Произошла ошибка... Свяжитесь со мной, пожалуйста - @n3th3rus и сообщите подробности.")
		go WriteErr(err, "Func SetFilter - GetSexAgeFilter func")
		return
	}

	// Из БД переменная ageArr поступает примерно как [cat1, cat2, cat3, cat4, cat5]
	// Где cat1/2/3/4/5 - число (категория)
	var c17, c25, c33, c41, c42 bool
	// Распрашиваем, какие категории выбраны
	for _, value := range ageArr {
		if value == 1 {
			c17 = true
		}
		if value == 2 {
			c25 = true
		}
		if value == 3 {
			c33 = true
		}
		if value == 4 {
			c41 = true
		}
		if value == 5 {
			c42 = true
		}
	}

	switch data {
	case "0": // Callback идентификатор
		if c17 { // Если уже выбрано, то удаляем из бд
			db.DeleteAgeFilter(bot, userID, 1, cfg.Vars.DB)
		} else { // Иначе задаем
			db.SetAgeFilter(bot, userID, "1", cfg.Vars.DB)
		}
	case "1":
		if c25 {
			db.DeleteAgeFilter(bot, userID, 2, cfg.Vars.DB)
		} else {
			db.SetAgeFilter(bot, userID, "2", cfg.Vars.DB)
		}
	case "2":
		if c33 {
			db.DeleteAgeFilter(bot, userID, 3, cfg.Vars.DB)
		} else {
			db.SetAgeFilter(bot, userID, "3", cfg.Vars.DB)
		}
	case "3":
		if c41 {
			db.DeleteAgeFilter(bot, userID, 4, cfg.Vars.DB)
		} else {
			db.SetAgeFilter(bot, userID, "4", cfg.Vars.DB)
		}
	case "4":
		if c42 {
			db.DeleteAgeFilter(bot, userID, 5, cfg.Vars.DB)
		} else {
			db.SetAgeFilter(bot, userID, "5", cfg.Vars.DB)
		}
	case "5":
		db.SetSexFilter(bot, userID, "женщина", cfg.Vars.DB)
	case "6":
		db.SetSexFilter(bot, userID, "мужчина", cfg.Vars.DB)
	case "7":
		db.SetSexFilter(bot, userID, "любой", cfg.Vars.DB)
	}

	Filters(bot, update, true, true)
	delConf := tgbotapi.NewEditMessageTextAndMarkup(userID, cfg.Vars.EditReplyMarkup[userID], "Кого вы ищите?\n\nКак только определитесь - введите /go для поиска собеседника", cfg.Vars.UpdReplyMarkup[userID])
	_, err = bot.Send(delConf)
	if err != nil {
		go WriteErr(err, "Func SetFilters - edit InlineKeyboard")
	}
}
