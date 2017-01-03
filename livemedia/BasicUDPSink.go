package livemedia

import (
	sys "syscall"

	gs "github.com/djwackey/dorsvr/groupsock"
)

type BasicUDPSink struct {
	MediaSink
	gs             *gs.GroupSock
	maxPayloadSize uint
	outputBuffer   []byte
	nextSendTime   sys.Timeval
}

func NewBasicUDPSink(gs *gs.GroupSock) *BasicUDPSink {
	sink := new(BasicUDPSink)
	sink.maxPayloadSize = 1450
	sink.outputBuffer = make([]byte, sink.maxPayloadSize)
	sink.gs = gs
	return sink
}

func (s *BasicUDPSink) ContinuePlaying() {
}
