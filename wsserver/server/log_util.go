package server

import (
	"log/slog"
	"net"
)

func ClientInfoLog(c WSClient, args ...any) (data []any) {
	data = make([]any, 0, 7+len(args)/2)

	port := ""
	if !c.IsClosed() {
		_, port, _ = net.SplitHostPort(c.GetConn().LocalAddr().String())
	}

	data = append(data,
		slog.Uint64("cId", uint64(c.GetId())),
		slog.Int64("agentId", int64(c.GetUserData().AgentId)),
		slog.Int64("playerId", int64(c.GetUserData().PlayerId)),
		slog.Int64("packageId", int64(c.GetUserData().PackageId)),
		slog.String("channelId", string(c.GetUserData().ChannelId)),
		slog.Bool("visitor", c.GetUserData().Visitor),
		slog.String("port", port),
	)

	for i := 0; i < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			data = append(data, slog.Any(key, args[i+1]))
		}
	}

	return data
}
