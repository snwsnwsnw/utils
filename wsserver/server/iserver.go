package server

import "context"

type IServer[MT MsgType] interface {
	Context() context.Context
	NextConnectionID() ClientID
	GetConnectionNum() uint64
	IsClosed() bool
	OnConnect() chan WSClient
	OnMessage() chan IMessageEvent[MT]
	OnDisconnect() chan IDisconnectEvent
}
