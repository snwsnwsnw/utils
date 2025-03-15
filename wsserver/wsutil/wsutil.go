package wsutil

import (
	"context"
	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"wsserver/common"
)

func sendWS(ctx context.Context, trackLog string, msg []byte) (err error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		slog.ErrorContext(ctx, "連線 Redis 失敗", "err", err)
		return
	}
	defer rdb.Close()
	_, err = rdb.Publish(ctx, common.KeyWebSocketRedisMessageChannel, msg).Result()
	if err != nil {
		slog.ErrorContext(ctx, "[Websocket] sendWS redis.Publish error", "err", err, "trackLog", trackLog)
	}
	return
}

func Stringify(data any) []byte {
	b, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return b
}

// DirectBroadcast 透過 redis 送出全服廣播的請求 客戶端會收到req轉成json的內容 此func回傳的第一個值為此次redis調用的追蹤串
func DirectBroadcast(ctx context.Context, req any, track TrackLog) (trackLog string, err error) {
	trackLog = track.Log()
	data, err := generateMessage(WSParam{Key: WebSocketKeyDirect, Data: Stringify(req), TrackLog: trackLog})
	if err != nil {
		return trackLog, err
	}
	err = sendWS(ctx, trackLog, data)
	return
}

// SpecificPlayerId 透過 redis 送出指定玩家廣播的請求
func SpecificPlayerId(ctx context.Context, playerIds []int64, req any, track TrackLog) (trackLog string, err error) {
	trackLog = track.Log()
	data, err := generateMessage(WSParam{Key: WebSocketKeySpecificPlayerId, Data: Stringify(req), PlayerIds: playerIds, TrackLog: trackLog})
	if err != nil {
		return trackLog, err
	}
	err = sendWS(ctx, trackLog, data)
	return
}

func sendByAgentId(ctx context.Context, key string, agentIds []int64, req any, track TrackLog) (trackLog string, err error) {
	trackLog = track.Log()
	data, err := generateMessage(WSParam{Key: key, Data: Stringify(req), AgentIds: agentIds, TrackLog: trackLog})
	if err != nil {
		return trackLog, err
	}
	err = sendWS(ctx, trackLog, data)
	return
}

// SpecificAgentId 透過 redis 送出指定代理商廣播的請求 所有連線
func SpecificAgentId(ctx context.Context, agentIds []int64, req any, track TrackLog) (trackLog string, err error) {
	return sendByAgentId(ctx, WebSocketKeySpecificAgentId, agentIds, req, track)
}

// SpecificAgentIdVisitor 透過 redis 送出指定代理商廣播的請求 針對訪客連線
func SpecificAgentIdVisitor(ctx context.Context, agentIds []int64, req any, track TrackLog) (trackLog string, err error) {
	return sendByAgentId(ctx, WebSocketKeySpecificAgentVisitor, agentIds, req, track)
}

// SpecificAgentIdPlayer 透過 redis 送出指定代理商廣播的請求 針對會員連線
func SpecificAgentIdPlayer(ctx context.Context, agentIds []int64, req any, track TrackLog) (trackLog string, err error) {
	return sendByAgentId(ctx, WebSocketKeySpecificAgentPlayer, agentIds, req, track)
}

func sendByPackageId(ctx context.Context, key string, packageIds []int64, req any, track TrackLog) (trackLog string, err error) {
	trackLog = track.Log()
	data, err := generateMessage(WSParam{Key: key, Data: Stringify(req), PackageIds: packageIds, TrackLog: trackLog})
	if err != nil {
		return trackLog, err
	}
	err = sendWS(ctx, trackLog, data)
	return
}

// SpecificPackageId 透過 redis 送出指定包廣播的請求 所有連線
func SpecificPackageId(ctx context.Context, packageIds []int64, req any, track TrackLog) (trackLog string, err error) {
	return sendByPackageId(ctx, WebSocketKeySpecificPackage, packageIds, req, track)
}

// SpecificPackageIdVisitor 透過 redis 送出指定包廣播的請求 針對訪客連線
func SpecificPackageIdVisitor(ctx context.Context, packageIds []int64, req any, track TrackLog) (trackLog string, err error) {
	return sendByPackageId(ctx, WebSocketKeySpecificPackageVisitor, packageIds, req, track)
}

// SpecificPackageIdPlayer 透過 redis 送出指定包廣播的請求 針對會員連線
func SpecificPackageIdPlayer(ctx context.Context, packageIds []int64, req any, track TrackLog) (trackLog string, err error) {
	return sendByPackageId(ctx, WebSocketKeySpecificPackagePlayer, packageIds, req, track)
}

func sendByChannelId(ctx context.Context, key string, channelIds []string, req any, track TrackLog) (trackLog string, err error) {
	trackLog = track.Log()
	data, err := generateMessage(WSParam{Key: key, Data: Stringify(req), ChannelIds: channelIds, TrackLog: trackLog})
	if err != nil {
		return trackLog, err
	}
	err = sendWS(ctx, trackLog, data)
	return
}

// SpecificChannelId 透過 redis 送出指定頻道廣播的請求 所有連線
func SpecificChannelId(ctx context.Context, channelIds []string, req any, track TrackLog) (trackLog string, err error) {
	return sendByChannelId(ctx, WebSocketKeySpecificChannel, channelIds, req, track)
}

// SpecificChannelIdVisitor 透過 redis 送出指定頻道廣播的請求 針對訪客連線
func SpecificChannelIdVisitor(ctx context.Context, channelIds []string, req any, track TrackLog) (trackLog string, err error) {
	return sendByChannelId(ctx, WebSocketKeySpecificChannelVisitor, channelIds, req, track)
}

// SpecificChannelIdPlayer 透過 redis 送出指定頻道廣播的請求 針對會員連線
func SpecificChannelIdPlayer(ctx context.Context, channelIds []string, req any, track TrackLog) (trackLog string, err error) {
	return sendByChannelId(ctx, WebSocketKeySpecificChannelPlayer, channelIds, req, track)
}

// SpecificPlayerIdKick 透過 redis 送出指定玩家踢出的請求 無論是否error trackLog始終回傳log字串
func SpecificPlayerIdKick(ctx context.Context, playerIds []int64, track TrackLog) (trackLog string, err error) {
	trackLog = track.Log()
	data, err := generateMessage(WSParam{Key: WebSocketKeySpecificPlayerIdKick, PlayerIds: playerIds, TrackLog: trackLog})
	if err != nil {
		return trackLog, err
	}
	err = sendWS(ctx, trackLog, data)
	return
}
