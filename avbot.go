package avbot

import (
	"log"
	"net"
	"net/http"

	"golang.org/x/net/proxy"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type AVBot struct {
	*tgbotapi.BotAPI
	hooks       []MessageHook
	client      *http.Client
	groupChatId int64
}

func (b *AVBot) AddMessageHook(hook MessageHook) {
	b.hooks = append(b.hooks, hook)
}

func NewBot(token string, chatId int64, socks5Addr string) *AVBot {
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
		BotAPI:      bot,
		hooks:       make([]MessageHook, 0, 0),
		client:      client,
		groupChatId: chatId,
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
		if update.Message != nil {
			b.onMessage(update.Message)
		}
	}
}

func (b *AVBot) onMessage(msg *tgbotapi.Message) {

	for _, h := range b.hooks {
		if h.Process(b, msg) {
			break
		}
	}
}

func (b *AVBot) GetBotApi() *tgbotapi.BotAPI {
	return b.BotAPI
}

func (b *AVBot) GetGroupChatId() int64 {
	return b.groupChatId
}
