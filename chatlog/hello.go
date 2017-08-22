package chatlog

import (
	"log"
	"strconv"
	"sync/atomic"

	"github.com/golang/protobuf/proto"

	avbot "github.com/hyqhyq3/avbot-telegram"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/telegram-bot-api.v4"
)

//go:generate protoc data.proto --go_out=.

type ChatLogHook struct {
	db    *leveldb.DB
	index uint64
}

var instance *ChatLogHook

func newChatLog() *ChatLogHook {
	chatLog := &ChatLogHook{}
	return chatLog
}

func GetInstance() *ChatLogHook {
	if instance == nil {
		instance = newChatLog()
	}
	return instance
}

func (h *ChatLogHook) Init(filepath string) (err error) {
	h.db, err = leveldb.OpenFile(filepath, nil)
	if err != nil {
		return
	}
	b, err := h.db.Get([]byte("_index"), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return
	}
	if err != nil {
		h.index = 0
	} else {
		h.index, err = strconv.ParseUint(string(b), 10, 64)
		if err != nil {
			h.index = 0
		}
	}
	return
}

func (h *ChatLogHook) AddLog(l *ChatLog) {
	id := atomic.AddUint64(&h.index, 1)
	l.MessageId = id

	b, err := proto.Marshal(l)
	if err != nil {
		log.Println(err)
		return
	}

	err = h.db.Put([]byte(strconv.FormatUint(l.MessageId, 10)), b, nil)
	if err != nil {
		log.Println(err)
		return
	}
}

func (h *ChatLogHook) Stop() {
	if h.db != nil {
		h.db.Close()
	}
}

func (h *ChatLogHook) Process(bot *avbot.AVBot, msg *tgbotapi.Message) (processed bool) {

	switch {
	case msg.Text != "":
		log := &ChatLog{
			Content: msg.Text,
			From:    msg.From.FirstName,
			Type:    MessageType_TEXT,
		}

		h.AddLog(log)
	case msg.Photo != nil:
	}
	return false
}
