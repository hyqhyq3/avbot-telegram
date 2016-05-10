package stat

//go:generate protoc data.proto --go_out=.

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"

	"gopkg.in/telegram-bot-api.v2"
)

type StatHook struct {
	filename string
	Groups   map[int32]*Group
}

func New(filename string) (h *StatHook) {

	h = &StatHook{}
	h.Groups = make(map[int32]*Group)
	h.filename = filename

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return
	}

	store := &Store{}
	err = proto.Unmarshal(b, store)
	if err != nil {
		log.Println(err)
		return
	}

	h.Groups = store.Groups

	fmt.Println(h)

	return
}

func (h *StatHook) Process(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) (processed bool) {
	if msg != nil {
		h.Inc(&msg.Chat, &msg.From)
		h.Save()
	}
	cmd := strings.Split(msg.Text, " ")
	if cmd[0] == "/stat" {
		mymsg := tgbotapi.NewMessage(msg.Chat.ID, h.GetStat(msg.Chat.ID))
		bot.Send(mymsg)
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

func (h *StatHook) GetStat(id int) string {
	data := make([]*User, 0)
	for _, v := range h.Groups[int32(id)].Users {
		data = append(data, v)
	}
	sort.Sort(Users(data))

	var str = ""
	for i := 0; i < min(10, len(data)); i++ {
		str = str + fmt.Sprintf("%s: %d\n", data[i].UserName, data[i].Count)
	}
	return str
}

func (h *StatHook) Inc(chat *tgbotapi.Chat, user *tgbotapi.User) {
	var chatid = int32(chat.ID)
	var uid = int32(user.ID)
	if _, ok := h.Groups[chatid]; !ok {
		h.Groups[chatid] = &Group{Users: make(map[int32]*User)}
	}
	if _, ok := h.Groups[chatid].Users[uid]; !ok {
		h.Groups[chatid].Users[uid] = &User{}
	}
	h.Groups[chatid].Users[uid].FirstName = user.FirstName
	h.Groups[chatid].Users[uid].LastName = user.LastName
	h.Groups[chatid].Users[uid].UserName = user.UserName
	h.Groups[chatid].Users[uid].Count++
}

func (h *StatHook) Save() {
	store := &Store{}
	store.Groups = h.Groups

	b, err := proto.Marshal(store)
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(h.filename, b, 0755)
	if err != nil {
		log.Println(err)
	}
}
