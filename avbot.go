package avbot

import (
	"log"
	"net"
	"net/http"

	"golang.org/x/net/proxy"
	"gopkg.in/telegram-bot-api.v2"
)

type AVBot struct {
	*tgbotapi.BotAPI
	hooks  []MessgaeHook
	client *http.Client
}

func (b *AVBot) AddMessageHook(hook MessgaeHook) {
	b.hooks = append(b.hooks, hook)
}

func NewBot(token string, socks5Addr string) *AVBot {
	dial := net.Dial
	if socks5Addr != "" {
		dialer, err := proxy.SOCKS5("tcp", socks5Addr, nil, proxy.Direct)
		if err != nil {
			panic(err)
		}
		dial = dialer.Dial
	}
	client := &http.Client{
		Transport: &http.Transport{
			Dial: dial,
		},
	}

	bot, err := tgbotapi.NewBotAPIWithClient(token, client)
	if err != nil {
		panic(err)
	}
	return &AVBot{
		BotAPI: bot,
		hooks:  make([]MessgaeHook, 0, 0),
		client: client,
	}
}

func (b *AVBot) Run() {
	b.Debug = true

	log.Printf("Authorized on account %s", b.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := b.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}

	for update := range updates {
		b.onMessage(&update.Message)
	}
}

func (b *AVBot) onMessage(msg *tgbotapi.Message) {

	for _, h := range b.hooks {
		if h.Process(b.BotAPI, msg) {
			break
		}
	}
}

func (b *AVBot) GetBotApi() *tgbotapi.BotAPI {
	return b.BotAPI
}
