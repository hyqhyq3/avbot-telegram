package stat

//go:generate protoc data.proto --go_out=.

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	avbot "github.com/hyqhyq3/avbot-telegram"
)

type StatHook struct {
	filename string
	Changed  bool
	closeCh  chan int
	store    *Store
	sendCh   chan<- *avbot.MessageInfo
}

func New(filename string) (h *StatHook) {

	h = &StatHook{}
	h.filename = filename

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
		return
	}

	store := &Store{}
	err = proto.Unmarshal(b, store)
	if err != nil {
		log.Fatal(err)
		return
	}

	if store.Users == nil {
		store.Users = make(map[string]*User)
	}

	h.store = store
	h.closeCh = make(chan int)

	go h.StartLoop()

	return
}

func (h *StatHook) GetName() string {
	return "Stat"
}

func (h *StatHook) StartLoop() {
mainLoop:
	for {
		select {
		case <-time.After(time.Second * 60):
			h.Save(false)
		case <-h.closeCh:
			break mainLoop
		}
	}
}

func (h *StatHook) Process(bot *avbot.AVBot, msg *avbot.MessageInfo) (processed bool) {
	if msg != nil {
		h.Inc(msg.From)
		h.Changed = true
	}
	cmd := strings.Split(msg.Content, " ")
	cmd = strings.Split(cmd[0], "@")
	if cmd[0] == "/stat" {
		mymsg := avbot.NewTextMessage(h, h.GetStat())

		bot.SendMessage(mymsg)
	}
	return false
}

type Users []*User

func (u Users) Swap(i, j int) {
	t := u[i]
	u[i] = u[j]
	u[j] = t
}

func (u Users) Less(i, j int) bool {
	return u[i].Count > u[j].Count
}

func (u Users) Len() int {
	return len(u)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h *StatHook) GetStat() string {
	data := make([]*User, 0)
	for _, v := range h.store.Users {
		data = append(data, v)
	}
	sort.Sort(Users(data))

	var str = ""
	for i := 0; i < min(10, len(data)); i++ {
		str = str + fmt.Sprintf("%s: %d\n", data[i].UserName, data[i].Count)
	}
	return str
}

func (h *StatHook) Inc(user string) {
	if _, ok := h.store.Users[user]; !ok {
		h.store.Users[user] = &User{}
	}
	h.store.Users[user].UserName = user
	h.store.Users[user].Count++
}

func (h *StatHook) Save(force bool) {
	if !h.Changed && !force {
		return
	}

	b, err := proto.Marshal(h.store)
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(h.filename, b, 0755)
	if err != nil {
		log.Println(err)
		return
	}
	h.Changed = false
	log.Println("save stat data")
}

func (h *StatHook) Stop() {
	h.closeCh <- 1
	h.Save(true)
}
