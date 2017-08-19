package avbot

import "gopkg.in/telegram-bot-api.v4"

type MessgaeHook interface {
	Process(bot *AVBot, msg *tgbotapi.Message) (processed bool)
}
