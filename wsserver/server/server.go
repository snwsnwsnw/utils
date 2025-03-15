package server

import (
	"context"
	"fmt"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"github.com/mssola/useragent"
	"github.com/snwsnwsnw/utils"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"wsserver/common"
	"wsserver/metric"
)

var Server *WSServer

type WSClient = IClient[ClientInfo, *websocket.Conn, websocket.MessageType]
type ClientRef = IMap[WSClient, struct{}]

type WSServer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	idAllocator   IIdGenerator
	ClientBus     ClientRef
	connectionNum uint64

	onConnect    chan WSClient
	onDisconnect chan IDisconnectEvent
	onMessage    chan IMessageEvent[websocket.MessageType]
	WorkerNum    int
	workerWg     *sync.WaitGroup

	PlayersById       IMap[PlayerId, ClientRef]
	PlayersByPackage  IMap[PackageId, ClientRef] // 玩家連線 依據PackageId做分組
	PlayersByAgent    IMap[AgentId, ClientRef]   // 玩家連線 依據AgentId做分組
	PlayersByChannel  IMap[ChannelId, ClientRef] // 玩家連線 依據ChannelId做分組
	VisitorsByPackage IMap[PackageId, ClientRef] // 訪客連線 依據PackageId做分組
	VisitorsByAgent   IMap[AgentId, ClientRef]   // 訪客連線 依據AgentId做分組
	VisitorsByChannel IMap[ChannelId, ClientRef] // 訪客連線 依據ChannelId做分組

	httpEngine *nbhttp.Engine
	done       bool
}

func Start(ctx context.Context) {
	if Server != nil {
		return
	}

	innerCtx, cancel := context.WithCancel(ctx)

	Server = &WSServer{
		ctx:               innerCtx,
		cancel:            cancel,
		idAllocator:       &idAllotment{},
		ClientBus:         utils.NewSafeMap[WSClient, struct{}](),
		onConnect:         make(chan WSClient, 1024),
		onDisconnect:      make(chan IDisconnectEvent, 1024),
		onMessage:         make(chan IMessageEvent[websocket.MessageType], 1024),
		PlayersById:       utils.NewSafeMap[PlayerId, ClientRef](),
		PlayersByPackage:  utils.NewSafeMap[PackageId, ClientRef](),
		PlayersByAgent:    utils.NewSafeMap[AgentId, ClientRef](),
		PlayersByChannel:  utils.NewSafeMap[ChannelId, ClientRef](),
		VisitorsByPackage: utils.NewSafeMap[PackageId, ClientRef](),
		VisitorsByAgent:   utils.NewSafeMap[AgentId, ClientRef](),
		VisitorsByChannel: utils.NewSafeMap[ChannelId, ClientRef](),
		WorkerNum:         32,
		workerWg:          &sync.WaitGroup{},
	}

	if Server.Init() != nil {
		return
	}

	go Server.Start()

	slog.InfoContext(ctx, "[WSServer] Started")
}

func (s *WSServer) Init() error {
	return s.initHttpServer()
}

func (s *WSServer) initHttpServer() error {
	if len(wsAddrs) == 0 {
		return nil
	}
	mux := &http.ServeMux{}
	mux.HandleFunc("/wsserver/connect", s.onWebsocket)

	s.httpEngine = nbhttp.NewEngine(nbhttp.Config{
		Network:                 "tcp",
		Addrs:                   wsAddrs,
		MaxLoad:                 1000000,
		ReleaseWebsocketPayload: true,
		Handler:                 mux,
		ReadBufferSize:          1024 * 4,
		IOMod:                   nbhttp.IOModMixed,
		MaxBlockingOnline:       100000,
	})

	logging.SetLogger(nil)

	err := s.httpEngine.Start()
	if err != nil {
		slog.ErrorContext(s.ctx, "[WSServer] initHttpServer error", "error", err.Error())
		return err
	} else {
		for _, addr := range wsAddrs {
			slog.InfoContext(s.ctx, fmt.Sprintf("[WSServer] start listen on %s", addr))
		}
	}
	return nil
}

func (s *WSServer) ParseUserAgent(userAgent string) (string, string, string, string, string) {
	ua := useragent.New(userAgent)
	info := ua.OSInfo()
	browserName, browserVersion := ua.Browser()
	return ua.Platform(), info.Name, info.Version, browserName, browserVersion
}

