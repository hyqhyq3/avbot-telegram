package avbot

import "gopkg.in/telegram-bot-api.v4"

type MessageHook interface {
	Process(bot *AVBot, msg *tgbotapi.Message) (processed bool)
}

type Stoppable interface {
	Stop()
}
