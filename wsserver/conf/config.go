package conf

var WSConfig WSConf = WSConf{
	MaxConnection: 0,
	PingTimeOut:   180, //second
}

type WSConf struct {
	MaxConnection uint64
	PingTimeOut   int
}
