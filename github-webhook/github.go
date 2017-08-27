package github

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	avbot "github.com/hyqhyq3/avbot-telegram"
)

type GithubHook struct {
	server *http.Server
	bot    *avbot.AVBot
	sendCh chan<- *avbot.MessageInfo
}

func New(bot *avbot.AVBot, addr string) (ret *GithubHook) {
	ret = &GithubHook{}
	ret.bot = bot
	ret.server = &http.Server{
		Addr:    addr,
		Handler: ret,
	}
	go ret.server.ListenAndServe()
	return
}

type Commit struct {
	Message  string
	Added    []string
	Removed  []string
	Modified []string
}

type Event struct {
	Action string
	Sender struct {
		Login string
	}
	Commits    []Commit
	Repository struct {
		Name     string
		FullName string `json:"full_name"`
	}
}

func getCommitDesc(commits []Commit) string {
	comments := ""
	for _, v := range commits {
		comments += v.Message + "\n-----------------------\n"
	}
	return comments
}

func (h *GithubHook) GetName() string {
	return "Github Webhook"
}

func (h *GithubHook) SetSendMessageChannel(ch chan *avbot.MessageInfo) {
	h.sendCh = ch
}

func (h *GithubHook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/github/avplayer" {
		name := r.Header.Get("X-GitHub-Event")
		content, _ := ioutil.ReadAll(r.Body)
		evt := &Event{}
		err := json.Unmarshal(content, evt)
		if err == nil {
			var desc string
			switch name {
			case "repository":
				desc = "repository " + evt.Repository.FullName
			case "push":
				desc = "push repository " + evt.Repository.FullName + "\n" + getCommitDesc(evt.Commits)
			default:
				log.Println("Github event", name)
				w.Write([]byte("OK"))
				return
			}
			m := avbot.NewTextMessage(h, string(evt.Sender.Login+" "+evt.Action+" "+desc))
			h.sendCh <- m
		}
	}
	w.Write([]byte("OK"))
}
