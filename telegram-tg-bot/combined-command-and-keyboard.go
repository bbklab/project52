package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-telegram-bot-api/telegram-bot-api"
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

	//bot.Debug = true

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

		if msg.IsCommand() {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "")
			switch msg.Command() {
			case "hi":
				reply.Text = "Hi :)"
			case "help":
				reply.Text = "pls choose a command list below:"
				reply.ReplyMarkup = commandKeyboard
			default:
				reply.Text = "I don't know that command, try /help"
			}

			bot.Send(reply)
		}
	}
}

var commandKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("/version"),
		tgbotapi.NewKeyboardButton("/license"),
		tgbotapi.NewKeyboardButton("/setting"),
		tgbotapi.NewKeyboardButton("/summary"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButtonLocation("location"),
		tgbotapi.NewKeyboardButtonContact("contact"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("/more"),
	),
)
