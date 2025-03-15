package server

import "github.com/lesismal/nbio/nbhttp/websocket"

func (s *WSServer) Broadcast(messageType websocket.MessageType, message []byte) {
	s.ClientBus.Range(func(c WSClient, _ struct{}) bool {
		_ = c.WriteMessage(messageType, message)
		return true
	})
}

func (s *WSServer) SendByPlayerId(playerId PlayerId, messageType websocket.MessageType, message []byte) {
	if clientRef, ok := s.PlayersById.Get(playerId); ok {
		clientRef.Range(func(c WSClient, _ struct{}) bool {
			_ = c.WriteMessage(messageType, message)
			return true
		})
	}
}

func (s *WSServer) SendByPackageId(packageId PackageId, visitors bool, messageType websocket.MessageType, message []byte) {
	if !visitors {
		if clientRef, ok := s.PlayersByPackage.Get(packageId); ok {
			clientRef.Range(func(c WSClient, _ struct{}) bool {
				_ = c.WriteMessage(messageType, message)
				return true
			})
		}
	} else {
		if clientRef, ok := s.VisitorsByPackage.Get(packageId); ok {
			clientRef.Range(func(c WSClient, _ struct{}) bool {
				_ = c.WriteMessage(messageType, message)
				return true
			})
		}
	}
}

func (s *WSServer) SendByAgentId(agentId AgentId, visitors bool, messageType websocket.MessageType, message []byte) {
	if !visitors {
		if clientRef, ok := s.PlayersByAgent.Get(agentId); ok {
			clientRef.Range(func(c WSClient, _ struct{}) bool {
				_ = c.WriteMessage(messageType, message)
				return true
			})
		}
	} else {
		if clientRef, ok := s.VisitorsByAgent.Get(agentId); ok {
			clientRef.Range(func(c WSClient, _ struct{}) bool {
				_ = c.WriteMessage(messageType, message)
				return true
			})
		}
	}
}

func (s *WSServer) SendByChannelId(channelId ChannelId, visitors bool, messageType websocket.MessageType, message []byte) {
	if !visitors {
		if clientRef, ok := s.PlayersByChannel.Get(channelId); ok {
			clientRef.Range(func(c WSClient, _ struct{}) bool {
				_ = c.WriteMessage(messageType, message)
				return true
			})
		}
	} else {
		if clientRef, ok := s.VisitorsByChannel.Get(channelId); ok {
			clientRef.Range(func(c WSClient, _ struct{}) bool {
				_ = c.WriteMessage(messageType, message)
				return true
			})
		}
	}
}

func (s *WSServer) KickByPlayerId(playerId PlayerId) {
	if clientRef, ok := s.PlayersById.Get(playerId); ok {
		clientRef.Range(func(c WSClient, _ struct{}) bool {
			_ = c.Close()
			return true
		})
	}
}

func (s *WSServer) KickAll() {
	s.ClientBus.Range(func(c WSClient, _ struct{}) bool {
		_ = c.Close()
		return true
	})
}
