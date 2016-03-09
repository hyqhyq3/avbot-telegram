package joke

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"gopkg.in/telegram-bot-api.v2"
)

var url = "http://www.qiushibaike.com/8hr/page/%d"

type JokeHook struct {
	oldJokes []int
}

func New() *JokeHook {
	return &JokeHook{
		oldJokes: make([]int, 0, 0),
	}
}

func (h *JokeHook) Process(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) (processed bool) {
	if strings.Contains(msg.Text, "大爷") && strings.Contains(msg.Text, "笑话") {
		j := h.getJoke()
		m := tgbotapi.NewMessage(msg.Chat.ID, j.Text)
		m.ReplyToMessageID = msg.MessageID
		bot.Send(m)
		return true
	}
	return false
}

type Joke struct {
	ID   int
	Text string
}

func (h *JokeHook) getJoke() *Joke {
	for {
		doc, err := goquery.NewDocument(fmt.Sprintf(url, rand.Int()%30))
		if err != nil {
			log.Println("joke.getJoke", err)
			return &Joke{ID: -1, Text: "获取笑话出错"}
		}
		arr := doc.Find(".article")

		n := arr.Nodes[rand.Int()%arr.Length()]
		var id int
		for _, a := range n.Attr {
			if a.Key == "id" {
				pos := strings.IndexAny(a.Val, "0123456789")
				id, _ = strconv.Atoi(a.Val[pos:])
			}
		}
		for _, i := range h.oldJokes {
			if i == id {
				continue
			}
		}
		h.oldJokes = append(h.oldJokes, id)
		t := strings.TrimSpace(goquery.NewDocumentFromNode(n).Find("div.content").Text())
		return &Joke{id, t}
	}
}
