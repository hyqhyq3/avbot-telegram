package main

import (
	"github.com/hyqhyq3/avbot-telegram"
	"github.com/hyqhyq3/avbot-telegram/hello"
	"github.com/hyqhyq3/avbot-telegram/joke"
)

func main() {
	//	bot := avbot.NewBot("154517069:AAElhGUMLDA4mV9isLQDgfJoBOpdSSu3Ch0")
	bot := avbot.NewBot("148772277:AAEnpizxwjkHA3M6j2u0edTUPssuIXLXhHM")
	//	bot.SetProxy("socks5://127.0.0.1:1080")
	bot.AddMessageHook(joke.New())
	bot.AddMessageHook(hello.New("@{{Name}} 你好，欢迎来到avplayer"))
	bot.Run()
}
