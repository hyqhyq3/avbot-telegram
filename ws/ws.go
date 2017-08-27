package ws

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/hyqhyq3/avbot-telegram/chatlog"
	"github.com/hyqhyq3/avbot-telegram/data"

	_ "golang.org/x/image/webp"

	"github.com/gorilla/mux"
	"github.com/hyqhyq3/avbot-telegram"
	"golang.org/x/net/websocket"
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
	FilePath  string       `json:"file_path"`
}

type Message struct {
	Cmd  data.MessageType `json:"cmd"`
	Data MessageData      `json:"data"`
}

type WSChatServer struct {
	http.ServeMux
	clients     map[int]*websocket.Conn
	clientMutex sync.Mutex
	bot         *avbot.AVBot
	index       int
	Token       string
	sendCh      chan<- *avbot.MessageInfo
}

func New(bot *avbot.AVBot, token string, port int) avbot.Component {
	wsServer := &websocket.Server{}
	handler := &WSChatServer{bot: bot}

	r := mux.NewRouter()
	r.Handle("/", wsServer)
	r.HandleFunc("/avbot/face/{uid}", handler.GetFace)
	r.HandleFunc("/avbot/file/{provider}/{fileid}", handler.GetFile)
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

func (ws *WSChatServer) GetName() string {
	return "WebSocketClient"
}

func (ws *WSChatServer) SetSendMessageChannel(sendCh chan<- *avbot.MessageInfo) {
	ws.sendCh = sendCh
}

/*
func (ws *WSChatServer) GetFace(w http.ResponseWriter, r *http.Request) {
	c, rw, _ := w.(http.Hijacker).Hijack()

	go func() {
		defer c.Close()

		uid, err := strconv.Atoi(mux.Vars(r)["uid"])
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
*/

func chatLogToWsMsg(msg *data.Message) *Message {
	wsMsg := &Message{}
	wsMsg.Cmd = msg.Type
	wsMsg.Data.Msg = msg.Content
	wsMsg.Data.From = msg.From
	wsMsg.Data.Timestamp = strconv.FormatInt(msg.Timestamp, 10)
	if msg.UID != 0 {
		wsMsg.Data.User = &MessageUser{Name: msg.From, ID: int(msg.UID)}
	}
	return wsMsg
}

func chatLogToWsMsgArr(msgs []*data.Message) []*Message {
	arr := make([]*Message, len(msgs))

	for k, v := range msgs {
		arr[k] = chatLogToWsMsg(v)
	}
	return arr
}

func (ws *WSChatServer) GetHistory(w http.ResponseWriter, r *http.Request) {
	from, _ := strconv.ParseUint(mux.Vars(r)["from"], 10, 64)
	to, _ := strconv.ParseUint(mux.Vars(r)["to"], 10, 64)

	msgs := chatLogToWsMsgArr(chatlog.GetInstance().Get(from, to))

	data, _ := json.Marshal(msgs)
	w.Write(data)
}

func (ws *WSChatServer) GetFile(w http.ResponseWriter, r *http.Request) {
	fileid := mux.Vars(r)["fileid"]
	provider := mux.Vars(r)["provider"]
	b, t, err := avbot.GetFile(provider, fileid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", t)
		w.Write(b)
	}
}

func (ws *WSChatServer) GetFace(w http.ResponseWriter, r *http.Request) {
	uid := mux.Vars(r)["uid"]
	b, t, err := avbot.GetFace("tg", uid)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		log.Println("no face " + uid)
	} else {
		w.Header().Set("Content-Type", t)
		w.Write(b)
	}
}

func (ws *WSChatServer) Process(bot *avbot.AVBot, msg *avbot.MessageInfo) bool {

	go ws.AsyncGetWsMsg(msg, func(wsMsg *Message) {
		if wsMsg != nil {
			ws.Broadcast(wsMsg)
		}
	})
	return false
}

func (ws *WSChatServer) AsyncGetWsMsg(msg *avbot.MessageInfo, cb func(wsMsg *Message)) {
	var wsMsg *Message
	var usr *MessageUser

	usr = &MessageUser{ID: int(msg.UID), Name: msg.From}

	ts := strconv.Itoa(avbot.GetNow())
	switch msg.Type {
	case data.MessageType_TEXT:
		wsMsg = &Message{
			Cmd: msg.Type,
			Data: MessageData{
				Timestamp: ts,
				Msg:       msg.Content,
				From:      msg.From,
				User:      usr,
			},
		}
	case data.MessageType_IMAGE, data.MessageType_VIDEO:

		wsMsg = &Message{
			Cmd: msg.Type,
			Data: MessageData{
				Timestamp: ts,
				Msg:       msg.Content,
				From:      msg.From,
				User:      usr,
				FilePath:  "avbot/file/" + msg.Message.Channel + "/" + msg.FileID,
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
			go ws.AsyncGetAvbotMsg(msg, func(botMsg *avbot.MessageInfo) {
				ws.sendCh <- botMsg
				ws.Broadcast(msg)
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

type WSImageData struct {
	Data []byte
	Type string
}

func (ws *WSChatServer) AsyncGetAvbotMsg(msg *Message, cb func(*avbot.MessageInfo)) {
	botMsg := &avbot.MessageInfo{}

	ts, err := strconv.ParseInt(msg.Data.Timestamp, 10, 64)
	if err != nil {
		ts = time.Now().Unix()
	}
	var dataMsg *data.Message

	switch msg.Cmd {
	case data.MessageType_TEXT:
		dataMsg = &data.Message{
			Type:      msg.Cmd,
			Timestamp: ts,
			Content:   msg.Data.Msg,
			From:      msg.Data.From,
		}
	case data.MessageType_IMAGE:

		dataMsg = &data.Message{
			Type:      msg.Cmd,
			Timestamp: ts,
			Content:   msg.Data.Msg,
			From:      msg.Data.From,
		}

	}

	if botMsg != nil {
		botMsg.Message = dataMsg
		botMsg.Channel = ws

		if msg.Cmd == data.MessageType_IMAGE {
			data, err := base64.StdEncoding.DecodeString(msg.Data.ImgData)
			if err == nil {
				botMsg.ExtraData = &WSImageData{
					Type: msg.Data.ImgType,
					Data: data,
				}
			}
		}

		cb(botMsg)
	}
}