func (s *WSServer) onWebsocket(w http.ResponseWriter, r *http.Request) {
	platform, os, osVersion, browser, browserVersion := s.ParseUserAgent(r.UserAgent())
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	r.Header.Del("Origin")

	queryParams := r.URL.Query()
	var deviceId, channelId, appUrl, token, fromDomain, devicePlatform, systemVersion = queryParams.Get("DeviceId"),
		queryParams.Get("ChannelId"),
		queryParams.Get("AppUrl"),
		queryParams.Get("Token"),
		queryParams.Get("FromDomain"),
		queryParams.Get("DevicePlatform"),
		queryParams.Get("SystemVersion")

	userData, errCode := GetClientInfo(s.ctx,
		platform, os, osVersion, browser, browserVersion, deviceId, channelId, ip,
		appUrl, token, fromDomain, devicePlatform, systemVersion)

	if errCode != 0 {
		http.Error(w, "", http.StatusInternalServerError)
	} else {
		err := Server.Upgrade(w, r, userData)
		if err != nil {
			slog.ErrorContext(s.ctx, "[WSServer][onWebsocket] error", "error", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *WSServer) Upgrade(w http.ResponseWriter, r *http.Request, clientInfo *ClientInfo) error {
	if s == nil {
		return fmt.Errorf("server not started")
	}

	newClient := NewClient[websocket.MessageType](s, clientInfo)
	if err := newClient.Upgrade(w, r); err != nil {
		return err
	}

	s.ClientBus.Set(newClient, struct{}{})

	return nil
}

func (s *WSServer) Context() context.Context {
	return s.ctx
}

func (s *WSServer) clearClientRef() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var needDeletePlayerIds []PlayerId
			s.PlayersById.Range(func(k PlayerId, v ClientRef) bool {
				var needDeleteClients []WSClient
				v.Range(func(k WSClient, _ struct{}) bool {
					if k.IsClosed() {
						needDeleteClients = append(needDeleteClients, k)
					}
					return true
				})
				if len(needDeleteClients) != 0 {
					v.Delete(needDeleteClients...)
				}
				if v.Len() == 0 {
					needDeletePlayerIds = append(needDeletePlayerIds, k)
				}
				return true
			})
			if len(needDeletePlayerIds) != 0 {
				s.PlayersById.Delete(needDeletePlayerIds...)
			}

			var needDeletePackageIds []PackageId
			s.PlayersByPackage.Range(func(k PackageId, v ClientRef) bool {
				var needDeleteClients []WSClient
				v.Range(func(k WSClient, _ struct{}) bool {
					if k.IsClosed() {
						needDeleteClients = append(needDeleteClients, k)
					}
					return true
				})
				if len(needDeleteClients) != 0 {
					v.Delete(needDeleteClients...)
				}
				if v.Len() == 0 {
					needDeletePackageIds = append(needDeletePackageIds, k)
				}
				return true
			})
			if len(needDeletePackageIds) != 0 {
				s.PlayersByPackage.Delete(needDeletePackageIds...)
			}

			var needDeleteAgentIds []AgentId
			s.PlayersByAgent.Range(func(k AgentId, v ClientRef) bool {
				var needDeleteClients []WSClient
				v.Range(func(k WSClient, _ struct{}) bool {
					if k.IsClosed() {
						needDeleteClients = append(needDeleteClients, k)
					}
					return true
				})
				if len(needDeleteClients) != 0 {
					v.Delete(needDeleteClients...)
				}
				if v.Len() == 0 {
					needDeleteAgentIds = append(needDeleteAgentIds, k)
				}
				return true
			})
			if len(needDeleteAgentIds) != 0 {
				s.PlayersByAgent.Delete(needDeleteAgentIds...)
			}

			var needDeleteChannelIds []ChannelId
			s.PlayersByChannel.Range(func(k ChannelId, v ClientRef) bool {
				var needDeleteClients []WSClient
				v.Range(func(k WSClient, _ struct{}) bool {
					if k.IsClosed() {
						needDeleteClients = append(needDeleteClients, k)
					}
					return true
				})
				if len(needDeleteClients) != 0 {
					v.Delete(needDeleteClients...)
				}
				if v.Len() == 0 {
					needDeleteChannelIds = append(needDeleteChannelIds, k)
				}
				return true
			})
			if len(needDeleteChannelIds) != 0 {
				s.PlayersByChannel.Delete(needDeleteChannelIds...)
			}

			needDeletePackageIds = needDeletePackageIds[:0]
			s.VisitorsByPackage.Range(func(k PackageId, v ClientRef) bool {
				var needDeleteClients []WSClient
				v.Range(func(k WSClient, _ struct{}) bool {
					if k.IsClosed() {
						needDeleteClients = append(needDeleteClients, k)
					}
					return true
				})
				if len(needDeleteClients) != 0 {
					v.Delete(needDeleteClients...)
				}
				if v.Len() == 0 {
					needDeletePackageIds = append(needDeletePackageIds, k)
				}
				return true
			})
			if len(needDeletePackageIds) != 0 {
				s.VisitorsByPackage.Delete(needDeletePackageIds...)
			}

			needDeleteAgentIds = needDeleteAgentIds[:0]
			s.VisitorsByAgent.Range(func(k AgentId, v ClientRef) bool {
				var needDeleteClients []WSClient
				v.Range(func(k WSClient, _ struct{}) bool {
					if k.IsClosed() {
						needDeleteClients = append(needDeleteClients, k)
					}
					return true
				})
				if len(needDeleteClients) != 0 {
					v.Delete(needDeleteClients...)
				}
				if v.Len() == 0 {
					needDeleteAgentIds = append(needDeleteAgentIds, k)
				}
				return true
			})
			if len(needDeleteAgentIds) != 0 {
				s.VisitorsByAgent.Delete(needDeleteAgentIds...)
			}

			needDeleteChannelIds = needDeleteChannelIds[:0]
			s.VisitorsByChannel.Range(func(k ChannelId, v ClientRef) bool {
				var needDeleteClients []WSClient
				v.Range(func(k WSClient, _ struct{}) bool {
					if k.IsClosed() {
						needDeleteClients = append(needDeleteClients, k)
					}
					return true
				})
				if len(needDeleteClients) != 0 {
					v.Delete(needDeleteClients...)
				}
				if v.Len() == 0 {
					needDeleteChannelIds = append(needDeleteChannelIds, k)
				}
				return true
			})
			if len(needDeleteChannelIds) != 0 {
				s.VisitorsByChannel.Delete(needDeleteChannelIds...)
			}

			var needDeleteClientIds []WSClient
			s.ClientBus.Range(func(k WSClient, _ struct{}) bool {
				if k.IsClosed() {
					needDeleteClientIds = append(needDeleteClientIds, k)
				}
				return true
			})
			if len(needDeleteClientIds) != 0 {
				s.ClientBus.Delete(needDeleteClientIds...)
			}

		case <-s.ctx.Done():
			return
		}
	}
}

