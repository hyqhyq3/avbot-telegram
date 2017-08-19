package joke

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/PuerkitoBio/goquery"
	avbot "github.com/hyqhyq3/avbot-telegram"
	"gopkg.in/telegram-bot-api.v4"
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

func (h *JokeHook) Process(bot *avbot.AVBot, msg *tgbotapi.Message) (processed bool) {
	if strings.Contains(msg.Text, "大爷") && strings.Contains(msg.Text, "笑话") {
		j := h.getJoke()
		m := tgbotapi.NewMessage(msg.Chat.ID, j.Text)
		m.ReplyToMessageID = msg.MessageID
		bot.Send(m)

		if j.Image != nil {
			m := tgbotapi.NewPhotoUpload(msg.Chat.ID, *j.Image)
			bot.Send(m)
		}

		return true
	}
	return false
}

type Joke struct {
	ID    int
	Text  string
	Image *tgbotapi.FileBytes
}

func (h *JokeHook) getJoke() (j *Joke) {
	defer func() {
		e := recover()
		if e != nil {
			log.Println("joke.getJoke", e)
		}
	}()
	j = &Joke{}
NextJoke:
	for {
		u := fmt.Sprintf(url, rand.Int()%30)
		doc, err := goquery.NewDocument(u)
		if err != nil {
			log.Println("joke.getJoke", err)
			return &Joke{ID: -1, Text: "获取笑话出错"}
		}
		arr := doc.Find(".article")

		n := arr.Nodes[rand.Int()%arr.Length()]
		for _, a := range n.Attr {
			if a.Key == "id" {
				pos := strings.IndexAny(a.Val, "0123456789")
				j.ID, _ = strconv.Atoi(a.Val[pos:])
			}
		}
		for _, i := range h.oldJokes {
			if i == j.ID {
				continue NextJoke
			}
		}
		h.oldJokes = append(h.oldJokes, j.ID)
		content := goquery.NewDocumentFromNode(n).Find("div.content")
		j.Text = strings.TrimSpace(content.Text())

		if imgs := goquery.NewDocumentFromNode(n).Find("div.thumb").Find("img"); imgs.Length() > 0 {
			for _, a := range imgs.Nodes[0].Attr {
				if a.Key == "src" {
					r, _ := http.NewRequest("GET", a.Val, nil)
					r.Header.Set("Referer", u)
					resp, err := http.DefaultClient.Do(r)
					if err != nil {
						break
					}
					b := &bytes.Buffer{}
					io.Copy(b, resp.Body)
					resp.Body.Close()
					j.Image = &tgbotapi.FileBytes{Name: "test.jpg", Bytes: b.Bytes()}
					break
				}
			}
		}

		return
	}
}
