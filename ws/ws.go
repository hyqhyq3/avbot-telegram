package ws

import (
	"encoding/base64"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/hyqhyq3/avbot-telegram"
	"golang.org/x/net/websocket"
	"gopkg.in/telegram-bot-api.v4"
)

type MessageUser struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type MessageData struct {
	Timestamp string       `json:"timestamp"`
	Msg       string       `json:"msg"`
	From      string       `json:"from"`
	ImgType   string       `json:"img_type"`
	ImgData   string       `json:"img_data"`
	Caption   string       `json:"caption"`
	User      *MessageUser `json:"user"`
}

type Message struct {
	Cmd  int         `json:"cmd"`
	Data MessageData `json:"data"`
}

type WSChatServer struct {
	http.ServeMux
	clients     map[int]*websocket.Conn
	clientMutex sync.Mutex
	bot         *avbot.AVBot
	index       int
	Token       string
}

func New(bot *avbot.AVBot, token string, port int) avbot.MessgaeHook {
	wsServer := &websocket.Server{}
	handler := &WSChatServer{bot: bot}
	handler.Handle("/", wsServer)
	handler.HandleFunc("/avbot/face/", handler.GetFace)
	handler.index = 1
	handler.clients = make(map[int]*websocket.Conn)
	wsServer.Handler = handler.OnNewClient
	handler.Token = token
	go func() {
		log.Printf("listening websocket on %s\n", port)
		http.ListenAndServe(":"+strconv.Itoa(port), handler)
	}()
	return handler
}

func (ws *WSChatServer) GetFace(w http.ResponseWriter, r *http.Request) {
	c, rw, _ := w.(http.Hijacker).Hijack()

	go func() {
		defer c.Close()

		str := strings.TrimPrefix(r.RequestURI, "/avbot/face/")
		uid, err := strconv.Atoi(str)
		if err != nil {
			log.Println(err)
			return
		}
		photos, err := ws.bot.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: uid, Offset: 0, Limit: 1})
		if err != nil {
			log.Println(err)
			return
		}
		if photos.TotalCount == 0 {
			log.Println("no photo")
			return
		}
		photoSize := photos.Photos[0][0]
		file, err := ws.bot.GetFile(tgbotapi.FileConfig{FileID: photoSize.FileID})
		if err != nil {
			log.Println("unable to get file")
			return
		}
		url, err := ws.bot.GetFileDirectURL(photoSize.FileID)
		if err != nil {
			log.Println("cannot get download link")
			return
		}
		resp, err := ws.bot.Client.Get(url)
		if err != nil {
			log.Println("cannot download photo")
			return
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("failed to download photo")
			return
		}

		rw.WriteString("HTTP/1.1 200 OK\r\n")
		rw.WriteString("Content-Type: " + mime.TypeByExtension(filepath.Ext(file.FilePath)) + "\r\n")
		rw.WriteString("Content-Length: " + strconv.Itoa(len(data)) + "\r\n")
		rw.WriteString("\r\n")
		rw.Write(data)
		rw.Flush()
	}()
}

func (ws *WSChatServer) Process(bot *avbot.AVBot, msg *tgbotapi.Message) bool {

	go ws.AsyncGetWsMsg(msg, func(wsMsg *Message) {
		if wsMsg != nil {
			ws.Broadcast(wsMsg)
		}
	})
	return false
}

func (ws *WSChatServer) AsyncGetWsMsg(msg *tgbotapi.Message, cb func(wsMsg *Message)) {
	var wsMsg *Message
	var usr *MessageUser

	if msg.From != nil {
		usr = &MessageUser{ID: msg.From.ID, Name: msg.From.FirstName}
	}

	ts := strconv.Itoa(getNow())
	switch {
	case msg.Text != "":
		wsMsg = &Message{
			Cmd: 1,
			Data: MessageData{
				Timestamp: ts,
				Msg:       msg.Text,
				From:      msg.From.FirstName,
				User:      usr,
			},
		}
	case msg.Photo != nil && len(*msg.Photo) > 0:
		file, err := ws.bot.GetFile(tgbotapi.FileConfig{FileID: (*msg.Photo)[0].FileID})
		if err != nil {
			log.Println(err)
			return
		}
		link := file.Link(ws.Token)

		resp, err := ws.bot.Client.Get(link)
		if err != nil {
			log.Println(err)
			return
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}
		imgData := base64.StdEncoding.EncodeToString(data)
		imgType := mime.TypeByExtension(filepath.Ext(file.FilePath))
		wsMsg = &Message{
			Cmd: 2,
			Data: MessageData{
				ImgType: imgType,
				ImgData: imgData,
				From:    msg.From.FirstName,
				Caption: msg.Caption,
				User:    usr,
			},
		}
	}

	if wsMsg != nil {
		cb(wsMsg)
	}

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
	index := ws.index
	ws.index++
	ws.clients[index] = c
	log.Println("new client")
	ws.clientMutex.Unlock()

	msg := &Message{}
	for {
		err := websocket.JSON.Receive(c, msg)
		if err == nil {
			log.Printf("received message type: %d from: %s text: %s", msg.Cmd, msg.Data.From, msg.Data.Msg)
			go ws.AsyncGetTgMsg(msg, func(tgmsg tgbotapi.Chattable) {
				ws.bot.Send(tgmsg)
				ws.Broadcast(msg)
			})

		} else {
			break
		}
	}
	c.Close()

	ws.clientMutex.Lock()
	log.Println("client disconnected")
	delete(ws.clients, index)
	ws.clientMutex.Unlock()
}

func (ws *WSChatServer) AsyncGetTgMsg(msg *Message, cb func(tgbotapi.Chattable)) {
	var tgmsg tgbotapi.Chattable
	chatId := ws.bot.GetGroupChatId()

	switch msg.Cmd {
	case 1:
		tgmsg = tgbotapi.NewMessage(chatId, msg.Data.From+": "+msg.Data.Msg)
	case 2:
		data, err := base64.StdEncoding.DecodeString(msg.Data.ImgData)
		if err != nil {
			log.Println("image data error")
		}
		photo := tgbotapi.NewPhotoUpload(chatId, tgbotapi.FileBytes{
			Name:  getRandomImageName(msg.Data.ImgType),
			Bytes: data,
		})
		photo.Caption = msg.Data.From + ":" + msg.Data.Caption
		tgmsg = photo
	}
	if tgmsg != nil {
		cb(tgmsg)
	}
}

func getNow() int {
	return int(time.Now().Unix())
}

func getRandomImageName(typ string) string {
	name := strconv.Itoa(getNow())
	ext, _ := mime.ExtensionsByType(typ)
	if ext != nil || len(ext) > 0 {
		return name + ext[0]
	}
	return name + ".png"
}
