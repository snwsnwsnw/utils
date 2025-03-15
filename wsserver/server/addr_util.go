package server

import (
	"log/slog"
	"strconv"
	"strings"
)

const WS_PORT_RANGE = "38001-38050"

var wsAddrs []string

func LoadWSAddrs() {
	sp := strings.Split(WS_PORT_RANGE, "-")
	if len(sp) != 2 {
		slog.Error("WS_PORT_RANGE is invalid")
		return
	}
	start, err := strconv.Atoi(sp[0])
	if err != nil {
		slog.Error("WS_PORT_RANGE is invalid")
		return
	}
	end, err := strconv.Atoi(sp[1])
	if err != nil {
		slog.Error("WS_PORT_RANGE is invalid")
		return
	}
	start, end = min(start, end), max(start, end)
	wsAddrs = make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		wsAddrs = append(wsAddrs, "0.0.0.0:"+strconv.Itoa(i))
	}
}
