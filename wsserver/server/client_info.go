package server

import (
	"context"
	"log/slog"
	"time"
)

type ChannelId string
type AgentId int64
type PackageId int64
type PlayerId int64

type ClientInfo struct {
	Visitor   bool
	DeviceId  string
	AppUrl    string
	Token     string
	PlayerId  PlayerId
	PackageId PackageId
	AgentId   AgentId
	ChannelId ChannelId
	Name      string
	UserAgent UserAgentInfo
}

func GetClientInfo(ctx context.Context,
	platform, os, osVersion, browser, browserVersion, deviceId, channelId, ip,
	appUrl, token, fromDomain, devicePlatform, systemVersion string) (wsClient *ClientInfo, errCode int) {

	// fake data
	if time.Now().Unix()%2 == 0 {
		wsClient = &ClientInfo{
			Visitor:   true,
			AppUrl:    appUrl,
			AgentId:   AgentId(111),
			PackageId: PackageId(222),
			ChannelId: ChannelId(channelId),
			DeviceId:  deviceId,
			UserAgent: UserAgentInfo{
				Platform:       platform,
				Os:             os,
				OsVersion:      osVersion,
				Browser:        browser,
				BrowserVersion: browserVersion,
				Ip:             ip,
			},
		}

		slog.DebugContext(ctx, "[GetClientInfo] Visitor info",
			"appUrl", appUrl, "agentId", wsClient.AgentId, "packageId", wsClient.PackageId, "channelId", channelId, "deviceId", deviceId,
			"platform", platform, "os", os, "osVersion", osVersion, "browser", browser, "browserVersion", browserVersion, "ip", ip)
	} else {
		wsClient = &ClientInfo{
			AppUrl:    appUrl,
			Token:     "THIS-IS-A-TOKEN",
			PlayerId:  PlayerId(333),
			AgentId:   AgentId(111),
			PackageId: PackageId(222),
			ChannelId: ChannelId("Channel1"),
			Name:      "userName",
			UserAgent: UserAgentInfo{
				Platform:       platform,
				Os:             os,
				OsVersion:      osVersion,
				Browser:        browser,
				BrowserVersion: browserVersion,
				Ip:             ip,
			},
		}
		slog.DebugContext(ctx, "[GetClientInfo] Player info",
			"appUrl", appUrl, "playerId", wsClient.PlayerId, "agentId", wsClient.AgentId,
			"packageId", wsClient.PackageId, "channelId", channelId, "name", wsClient.Name, "deviceId", deviceId,
			"platform", platform, "os", os, "osVersion", osVersion, "browser", browser, "browserVersion", browserVersion, "ip", ip)
	}

	return wsClient, 0
}
