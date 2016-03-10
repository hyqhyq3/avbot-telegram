package hello

import (
	"bytes"
	"text/template"

	"gopkg.in/telegram-bot-api.v2"
)

type HelloHook struct {
	*template.Template
}

func New(str string) *HelloHook {
	tpl := template.New("")
	return &HelloHook{template.Must(tpl.Parse(str))}
}

func (h *HelloHook) Process(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) (processed bool) {
	if msg.NewChatParticipant.ID != 0 {
		b := &bytes.Buffer{}
		h.Execute(b, map[string]string{"Name": msg.NewChatParticipant.UserName})
		m := tgbotapi.NewMessage(msg.Chat.ID, b.String())
		bot.Send(m)
	}
	return false
}
