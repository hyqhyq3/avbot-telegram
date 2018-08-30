package question

import (
	"strconv"
	"time"

	"github.com/hyqhyq3/avbot-telegram"
	"github.com/hyqhyq3/avbot-telegram/data"
	"github.com/hyqhyq3/avbot-telegram/telegram"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type Member struct {
	ChatId int64
	Uid    int64
}

type Hook struct {
	telegram      *telegram.Telegram
	answered      map[int64]bool
	waitForAnswer map[int64]bool
}

func New(tg *telegram.Telegram) *Hook {
	return &Hook{
		telegram:      tg,
		answered:      make(map[int64]bool),
		waitForAnswer: make(map[int64]bool),
	}
}

func (h *Hook) GetName() string {
	return "question"
}

func (h *Hook) Process(bot *avbot.AVBot, msg *avbot.MessageInfo) bool {
	if msg.Type == data.MessageType_NEW_MEMBER {

		h.waitForAnswer[msg.UID] = true
		bot.SendMessage(avbot.NewTextMessage(h, "["+msg.From+"](tg://user?id="+strconv.FormatInt(msg.UID, 10)+")"+` 请回答问题 printf("%#x", 65535); 的输出是多少？（你有60秒时间作答）`))
		go func() {
			<-time.After(time.Second * 60)
			if b, ok := h.answered[msg.UID]; !b || !ok {
				config := tgbotapi.KickChatMemberConfig{}
				config.ChatID = h.telegram.ChatID
				config.UserID = int(msg.UID)
				config.UntilDate = time.Now().Unix() + 60
				h.telegram.KickChatMember(config)
			}
		}()
	}
	if a, ok := h.waitForAnswer[msg.UID]; a && ok {
		if msg.Content == "0xffff" {

			m := tgbotapi.NewMessage(h.telegram.ChatID, "回答正确")
			m.ReplyToMessageID = msg.MessageID
			h.telegram.Send(m)

			h.answered[msg.UID] = true
			delete(h.waitForAnswer, msg.UID)
		} else if msg.Content != "" {
			m := tgbotapi.NewMessage(h.telegram.ChatID, "回答错误")
			m.ReplyToMessageID = msg.MessageID
			h.telegram.Send(m)
		}
	}
	return false
}
