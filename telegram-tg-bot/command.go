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
			case "help":
				reply.Text = "type /sayhi or /status."
			case "sayhi":
				reply.Text = "Hi :)"
			case "status":
				reply.Text = "I'm ok."
			default:
				reply.Text = "I don't know that command"
			}
			bot.Send(reply)
		}
	}
}
