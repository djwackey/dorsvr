package liveMedia

import (
    . "groupsock"
    . "include"
)

type BasicUDPSink struct {
    gs *GroupSock
    maxPayloadSize uint
    outputBuffer []byte
    nextSendTime timeval
}

func NewBasicUDPSink(gs *GroupSock) *BasicUDPSink {
    udpSink := new(BasicUDPSink)
    udpSink.maxPayloadSize = 1450
    udpSink.outputBuffer = make([]byte, udpSink.maxPayloadSize)
    udpSink.gs = gs
    return udpSink
}

func (this *BasicUDPSink) continuePlaying() {
}
