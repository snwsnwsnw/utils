package server

type DisconnectEvent struct {
	client WSClient
	err    error
}

func (e *DisconnectEvent) GetClient() WSClient {
	return e.client
}

func (e *DisconnectEvent) Error() error {
	return e.err
}

type MessageEvent[MT MsgType] struct {
	client      WSClient
	messageType MT
	data        []byte
}

func (e *MessageEvent[MT]) GetClient() WSClient {
	return e.client
}

func (e *MessageEvent[MT]) MessageType() MT {
	return e.messageType
}

func (e *MessageEvent[MT]) Data() []byte {
	return e.data
}
