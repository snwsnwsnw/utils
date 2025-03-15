package server

type IMessageEvent[MT MsgType] interface {
	GetClient() WSClient
	MessageType() MT
	Data() []byte
}

type IDisconnectEvent interface {
	GetClient() WSClient
	Error() error
}
