package avbot

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
)

type AVBot struct {
	components map[string]Component
	processors []HasProcess
	client     *http.Client
	closeCh    chan int

	sendMessageChan chan *MessageInfo
}

func (b *AVBot) AddComponent(hook Component) {
	if b.components == nil {
		b.components = make(map[string]Component)
	}

	if _, found := b.components[hook.GetName()]; found {
		log.Fatal(errors.New("Component " + hook.GetName() + " exists"))
	}

	if p, ok := hook.(HasSetSendMessageChannel); ok {
		p.SetSendMessageChannel(b.sendMessageChan)
	}
	if p, ok := hook.(HasInit); ok {
		p.Init()
	}
	if p, ok := hook.(HasProcess); ok {
		log.Println("register processor", hook.GetName())
		b.processors = append(b.processors, p)
	}
	b.components[hook.GetName()] = hook
}

func NewBot() *AVBot {
	return &AVBot{
		closeCh:         make(chan int),
		sendMessageChan: make(chan *MessageInfo),
	}
}

func (b *AVBot) Run() {
	log.Println("bot running")
	go b.HandleSignal()

mainLoop:
	for {
		select {
		case msg := <-b.sendMessageChan:
			log.Println("receved message ", msg)
			b.SendMessage(msg)
		case <-b.closeCh:
			break mainLoop
		}
	}

	b.Stop()
}

func (b *AVBot) HandleSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	b.closeCh <- 1
	log.Println("received interrupt signal")
}

func (b *AVBot) SendMessage(msg *MessageInfo) {
	for _, h := range b.processors {
		if h.(Component) != msg.Channel {
			h.Process(b, msg)
		}
	}
}

func (b *AVBot) Stop() {
	log.Println("stop all components")
	for _, v := range b.components {
		if o, ok := v.(Stoppable); ok {
			o.Stop()
		}
	}
}
