package ws

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hyqhyq3/avbot-telegram/chatlog"

	_ "golang.org/x/image/webp"

	"github.com/gorilla/mux"
	"github.com/hyqhyq3/avbot-telegram"
	"golang.org/x/net/websocket"
	"gopkg.in/telegram-bot-api.v4"
)

type MessageType int

const (
	MessageType_Text MessageType = iota + 1
	MessageType_Image
	MessageType_Video
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
	VideoType string       `json:"video_type"`
	VideoData string       `json:"video_data"`
	Caption   string       `json:"caption"`
	User      *MessageUser `json:"user"`
}

type Message struct {
	Cmd  MessageType `json:"cmd"`
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

func New(bot *avbot.AVBot, token string, port int) avbot.MessageHook {
	wsServer := &websocket.Server{}
	handler := &WSChatServer{bot: bot}

	r := mux.NewRouter()
	r.Handle("/", wsServer)
	r.HandleFunc("/avbot/face/", handler.GetFace)
	r.HandleFunc("/avbot/history/{from}-{to}", handler.GetHistory)
	handler.Handle("/", r)

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

		if r.Header.Get("If-None-Match") == file.FileID {
			rw.WriteString("HTTP/1.1 304 Not Modified\r\n")
			rw.WriteString("\r\n")
			rw.Flush()
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
		rw.WriteString("Cache-Control: max-age=2592000\r\n")
		rw.WriteString("ETag: " + file.FileID + "\r\n")
		rw.WriteString("Content-Type: " + mime.TypeByExtension(filepath.Ext(file.FilePath)) + "\r\n")
		rw.WriteString("Content-Length: " + strconv.Itoa(len(data)) + "\r\n")
		rw.WriteString("\r\n")
		rw.Write(data)
		rw.Flush()
	}()
}

func chatLogToWsMsg(msg *chatlog.ChatLog) *Message {
	wsMsg := &Message{}
	wsMsg.Cmd = MessageType(msg.Type)
	wsMsg.Data.Msg = msg.Content
	wsMsg.Data.From = msg.From
	wsMsg.Data.Timestamp = strconv.FormatInt(msg.Timestamp, 10)
	if msg.UID != 0 {
		wsMsg.Data.User = &MessageUser{Name: msg.From, ID: int(msg.UID)}
	}
	return wsMsg
}

func chatLogToWsMsgArr(msgs []*chatlog.ChatLog) []*Message {
	arr := make([]*Message, len(msgs))

	for k, v := range msgs {
		arr[k] = chatLogToWsMsg(v)
	}
	return arr
}

func (ws *WSChatServer) GetHistory(w http.ResponseWriter, r *http.Request) {
	from, _ := strconv.ParseUint(mux.Vars(r)["from"], 10, 64)
	to, _ := strconv.ParseUint(mux.Vars(r)["to"], 10, 64)

	msgs := make([]*Message, 0, 100)
	for _, msg := range chatlog.GetInstance().Get(from, to) {
		wsMsg := &Message{}
		wsMsg.Cmd = MessageType(msg.Type)
		wsMsg.Data.Msg = msg.Content
		wsMsg.Data.From = msg.From
		wsMsg.Data.Timestamp = strconv.FormatInt(msg.Timestamp, 10)
		msgs = append(msgs, wsMsg)
	}

	data, _ := json.Marshal(msgs)
	w.Write(data)
}

func (ws *WSChatServer) Process(bot *avbot.AVBot, msg *tgbotapi.Message) bool {

	go ws.AsyncGetWsMsg(msg, func(wsMsg *Message) {
		if wsMsg != nil {
			ws.Broadcast(wsMsg)
		}
	})
	return false
}

func (ws *WSChatServer) Download(fileID string) (data []byte, typ string, err error) {
	file, err := ws.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return
	}
	link := file.Link(ws.Token)

	resp, err := ws.bot.Client.Get(link)
	if err != nil {
		return
	}
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	typ = mime.TypeByExtension(filepath.Ext(file.FilePath))
	return
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
			Cmd: MessageType_Text,
			Data: MessageData{
				Timestamp: ts,
				Msg:       msg.Text,
				From:      msg.From.FirstName,
				User:      usr,
			},
		}
	case (msg.Photo != nil && len(*msg.Photo) > 0) || msg.Sticker != nil || msg.Document != nil:
		cmdType := MessageType_Image
		var fileID string
		if msg.Photo != nil && len(*msg.Photo) > 0 {
			fileID = (*msg.Photo)[0].FileID
		} else if msg.Sticker != nil {
			if msg.Sticker.Thumbnail != nil {
				fileID = msg.Sticker.Thumbnail.FileID
			} else {
				fileID = msg.Sticker.FileID
			}
		} else if msg.Document != nil {
			fileID = msg.Document.FileID
			cmdType = MessageType_Video
		}
		data, fileType, err := ws.Download(fileID)
		if err != nil {
			log.Println(err)
			return
		}

		img, fileType, err := image.Decode(bytes.NewReader(data))

		if err == nil && fileType == "webp" {
			buf := &bytes.Buffer{}
			png.Encode(buf, img)
			data = buf.Bytes()
			fileType = "image/png"
		}

		fileData := base64.StdEncoding.EncodeToString(data)
		wsMsg = &Message{
			Cmd: cmdType,
			Data: MessageData{
				Timestamp: ts,
				From:      msg.From.FirstName,
				Caption:   msg.Caption,
				User:      usr,
			},
		}
		if cmdType == MessageType_Image {
			wsMsg.Data.ImgData = fileData
			wsMsg.Data.ImgType = fileType
		} else if cmdType == MessageType_Video {
			wsMsg.Data.VideoData = fileData
			wsMsg.Data.VideoType = fileType
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

	chatLog := chatlog.GetInstance().Last(10)

	for _, v := range chatLogToWsMsgArr(chatLog) {
		websocket.JSON.Send(c, v)
	}

	msg := &Message{}
	for {
		c.SetReadDeadline(time.Now().Add(time.Second * 60))
		err := websocket.JSON.Receive(c, msg)
		if err == nil {
			log.Printf("received message type: %d from: %s text: %s", msg.Cmd, msg.Data.From, msg.Data.Msg)
			go ws.AsyncGetTgMsg(msg, func(tgmsg tgbotapi.Chattable) {
				ws.bot.Send(tgmsg)
				ws.Broadcast(msg)

				chatLog := &chatlog.ChatLog{}
				chatLog.Content = msg.Data.Msg
				chatLog.Type = chatlog.MessageType(msg.Cmd)
				chatLog.From = msg.Data.From
				if msg.Data.User != nil {
					chatLog.UID = int64(msg.Data.User.ID)
				}
				chatlog.GetInstance().AddLog(chatLog)
			})

		} else {
			if e, ok := err.(net.Error); ok {
				if e.Temporary() {
					continue
				}
			}
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
	case MessageType_Text:
		tgmsg = tgbotapi.NewMessage(chatId, msg.Data.From+": "+msg.Data.Msg)
	case MessageType_Image:
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
