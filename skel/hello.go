package hello

import (
	"bytes"
	"text/template"

	avbot "github.com/hyqhyq3/avbot-telegram"
	"gopkg.in/telegram-bot-api.v4"
)

type HelloHook struct {
	*template.Template
}

func New(str string) *HelloHook {
	tpl := template.New("")
	return &HelloHook{template.Must(tpl.Parse(str))}
}

func (h *HelloHook) Process(bot *avbot.AVBot, msg *tgbotapi.Message) (processed bool) {
	if msg.NewChatMember != nil {
		b := &bytes.Buffer{}
		h.Execute(b, map[string]string{"UserName": msg.NewChatMember.UserName, "FirstName": msg.NewChatMember.FirstName})
		m := tgbotapi.NewMessage(msg.Chat.ID, b.String())
		bot.Send(m)
	}
	return false
}