func (s *WSServer) Start() {
	go s.clearClientRef()
	go s.startHandleEvent()
}

func (s *WSServer) startHandleEvent() {
	s.workerWg.Add(s.WorkerNum)
	for range s.WorkerNum {
		go s.handleEvent()
	}
	s.workerWg.Wait()
}

func (s *WSServer) handleEvent() {
	defer s.workerWg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return

		case client := <-s.onConnect:
			s.processOnConnect(client)

		case event := <-s.onDisconnect:
			s.processOnDisconnect(event)

		case event := <-s.onMessage:
			s.processOnMessage(event)

		}
	}
}

func (s *WSServer) GetConnectionNum() uint64 {
	return s.connectionNum
}

func (s *WSServer) processOnConnect(client WSClient) {
	atomic.AddUint64(&s.connectionNum, 1)
	metric.MircoWebSocketConnectedClient.Inc()

	slog.InfoContext(s.ctx, "[WSServer] onConnect", ClientInfoLog(client)...)

	if !client.GetUserData().Visitor {
		m, exist := s.PlayersById.Get(client.GetUserData().PlayerId)
		if !exist {
			m = utils.NewSafeMap[WSClient, struct{}]()
			s.PlayersById.Set(client.GetUserData().PlayerId, m)
		}
		m.Set(client, struct{}{})

		m, exist = s.PlayersByPackage.Get(client.GetUserData().PackageId)
		if !exist {
			m = utils.NewSafeMap[WSClient, struct{}]()
			s.PlayersByPackage.Set(client.GetUserData().PackageId, m)
		}
		m.Set(client, struct{}{})

		m, exist = s.PlayersByAgent.Get(client.GetUserData().AgentId)
		if !exist {
			m = utils.NewSafeMap[WSClient, struct{}]()
			s.PlayersByAgent.Set(client.GetUserData().AgentId, m)
		}
		m.Set(client, struct{}{})

		m, exist = s.PlayersByChannel.Get(client.GetUserData().ChannelId)
		if !exist {
			m = utils.NewSafeMap[WSClient, struct{}]()
			s.PlayersByChannel.Set(client.GetUserData().ChannelId, m)
		}
		m.Set(client, struct{}{})

	} else {
		m, exist := s.VisitorsByPackage.Get(client.GetUserData().PackageId)
		if !exist {
			m = utils.NewSafeMap[WSClient, struct{}]()
			s.VisitorsByPackage.Set(client.GetUserData().PackageId, m)
		}
		m.Set(client, struct{}{})

		m, exist = s.VisitorsByAgent.Get(client.GetUserData().AgentId)
		if !exist {
			m = utils.NewSafeMap[WSClient, struct{}]()
			s.VisitorsByAgent.Set(client.GetUserData().AgentId, m)
		}
		m.Set(client, struct{}{})

		m, exist = s.VisitorsByChannel.Get(client.GetUserData().ChannelId)
		if !exist {
			m = utils.NewSafeMap[WSClient, struct{}]()
			s.VisitorsByChannel.Set(client.GetUserData().ChannelId, m)
		}
		m.Set(client, struct{}{})

	}
}

