package kite

import "sync"

func NewServer() *Server {
	s := &Server{}
	s.init()
	return s
}

var lock sync.Mutex

func RegisterMsgType(mt MsgType, name string) {
	lock.Lock()
	usrMsgTypes[mt] = name
	lock.Unlock()
}
