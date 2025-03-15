package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MircoWebSocketConnectedClient = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mirco_websocket_connected_client",
		Help: "The current number of connected websocket client",
	})

	//MircoWebSocketPingPongConsumed = promauto.NewGauge(prometheus.GaugeOpts{
	//	Name: "mirco_websocket_ping_pong_consumed",
	//	Help: "The consume of websocket ping pong",
	//})
)
