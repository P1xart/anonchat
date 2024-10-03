package config

import (
	"database/sql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	db "github.com/p1xart/anonchat/internal/database"
	"sync"
)

// Общие переменные
type BotVars struct {
	DB     *sql.DB
	Lock   sync.Mutex
	Router map[int64]int64
	RouterFilter []User
	Online int
	Pool   []int64
	EditReplyMarkup map[int64]int
	UpdReplyMarkup map[int64]tgbotapi.InlineKeyboardMarkup

}

// Машина состояний (FSM)
type State struct {
	Lock                sync.Mutex
	CurrentStateAge     map[int64]string
	CurrentStateSex     map[int64]string
	CurrentStateImprove map[int64]string
	CurrentStateDeal    map[int64]string
	CurrentStateFilter  map[int64]string
}

// Структура пользователя
type User struct {
	ID         uint64
	Age        uint8
	Gender     string
	AgeFind    uint8
	GenderFind uint8
}


var Vars BotVars = BotVars{
	DB:     db.ConnectBase(),
	Lock:   sync.Mutex{},
	Router: make(map[int64]int64),
	RouterFilter: make([]User, 0, 5000),
	Online: 0,
	Pool:   make([]int64, 0),
	EditReplyMarkup: make(map[int64]int),
	UpdReplyMarkup: make(map[int64]tgbotapi.InlineKeyboardMarkup),
}

var Fsm State = State{
	Lock:                sync.Mutex{},
	CurrentStateAge:     make(map[int64]string),
	CurrentStateSex:     make(map[int64]string),
	CurrentStateImprove: make(map[int64]string),
	CurrentStateDeal:    make(map[int64]string),
	CurrentStateFilter:  make(map[int64]string),
}

// Inline клавиатура начала поиска
var SearchKb = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Начать поиск")))
var StopKb = tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Остановить поиск")))
