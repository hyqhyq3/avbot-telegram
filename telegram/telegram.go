package telegram

import (
	"errors"
	"io/ioutil"
	"log"
	"math"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hyqhyq3/avbot-telegram/ws"

	"github.com/hyqhyq3/avbot-telegram"
	"github.com/hyqhyq3/avbot-telegram/data"

	"golang.org/x/net/proxy"
	"gopkg.in/telegram-bot-api.v4"
)

func init() {
	mime.AddExtensionType(".webp", "image/webp")
}

type Telegram struct {
	*tgbotapi.BotAPI
	closeCh chan int
	sendCh  chan<- *avbot.MessageInfo
	ChatID  int64
}

func New(token, socks5Addr string, ChatID int64) *Telegram {
	dial := net.Dial
	if socks5Addr != "" {
		dialer, err := proxy.SOCKS5("tcp", socks5Addr, nil, proxy.Direct)
		if err != nil {
			panic(err)
		}
		dial = dialer.Dial
	}
	client := &http.Client{
		Transport: &http.Transport{
			Dial: dial,
		},
	}

	bot, err := tgbotapi.NewBotAPIWithClient(token, client)
	if err != nil {
		panic(err)
	}

	h := &Telegram{BotAPI: bot, closeCh: make(chan int), ChatID: ChatID}

	avbot.RegisterFileProvider("tg", h)
	avbot.RegisterFaceProvider("tg", h)
	return h
}

func (h *Telegram) GetFile(fileid string) (p []byte, filetype string, err error) {

	file, err := h.BotAPI.GetFile(tgbotapi.FileConfig{FileID: fileid})
	if err != nil {
		return
	}

	log.Println("file path: ", file.FilePath)
	ext := filepath.Ext(file.FilePath)
	if ext == "" && strings.HasPrefix(file.FilePath, "stickers") {
		filetype = "image/webp"
	} else {
		filetype = mime.TypeByExtension(ext)
	}

	rsp, err := h.Client.Get(file.Link(h.Token))
	if err != nil {
		return
	}
	p, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		return
	}

	return
}

func (h *Telegram) GetFace(uid string) (p []byte, filetype string, err error) {
	userid, _ := strconv.ParseInt(uid, 10, 64)
	photo, err := h.BotAPI.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: int(userid)})
	if err != nil {
		return
	}
	if photo.TotalCount == 0 {
		return nil, "", errors.New("no face")
	}

	fileid := h.GetPhotoFileID(&photo.Photos[0])

	return h.GetFile(fileid)
}

func (h *Telegram) GetName() string {
	return "Telegram"
}

func (h *Telegram) Init() {
	log.Println("telegram init")
	go h.LoopTelegram()
}

func (h *Telegram) SetSendMessageChannel(ch chan<- *avbot.MessageInfo) {
	h.sendCh = ch
}

func (h *Telegram) LoopTelegram() {
	u := tgbotapi.NewUpdate(0)
	updates, err := h.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}
mainLoop:
	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				h.Forward(update.Message)
			}
		case <-h.closeCh:
			break mainLoop
		}
	}
	log.Println("stop telegram")
}

func (h *Telegram) Process(bot *avbot.AVBot, msg *avbot.MessageInfo) (processed bool) {
	log.Println("send message to telegram")

	prefix := ""
	if msg.From != "" {
		prefix = msg.From + ": "
	}
	switch msg.Type {
	case data.MessageType_TEXT:
		m := tgbotapi.NewMessage(h.ChatID, prefix+msg.Content)
		m.ParseMode = tgbotapi.ModeMarkdown
		h.Send(m)
	case data.MessageType_IMAGE:
		log.Println("send image")
		var tgmsg tgbotapi.Chattable
		if msg.FileID == "" {
			if b, ok := msg.ExtraData.(*ws.WSImageData); ok {
				log.Println("upload image")
				name := getRandomImageName(b.Type)
				photo := tgbotapi.NewPhotoUpload(h.ChatID, tgbotapi.FileBytes{Bytes: b.Data, Name: name})
				photo.Caption = prefix + msg.Content
				tgmsg = photo
				tgmsg2, err := h.Send(tgmsg)
				if err != nil {
					msg.FileID = h.GetPhotoFileID(tgmsg2.Photo)
				}
			}
		} else {
			tgmsg = tgbotapi.NewPhotoShare(h.ChatID, msg.FileID)
			h.Send(tgmsg)
		}
	}
	return false
}

func (h *Telegram) Forward(msg *tgbotapi.Message) {
	var botMsg *avbot.MessageInfo
	switch {
	case msg.Text != "":
		botMsg = avbot.NewTextMessage(h, msg.Text)
	case msg.Sticker != nil:
		botMsg = avbot.NewImageMessage(h, h.GetStickerFileID(msg.Sticker))
	case msg.Photo != nil:
		botMsg = avbot.NewImageMessage(h, h.GetPhotoFileID(msg.Photo))
	case msg.Document != nil:
		botMsg = avbot.NewVideoMessage(h, h.GetDocumentFileID(msg.Document))
	case msg.NewChatMembers != nil:
		botMsg = avbot.NewChatMemberMessage(h)
	}
	if botMsg != nil {
		botMsg.MessageID = msg.MessageID
		if msg.NewChatMembers != nil {
			botMsg.From = (*msg.NewChatMembers)[0].UserName
			botMsg.UID = int64((*msg.NewChatMembers)[0].ID)
		} else {
			botMsg.From = msg.From.UserName
			botMsg.UID = int64(msg.From.ID)
		}
		botMsg.Message.Channel = "tg"
		h.sendCh <- botMsg
	}
}

func (h *Telegram) GetStickerFileID(sticker *tgbotapi.Sticker) string {
	if sticker != nil {
		return sticker.FileID
	}
	return ""
}

func (h *Telegram) GetDocumentFileID(doc *tgbotapi.Document) string {
	if doc != nil {
		return doc.FileID
	}
	return ""
}

func (h *Telegram) GetPhotoFileID(photo *[]tgbotapi.PhotoSize) string {
	if photo != nil {
		minSize := math.MaxInt32
		fileID := ""
		for _, p := range *photo {
			if p.Width <= 320 && p.Height <= 320 {
				if p.FileSize > minSize {
					minSize = p.FileSize
					fileID = p.FileID
				}
			} else {
				if p.FileSize < minSize {
					minSize = p.FileSize
					fileID = p.FileID
				}
			}
		}
		return fileID
	}
	return ""
}

func (h *Telegram) Stop() {
	h.closeCh <- 1
}

func getRandomImageName(typ string) string {
	name := strconv.Itoa(avbot.GetNow())
	ext, _ := mime.ExtensionsByType(typ)
	if ext != nil || len(ext) > 0 {
		return name + ext[0]
	}
	return name + ".png"
}
