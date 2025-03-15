package server

import (
	"context"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/snwsnwsnw/utils/pool"
	"log/slog"
	"wsserver/common"
	"wsserver/wsutil"
)

const (
	minWorkerCount = 10
	maxWorkerCount = 100
	scaleThreshold = 30 // 30% lag
	queueSize      = 1000
)

func RedisSubscribe() {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		slog.ErrorContext(ctx, "連線 Redis 失敗", "err", err)
		return
	}
	slog.InfoContext(ctx, "連線 Redis 成功", "pong", pong)

	subscriber := rdb.Subscribe(ctx, common.KeyWebSocketRedisMessageChannel)
	defer subscriber.Close()

	workPool := pool.NewAutoScaleWorkerPool(
		ctx, minWorkerCount, maxWorkerCount, queueSize, scaleThreshold,
		pool.DefaultIdleTimeout, subscriber.Channel(),
		func(ctx context.Context, message *redis.Message) {
			Server.ProcessRedisMessage(ctx, message.Payload)
		})
	workPool.WG.Wait()
}

func (s *WSServer) ProcessRedisMessage(ctx context.Context, payload string) {
	notification := wsutil.GetSpecificNotification([]byte(payload))

	key := string(notification.WebSocketKey())
	data := notification.DataBytes()

	slog.DebugContext(ctx, "WS receive data", "webSocketKey", key, "trackLog", string(notification.TrackLog()),
		"playerCount", notification.PlayerIdLength(), "channelCount", notification.ChannelIdLength(),
		"packageCount", notification.PackageIdLength(), "agentCount", notification.AgentIdLength(), "payload", string(data))

	switch key {
	case wsutil.WebSocketKeyDirect:
		s.Broadcast(websocket.TextMessage, data)

	case wsutil.WebSocketKeySpecificPlayerId:
		for idx := range notification.PlayerIdLength() {
			playerId := notification.PlayerId(idx)
			s.SendByPlayerId(PlayerId(playerId), websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificPackage:
		for idx := range notification.PackageIdLength() {
			packageId := notification.PackageId(idx)
			s.SendByPackageId(PackageId(packageId), false, websocket.TextMessage, data)
			s.SendByPackageId(PackageId(packageId), true, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificAgentId:
		for idx := range notification.AgentIdLength() {
			agentId := notification.AgentId(idx)
			s.SendByAgentId(AgentId(agentId), false, websocket.TextMessage, data)
			s.SendByAgentId(AgentId(agentId), true, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificChannel:
		for idx := range notification.ChannelIdLength() {
			channelId := notification.ChannelId(idx)
			s.SendByChannelId(ChannelId(channelId), false, websocket.TextMessage, data)
			s.SendByChannelId(ChannelId(channelId), true, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificPackagePlayer:
		for idx := range notification.PackageIdLength() {
			packageId := notification.PackageId(idx)
			s.SendByPackageId(PackageId(packageId), false, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificAgentPlayer:
		for idx := range notification.AgentIdLength() {
			agentId := notification.AgentId(idx)
			s.SendByAgentId(AgentId(agentId), false, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificChannelPlayer:
		for idx := range notification.ChannelIdLength() {
			channelId := notification.ChannelId(idx)
			s.SendByChannelId(ChannelId(channelId), false, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificPackageVisitor:
		for idx := range notification.PackageIdLength() {
			packageId := notification.PackageId(idx)
			s.SendByPackageId(PackageId(packageId), true, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificAgentVisitor:
		for idx := range notification.AgentIdLength() {
			agentId := notification.AgentId(idx)
			s.SendByAgentId(AgentId(agentId), true, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificChannelVisitor:
		for idx := range notification.ChannelIdLength() {
			channelId := notification.ChannelId(idx)
			s.SendByChannelId(ChannelId(channelId), true, websocket.TextMessage, data)
		}

	case wsutil.WebSocketKeySpecificPlayerIdKick:
		for idx := range notification.PlayerIdLength() {
			playerId := notification.PlayerId(idx)
			s.KickByPlayerId(PlayerId(playerId))
		}

	default:
		slog.WarnContext(ctx, "WS receive unknown key", "webSocketKey", key, "trackLog", string(notification.TrackLog()),
			"playerCount", notification.PlayerIdLength(), "channelCount", notification.ChannelIdLength(),
			"packageCount", notification.PackageIdLength(), "agentCount", notification.AgentIdLength(), "payload", string(data))
	}

}
