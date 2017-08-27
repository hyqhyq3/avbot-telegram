package avbot

type Component interface {
	GetName() string
}

type HasProcess interface {
	Process(bot *AVBot, msg *MessageInfo) (processed bool)
}

type HasInit interface {
	Init()
}

type HasSetSendMessageChannel interface {
	SetSendMessageChannel(msgChan chan<- *MessageInfo)
}

type Stoppable interface {
	Stop()
}

type HasUid interface {
	GetUid() int
}

type HasFace interface {
	GetFace(int) ([]byte, error)
}
