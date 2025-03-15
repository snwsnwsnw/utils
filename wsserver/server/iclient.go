package server

type IClient[USERDATA any, CONN any, MT MsgType] interface {
	GetId() ClientID
	GetUserData() USERDATA
	GetConn() CONN
	IsClosed() bool
	Close() error
	WriteMessage(MT, []byte) error
}
