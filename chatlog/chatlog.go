package chatlog

import (
	"encoding/binary"
	"log"

	"github.com/golang/protobuf/proto"

	avbot "github.com/hyqhyq3/avbot-telegram"
	"github.com/hyqhyq3/avbot-telegram/data"
	"github.com/hyqhyq3/avbot-telegram/store"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

//go:generate protoc data.proto --go_out=.

type ChatLogHook struct {
	db *leveldb.DB
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
	return
}

func (h *ChatLogHook) GetName() string {
	return "ChatLog"
}

func (h *ChatLogHook) AddLog(l *data.Message) {

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
}

func (h *ChatLogHook) Stop() {
	if h.db != nil {
		h.db.Close()
	}
}

func (h *ChatLogHook) Get(from, to uint64) []*data.Message {
	i := make([]byte, 8)
	j := make([]byte, 8)
	binary.LittleEndian.PutUint64(i, from)
	binary.LittleEndian.PutUint64(j, to)
	logs := make([]*data.Message, 0, 100)
	iter := h.db.NewIterator(&util.Range{Start: i, Limit: j}, nil)
	for iter.Next() {
		p := iter.Value()
		msg := &data.Message{}
		proto.Unmarshal(p, msg)

		logs = append(logs, msg)
		log.Println(msg)
	}
	iter.Release()
	return logs
}

func (h *ChatLogHook) Last(num uint64) []*data.Message {
	var from uint64
	var index = store.GetStore().MessageIDIndex
	if index > num {
		from = store.GetStore().MessageIDIndex - num
	} else {
		from = 0
	}
	return h.Get(from, index)
}

func (h *ChatLogHook) Process(bot *avbot.AVBot, msg *avbot.MessageInfo) (processed bool) {
	h.AddLog(msg.Message)
	return false
}
