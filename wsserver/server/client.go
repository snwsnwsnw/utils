package server

import (
	"fmt"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"log/slog"
	"net/http"
	"time"
	"wsserver/conf"
)

type Client[MT MsgType] struct {
	ClientInfo
	Id       ClientID
	server   IServer[MT]
	conn     *websocket.Conn
	wsEngine *websocket.Upgrader
}

func NewClient[MT MsgType](s IServer[MT], clientInfo *ClientInfo) *Client[MT] {
	newClient := Client[MT]{
		ClientInfo: *clientInfo,
		Id:         s.NextConnectionID(),
		server:     s,
		wsEngine:   websocket.NewUpgrader(),
	}
	newClient.initHandler()
	return &newClient
}

func (client *Client[MT]) initHandler() {
	client.wsEngine.OnOpen(func(c *websocket.Conn) {
		//slog.DebugContext(client.server.Context(), "[WSServer] new connection", ClientInfoLog(client)...)
		//_, serverPort, _ := net.SplitHostPort(c.LocalAddr().String())
		//slog.DebugContext(client.server.Context(), "[WSServer] new connection", "port", serverPort)
		if conf.WSConfig.MaxConnection != 0 && client.server.GetConnectionNum() > conf.WSConfig.MaxConnection {
			_, _ = c.Write([]byte(fmt.Sprintf("connection on limit %d/%d", client.server.GetConnectionNum(), conf.WSConfig.MaxConnection)))
			slog.WarnContext(client.server.Context(), "[WSServer] connection on limit", "num", client.server.GetConnectionNum(), "max", conf.WSConfig.MaxConnection)
			_ = c.Close()
			return
		}
		if !client.server.IsClosed() {
			client.server.OnConnect() <- client
		}
	})

	client.wsEngine.OnMessage(func(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
		err := c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(conf.WSConfig.PingTimeOut)))
		if err != nil {
			slog.ErrorContext(client.server.Context(), "[WSServer] SetReadDeadline error", ClientInfoLog(client, "err", err)...)
			_ = c.Close()
			return
		}

		//slog.DebugContext(client.server.Context(), "[WSServer] receive message", ClientInfoLog(client, "messageType", messageType, "data", string(data))...)
		if !client.server.IsClosed() {
			client.server.OnMessage() <- &MessageEvent[MT]{client: client, messageType: MT(messageType), data: data}
		}
	})

	client.wsEngine.OnClose(func(c *websocket.Conn, err error) {
		//slog.DebugContext(client.server.Context(), "[WSServer] connection closed", ClientInfoLog(client, "err", err)...)
		if !client.server.IsClosed() {
			client.server.OnDisconnect() <- &DisconnectEvent{client: client, err: err}
		}
		client.conn = nil
	})
}

func (client *Client[MT]) GetId() ClientID {
	return client.Id
}

func (client *Client[MT]) GetConn() *websocket.Conn {
	return client.conn
}

func (client *Client[MT]) IsClosed() bool {
	return client.conn == nil
}

func (client *Client[MT]) Close() error {
	if client.conn != nil {
		return client.conn.Close()
	}
	return nil
}

func (client *Client[MT]) WriteMessage(messageType websocket.MessageType, data []byte) error {
	if client.conn == nil {
		return fmt.Errorf("connection is closed")
	}
	return client.conn.WriteMessage(messageType, data)
}

func (client *Client[MT]) GetUserData() ClientInfo {
	return client.ClientInfo
}

func (client *Client[MT]) Upgrade(w http.ResponseWriter, r *http.Request) error {
	conn, err := client.wsEngine.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	client.conn = conn
	return nil
}
