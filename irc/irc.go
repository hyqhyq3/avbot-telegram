package irc

import (
	"fmt"

	avbot "github.com/hyqhyq3/avbot-telegram"
	"gopkg.in/sorcix/irc.v1"
	"gopkg.in/telegram-bot-api.v4"
)

type JokeHook struct {
	Irc    *irc.Conn
	bot    *avbot.AVBot
	chatId int64
	ch     string
	nick   string
}

func New(bot *avbot.AVBot, ch, nick string) *JokeHook {
	return &JokeHook{
		bot:    bot,
		ch:     ch,
		nick:   nick,
		chatId: 0,
	}
}

func (h *JokeHook) Process(bot *avbot.AVBot, msg *tgbotapi.Message) (processed bool) {
	h.SendToIrc(msg.From.FirstName + ":" + msg.Text)
	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		h.chatId = msg.Chat.ID
	}
	return false
}

func (h *JokeHook) SendToIrc(text string) {
	if h.Irc == nil {
		h.ConnectToIrc()
	}
	if h.Irc != nil {
		msg := &irc.Message{Command: irc.PRIVMSG, Params: []string{h.ch}, Trailing: text}
		h.Irc.Encode(msg)
	}
}

func (h *JokeHook) SendToTg(text string) {
	if h.chatId != 0 {
		msg := tgbotapi.NewMessage(h.chatId, text)
		h.bot.Send(msg)
	}
}

func (h *JokeHook) ConnectToIrc() {
	fmt.Println("connect to irc")
	c, err := irc.Dial("chat.freenode.net:6667")
	if err != nil {
		fmt.Println(err)
		return
	}
	msg := &irc.Message{Command: irc.PASS, Params: []string{h.nick}}
	c.Encode(msg)

	msg = &irc.Message{Command: irc.USER, Params: []string{"guest", "0", "*", ":" + h.nick}}
	c.Encode(msg)

	msg = &irc.Message{Command: irc.NICK, Params: []string{h.nick}}
	c.Encode(msg)

	msg = &irc.Message{Command: irc.JOIN, Params: []string{h.ch}}
	c.Encode(msg)
	h.Irc = c

	go h.HandleIrc()
}

func (h *JokeHook) HandleIrc() {
	for {
		msg, err := h.Irc.Decode()
		if err != nil {
			h.Irc = nil
			break
		}
		if msg.Command == irc.PING {
			h.Irc.Encode(&irc.Message{Command: irc.PONG, Params: msg.Params})
		}
		if msg.Command == irc.PRIVMSG && len(msg.Params) >= 1 && msg.Params[0] == h.ch {
			h.SendToTg(msg.Name + ":" + msg.Trailing)
		}
	}
}
