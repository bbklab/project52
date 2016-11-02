package main

import (
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
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

	// Optional: wait for updates and clear them if you don't want to handle
	// a large backlog of old messages
	// time.Sleep(time.Millisecond * 500)
	// updates.Clear()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for ev := range updates {
		msg := ev.Message
		if msg == nil {
			continue
		}

		fmt.Println(msg.From.UserName, ":", msg.Text)

		reply := tgbotapi.NewMessage(msg.Chat.ID, msg.Text+"...") // reply to this chat with original message text + suffix ...
		// reply.ReplyToMessageID = msg.MessageID                    // reply to this message
		bot.Send(reply)
	}
}
