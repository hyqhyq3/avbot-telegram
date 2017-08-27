package chatlog

import (
	"encoding/binary"
	"log"
	"sync/atomic"

	"github.com/golang/protobuf/proto"

	avbot "github.com/hyqhyq3/avbot-telegram"
	"github.com/hyqhyq3/avbot-telegram/data"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
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

func (h *ChatLogHook) GetName() string {
	return "ChatLog"
}

func (h *ChatLogHook) AddLog(l *data.Message) {
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
	if h.index > num {
		from = h.index - num
	} else {
		from = 0
	}
	return h.Get(from, h.index)
}

func (h *ChatLogHook) Process(bot *avbot.AVBot, msg *avbot.MessageInfo) (processed bool) {
	h.AddLog(msg.Message)
	return false
}
