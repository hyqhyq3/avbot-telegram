package ws

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/hyqhyq3/avbot-telegram"
	"golang.org/x/net/websocket"
	"gopkg.in/telegram-bot-api.v2"
)

type MessageData struct {
	Timestamp string `json:"timestamp"`
	Msg       string `json:"msg"`
	From      string `json:"from"`
}

type Message struct {
	Cmd  int         `json:"cmd"`
	Data MessageData `json:"data"`
}

type WSChatServer struct {
	websocket.Server
	clients     map[string]*websocket.Conn
	clientMutex sync.Mutex
	bot         *tgbotapi.BotAPI
	chatId      int
}

func New(bot *tgbotapi.BotAPI, port int) avbot.MessgaeHook {
	handler := &WSChatServer{bot: bot}
	handler.clients = make(map[string]*websocket.Conn)
	handler.Handler = handler.OnNewClient
	go func() { http.ListenAndServe(":"+strconv.Itoa(port), handler) }()
	return handler
}

func (ws *WSChatServer) Process(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) bool {

	if msg.Chat.Type == "group" || msg.Chat.Type == "supergroup" {
		ws.chatId = msg.Chat.ID
	}
	ts := strconv.Itoa(getNow())
	wsMsg := &Message{
		Cmd: 1,
		Data: MessageData{
			Timestamp: ts,
			Msg:       msg.Text,
			From:      msg.From.FirstName,
		},
	}
	ws.Broadcast(wsMsg)
	return false
}

func (ws *WSChatServer) Broadcast(msg *Message) {
	ws.clientMutex.Lock()
	for _, v := range ws.clients {
		websocket.JSON.Send(v, msg)
	}
	ws.clientMutex.Unlock()
}

func (ws *WSChatServer) OnNewClient(c *websocket.Conn) {
	ws.clientMutex.Lock()
	ws.clients[c.RemoteAddr().String()] = c
	ws.clientMutex.Unlock()

	msg := &Message{}
	for {
		if err := websocket.JSON.Receive(c, msg); err == nil {
			tgmsg := tgbotapi.NewMessage(ws.chatId, msg.Data.Msg)
			ws.bot.Send(tgmsg)
		}
	}
}

func getNow() int {
	return int(time.Now().Unix())
}
