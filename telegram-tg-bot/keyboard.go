package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("1-1"),
		tgbotapi.NewKeyboardButton("1-2"),
		tgbotapi.NewKeyboardButton("1-3"),
		tgbotapi.NewKeyboardButtonLocation("location button"),
		tgbotapi.NewKeyboardButtonContact("contact button"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("2-1"),
		tgbotapi.NewKeyboardButton("2-2"),
		tgbotapi.NewKeyboardButton("2-3"),
	),
)

func init() {
	os.Setenv("HTTP_PROXY", "socks5://127.0.0.1:1080")
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:1080")
}

func main() {
	bot, err := tgbotapi.NewBotAPI("my-tg-id:my-tg-key")
	if err != nil {
		log.Panic(err)
	}

	// bot.Debug = true

	log.Println("Authorized on bot", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for ev := range updates {
		var (
			msg = ev.Message
		)

		if msg == nil {
			continue
		}

		fmt.Println(msg.From.UserName, ":", msg.Text)

		reply := tgbotapi.NewMessage(msg.Chat.ID, msg.Text)

		switch strings.ToLower(msg.Text) {
		case "open":
			reply.ReplyMarkup = numericKeyboard
		case "close":
			reply.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		}

		bot.Send(reply)
	}
}
