package hello

import (
	"bytes"
	"text/template"

	"github.com/hyqhyq3/avbot-telegram/data"

	avbot "github.com/hyqhyq3/avbot-telegram"
)

type HelloHook struct {
	*template.Template
}

func New(str string) *HelloHook {
	tpl := template.New("")
	return &HelloHook{template.Must(tpl.Parse(str))}
}

func (h *HelloHook) GetName() string {
	return "Welcome"
}

func (h *HelloHook) Process(bot *avbot.AVBot, msg *avbot.MessageInfo) (processed bool) {
	if msg.Type == data.MessageType_NEW_MEMBER {
		b := &bytes.Buffer{}
		h.Execute(b, map[string]string{"UserName": msg.From})

		bot.SendMessage(avbot.NewTextMessage(h, b.String()))
	}

	return false
}
