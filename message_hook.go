package avbot

import "gopkg.in/telegram-bot-api.v2"

type MessgaeHook interface {
	Process(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) (processed bool)
}
