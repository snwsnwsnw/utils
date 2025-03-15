package wsutil

import (
	"fmt"
	"github.com/google/flatbuffers/go"
	"sync"
	"wsserver/wsutil/internal"
)

// flatc -o ../ --go specificNotification.fbs

var builderPool = sync.Pool{
	New: func() interface{} {
		// 50Kb
		return flatbuffers.NewBuilder(51200)
	},
}

type WSParam struct {
	Key        string
	Data       []byte
	TrackLog   string
	PlayerIds  []int64
	ChannelIds []string
	PackageIds []int64
	AgentIds   []int64
}

func generateMessage(param WSParam) ([]byte, error) {
	builder := builderPool.Get().(*flatbuffers.Builder)
	builder.Reset()
	defer builderPool.Put(builder)

	if len(param.Key) == 0 {
		return nil, fmt.Errorf("webSocketKey is empty")
	}

	webSocketKeyOffset := builder.CreateString(param.Key)
	dataVector := builder.CreateByteVector(param.Data)
	playerVector := createInt64Vector(builder, param.PlayerIds, internal.SpecificNotificationStartPlayerIdVector)
	channelVector := createStringVector(builder, param.ChannelIds)
	packageVector := createInt64Vector(builder, param.PackageIds, internal.SpecificNotificationStartPackageIdVector)
	agentVector := createInt64Vector(builder, param.AgentIds, internal.SpecificNotificationStartAgentIdVector)
	trackLogOffset := builder.CreateString(param.TrackLog)

	internal.SpecificNotificationStart(builder)
	internal.SpecificNotificationAddTrackLog(builder, trackLogOffset)
	internal.SpecificNotificationAddPlayerId(builder, playerVector)
	internal.SpecificNotificationAddChannelId(builder, channelVector)
	internal.SpecificNotificationAddPackageId(builder, packageVector)
	internal.SpecificNotificationAddAgentId(builder, agentVector)
	internal.SpecificNotificationAddWebSocketKey(builder, webSocketKeyOffset)
	internal.SpecificNotificationAddData(builder, dataVector)

	builder.Finish(internal.SpecificNotificationEnd(builder))
	return builder.FinishedBytes(), nil
}

func GeneratePlayerInfoCache(playerId int64, realName string, agentId, packageId int64, channelId, loginDeviceId string) ([]byte, error) {
	builder := builderPool.Get().(*flatbuffers.Builder)
	builder.Reset()
	defer builderPool.Put(builder)

	realNameOffset := builder.CreateString(realName)
	channelIdOffset := builder.CreateString(channelId)
	loginDeviceIdOffset := builder.CreateString(loginDeviceId)

	internal.PlayerInfoCacheStart(builder)
	internal.PlayerInfoCacheAddPlayerId(builder, playerId)
	internal.PlayerInfoCacheAddRealName(builder, realNameOffset)
	internal.PlayerInfoCacheAddAgentId(builder, agentId)
	internal.PlayerInfoCacheAddPackageId(builder, packageId)
	internal.PlayerInfoCacheAddChannelId(builder, channelIdOffset)
	internal.PlayerInfoCacheAddLoginDeviceId(builder, loginDeviceIdOffset)
	builder.Finish(internal.PlayerInfoCacheEnd(builder))

	return builder.FinishedBytes(), nil
}

func createStringVector(builder *flatbuffers.Builder, strs []string) flatbuffers.UOffsetT {
	count := len(strs)
	offsets := make([]flatbuffers.UOffsetT, count)
	for i, s := range strs {
		offsets[i] = builder.CreateString(s)
	}
	// 反向插入向量
	internal.SpecificNotificationStartChannelIdVector(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	return builder.EndVector(count)
}

func createInt64Vector(builder *flatbuffers.Builder, values []int64, startFunc func(*flatbuffers.Builder, int) flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	count := len(values)
	startFunc(builder, count)
	for i := count - 1; i >= 0; i-- {
		builder.PrependInt64(values[i])
	}
	return builder.EndVector(count)
}

func GetSpecificNotification(buf []byte) *internal.SpecificNotification {
	return internal.GetRootAsSpecificNotification(buf, 0)
}

func GetPlayerInfoCache(buf []byte) *internal.PlayerInfoCache {
	return internal.GetRootAsPlayerInfoCache(buf, 0)
}
