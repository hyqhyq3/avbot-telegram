package chatlog

import (
	"encoding/binary"
	"log"
	"sync/atomic"

	"github.com/golang/protobuf/proto"

	avbot "github.com/hyqhyq3/avbot-telegram"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
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
		if len(b) == 8 {
			h.index = binary.LittleEndian.Uint64(b)
		} else {
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

	i := make([]byte, 8)
	binary.LittleEndian.PutUint64(i, l.MessageId)
	err = h.db.Put(i, b, nil)
	if err != nil {
		log.Println(err)
		return
	}

	binary.LittleEndian.PutUint64(i, h.index)

	h.db.Put([]byte("_index"), i, nil)
}

func (h *ChatLogHook) Stop() {
	if h.db != nil {
		h.db.Close()
	}
}

func (h *ChatLogHook) Get(from, to uint64) []*ChatLog {
	i := make([]byte, 8)
	j := make([]byte, 8)
	binary.LittleEndian.PutUint64(i, from)
	binary.LittleEndian.PutUint64(j, to)
	logs := make([]*ChatLog, 0, 100)
	iter := h.db.NewIterator(&util.Range{Start: i, Limit: j}, nil)
	for iter.Next() {
		data := iter.Value()
		msg := &ChatLog{}
		proto.Unmarshal(data, msg)

		logs = append(logs, msg)
		log.Println(msg)
	}
	iter.Release()
	return logs
}

func (h *ChatLogHook) Process(bot *avbot.AVBot, msg *tgbotapi.Message) (processed bool) {

	switch {
	case msg.Text != "":
		log := &ChatLog{
			Content:   msg.Text,
			From:      msg.From.FirstName,
			Type:      MessageType_TEXT,
			Timestamp: msg.Time().Unix(),
		}

		h.AddLog(log)
	case msg.Photo != nil:
	}
	return false
}
