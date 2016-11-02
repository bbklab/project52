package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonURL("g.cn", "http://g.cn"),
		tgbotapi.NewInlineKeyboardButtonURL("sina.com.cn", "https://sina.com.cn"),
		tgbotapi.NewInlineKeyboardButtonData("3", "3"),
		tgbotapi.NewInlineKeyboardButtonData("4", "4"),
		tgbotapi.NewInlineKeyboardButtonData("5", "5"),
		tgbotapi.NewInlineKeyboardButtonData("6", "6"),
		tgbotapi.NewInlineKeyboardButtonData("7", "7"),
	),
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("2-1", "2.1"),
		tgbotapi.NewInlineKeyboardButtonData("2-2", "2.2"),
		tgbotapi.NewInlineKeyboardButtonData("2-3", "2.3"),
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
			cbquery = ev.CallbackQuery
			msg     = ev.Message
		)

		if cbquery != nil {
			fmt.Println("get call back query", cbquery)
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(cbquery.ID, cbquery.Data))
			bot.Send(tgbotapi.NewMessage(cbquery.Message.Chat.ID, fmt.Sprintf("You pressed [%s]", cbquery.Data)))
		}

		if msg != nil {
			fmt.Println(msg.From.UserName, ":", msg.Text)
			switch strings.ToLower(msg.Text) {
			case "open":
				reply := tgbotapi.NewMessage(msg.Chat.ID, "try following buttons")
				reply.ReplyMarkup = numericKeyboard // with inline keyboard
				bot.Send(reply)
			}
		}

	}
}
