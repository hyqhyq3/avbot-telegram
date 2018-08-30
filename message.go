package avbot

import (
	"time"

	"github.com/hyqhyq3/avbot-telegram/data"
)

type MessageInfo struct {
	*data.Message
	Channel   Component
	ExtraData interface{}
	MessageID int
}

func NewTextMessage(c Component, text string) *MessageInfo {
	ts := time.Now().Unix()
	return &MessageInfo{
		Message: &data.Message{
			Type:      data.MessageType_TEXT,
			Content:   text,
			Timestamp: ts,
		},
		Channel: c,
	}
}

func NewImageMessage(c Component, fileID string) *MessageInfo {
	ts := time.Now().Unix()

	return &MessageInfo{
		Message: &data.Message{
			Type:      data.MessageType_IMAGE,
			FileID:    fileID,
			Timestamp: ts,
		},
		Channel: c,
	}
}

func NewVideoMessage(c Component, fileID string) *MessageInfo {
	ts := time.Now().Unix()

	return &MessageInfo{
		Message: &data.Message{
			Type:      data.MessageType_VIDEO,
			FileID:    fileID,
			Timestamp: ts,
		},
		Channel: c,
	}
}

func NewChatMemberMessage(c Component) *MessageInfo {
	ts := time.Now().Unix()

	return &MessageInfo{
		Message: &data.Message{
			Type:      data.MessageType_NEW_MEMBER,
			Timestamp: ts,
		},
		Channel: c,
	}
}
