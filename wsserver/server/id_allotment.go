package server

import "sync/atomic"

type IIdGenerator interface {
	NextConnectionID() ClientID
}

type ClientID uint64

type idAllotment struct {
	connIDAllotment ClientID
}

func (cid *idAllotment) NextConnectionID() ClientID {
	u := uint64(cid.connIDAllotment)
	cid.connIDAllotment = ClientID(atomic.AddUint64(&u, 1))
	return cid.connIDAllotment
}
