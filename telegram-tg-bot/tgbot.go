package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/net/proxy"
)

var (
	socks5Proxy = "127.0.0.1:1080"
	tgToken     = "my_tg_id:my_tg_token"
	bot         *tgbot
)

func main() {
	var err error
	bot, err = newTGBot(tgToken)
	if err != nil {
		log.Fatalln(err)
	}

	select {}
}

var commandKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("来个新节点"),
		tgbotapi.NewKeyboardButton("节点列表"),
		tgbotapi.NewKeyboardButton("删掉节点"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("唠会?"),
		tgbotapi.NewKeyboardButton("再见!"),
	),
)

type tgbot struct {
	sync.RWMutex                  // protect the followings
	api          *tgbotapi.BotAPI // tg bot api
	running      bool             // flag if running
	errmsg       string           // startup error message
	stopCh       chan struct{}    // stop notify channel
}

func newTGBot(token string) (*tgbot, error) {
	if token == "" {
		return nil, errors.New("bot token required")
	}

	dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	client := insecureHTTPClient()
	client.Transport.(*http.Transport).Dial = dialer.Dial
	botapi, err := tgbotapi.NewBotAPIWithClient(token, client)
	if err != nil {
		return nil, fmt.Errorf("bot token %s met error: %v", token, err)
	}

	bot = &tgbot{
		api:     botapi,
		running: false,
		errmsg:  "",
		stopCh:  make(chan struct{}),
	}

	go bot.run()
	return bot, nil
}

func (b *tgbot) name() string {
	b.RLock()
	defer b.RUnlock()
	if b.api == nil {
		return "" // no name means the tgbot not initialized
	}
	return b.api.Self.String()
}

func (b *tgbot) token() string {
	b.RLock()
	defer b.RUnlock()
	if b.api == nil {
		return "" // no token means the tgbot not initialized
	}
	return b.api.Token
}

// run start the tg bot monitor loop
func (b *tgbot) run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := b.api.GetUpdatesChan(u)
	if err != nil {
		b.errmsg = err.Error()
		log.Printf("telegram bot %s subscribe updates error: %s", b.name(), err.Error())
		return
	}

	b.running = true
	b.errmsg = ""
	log.Printf("telegram bot %s started", b.name())

	defer func() {
		b.running = false
		log.Printf("telegram bot %s stopped", b.name())
	}()

	var mode string
	var talks = []string{"嗯嗯", "我啥都没听见", "哦"}

	for {
		select {

		case ev := <-updates:
			var (
				msg = ev.Message
			)
			if msg == nil {
				continue
			}

			log.Println("tg message:", msg.From.UserName, ":", msg.Text)

			var (
				reply = tgbotapi.NewMessage(msg.Chat.ID, "")
			)

			// handle command message
			if msg.IsCommand() {
				switch msg.Command() {
				case "hi", "hello":
					reply.Text = "嗨不嗨!"
				case "bye":
					reply.Text = "好走不送!"
				case "help":
					reply.Text = "看看这菜单合不合您胃口:"
					reply.ReplyMarkup = commandKeyboard
				default:
					reply.Text = "不懂你说啥嘞！试试 /help"
				}

				b.api.Send(reply)
				continue
			}

			// handle generic message
			switch strings.ToLower(msg.Text) {
			case "来个新节点":
				reply.Text = "稍等,创建中...好了第一时间通知您!"
				b.api.Send(reply)

			case "删掉节点":
				reply.Text = "删无可删"
				b.api.Send(reply)

			case "节点列表":
				reply.Text = "其实啥都没有"
				b.api.Send(reply)

			case "唠会?":
				mode = "talk"
				reply.Text = "来吧,你想说点啥?"
				b.api.Send(reply)

			case "再见!":
				mode = ""
				reply.Text = "回见!"
				reply.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				b.api.Send(reply)

			default: // keep quiet
				if mode == "talk" {
					reply.Text = talks[rand.Intn(len(talks))]
				} else {
					reply.Text = "不懂你说啥嘞！试试 /help"
				}
				b.api.Send(reply)

			}

		case <-b.stopCh:
			return
		}
	}
}

// note: concurrency safe for stop() call
func (b *tgbot) stop() {
	b.Lock()
	defer b.Unlock()

	if b.api != nil { // prevent panic if first startup
		b.api.StopReceivingUpdates() // stop tg inner goroutine to receive updates
	}

	select {
	case b.stopCh <- struct{}{}: // prevent block
	default:
	}
}

func (b *tgbot) status() map[string]interface{} {
	return map[string]interface{}{
		"name":    b.name(),
		"running": b.running,
		"errmsg":  b.errmsg,
	}
}

var (
	once sync.Once
	cli  *http.Client
)

func insecureHTTPClient() *http.Client {
	once.Do(func() {
		cli = &http.Client{
			Timeout: time.Second * 180,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	})
	return cli
}