func (s *WSServer) processOnDisconnect(event IDisconnectEvent) {
	atomic.AddUint64(&s.connectionNum, ^uint64(0)) // -1
	metric.MircoWebSocketConnectedClient.Dec()

	//slog.DebugContext(s.ctx, "[WSServer] onDisconnect", ClientInfoLog(event.Client, "err", event.Err)...)
}

func (s *WSServer) processOnMessage(event IMessageEvent[websocket.MessageType]) {
	switch event.MessageType() {
	case websocket.TextMessage:
		//slog.DebugContext(s.ctx, "[WSServer] received TextMessage", ClientInfoLog(event.Client, "msg", string(event.Data))...)
	case websocket.BinaryMessage:
		//slog.DebugContext(s.ctx, "[WSServer] received BinaryMessage", ClientInfoLog(event.Client, "msgLength", len(event.Data))...)
	default:
		slog.ErrorContext(s.ctx, "[WSServer] received unknown message type", ClientInfoLog(event.GetClient(), "msgType", event.MessageType)...)
	}

	err := event.GetClient().WriteMessage(websocket.TextMessage, common.PongMsg)
	if err != nil {
		slog.ErrorContext(s.ctx, "[WSServer] write pong message error", ClientInfoLog(event.GetClient(), "err", err)...)
		_ = event.GetClient().Close()
	}
}

func (s *WSServer) release() {
	s.done = true
	if s.httpEngine != nil {
		s.httpEngine.Stop()
	}
	close(s.onConnect)
	close(s.onMessage)
	close(s.onDisconnect)

	for client := range s.onConnect {
		s.processOnConnect(client)
	}

	for event := range s.onMessage {
		s.processOnMessage(event)
	}

	for event := range s.onDisconnect {
		s.processOnDisconnect(event)
	}

	s.ClientBus.Range(func(client WSClient, _ struct{}) bool {
		_ = client.Close()
		return true
	})
}

func (s *WSServer) NextConnectionID() ClientID {
	return s.idAllocator.NextConnectionID()
}

func (s *WSServer) OnConnect() chan WSClient {
	return s.onConnect
}

func (s *WSServer) OnMessage() chan IMessageEvent[websocket.MessageType] {
	return s.onMessage
}

func (s *WSServer) OnDisconnect() chan IDisconnectEvent {
	return s.onDisconnect
}

func (s *WSServer) IsClosed() bool {
	return s.done
}

func (s *WSServer) Execute(context.Context) {
	slog.InfoContext(s.ctx, "[WSServer] start shutdown")
	s.cancel()
	s.release()
	slog.InfoContext(s.ctx, "[WSServer] shutdown done")
}
