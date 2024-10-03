package app

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	chat "github.com/p1xart/anonchat/internal/services/chat"
	"log"
)

func Run() {
	bot, err := tgbotapi.NewBotAPI("6693384071:AAEmSkxwx6R_d6PEHgWkMn6Ux6nb28ij4Bc")
	if err != nil {
		log.Fatal(err)
	}
	bot.Debug = false
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	chat.Chat(bot, updates)
}
