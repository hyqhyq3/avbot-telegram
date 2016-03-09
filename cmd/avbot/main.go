package main

import (
	"github.com/hyqhyq3/avbot-telegram"
	"github.com/hyqhyq3/avbot-telegram/joke"
)

func main() {
	bot := avbot.NewBot("154517069:AAElhGUMLDA4mV9isLQDgfJoBOpdSSu3Ch0")
	bot.AddMessageHook(joke.New())
	bot.Run()
}
