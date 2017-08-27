package hello

import (
	"text/template"

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
	// b := &bytes.Buffer{}
	// h.Execute(b, map[string]string{"UserName": msg.From})

	// bot.SendMessage(avbot.NewTextMessage(h, b.String()))

	return false
}
