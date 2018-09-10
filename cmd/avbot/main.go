package main

import (
	"flag"
	"io/ioutil"

	"github.com/go-yaml/yaml"
	"github.com/hyqhyq3/avbot-telegram"
	"github.com/hyqhyq3/avbot-telegram/chatlog"
	github "github.com/hyqhyq3/avbot-telegram/github-webhook"
	// "github.com/hyqhyq3/avbot-telegram/hello"
	"github.com/hyqhyq3/avbot-telegram/irc"
	"github.com/hyqhyq3/avbot-telegram/stat"
	"github.com/hyqhyq3/avbot-telegram/telegram"
	"github.com/hyqhyq3/avbot-telegram/ws"
	"github.com/hyqhyq3/avbot-telegram/question"
)

type Config struct {
	Secret string

	GroupChatID int64

	Welcome string

	Github struct {
		Listen string
	}

	WebSocket struct {
		Port int
	}

	Proxy struct {
		Socks5 string
	}

	ChatLog struct {
		Path string
	}
}

var config Config

func init() {

	var configFile string
	flag.StringVar(&configFile, "c", "avbot.yaml", "config file")
	flag.Parse()
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	yaml.Unmarshal(data, &config)

}

func main() {

	token := config.Secret
	bot := avbot.NewBot()
	tg := telegram.New(token, config.Proxy.Socks5, config.GroupChatID)
	bot.AddComponent(tg)
	bot.AddComponent(irc.New(bot, "#avplayer", "avbot-tg"))
	bot.AddComponent(ws.New(bot, token, config.WebSocket.Port))
	bot.AddComponent(question.New(tg))
	// bot.AddComponent(joke.New())
	// bot.AddComponent(hello.New(config.Welcome))
	bot.AddComponent(stat.New("stat.dat"))
	bot.AddComponent(chatlog.GetInstance())
	chatlog.GetInstance().Init(config.ChatLog.Path)

	bot.AddComponent(github.New(bot, config.Github.Listen))
	bot.Run()
}
