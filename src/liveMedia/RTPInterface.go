package liveMedia

import (
	. "groupsock"
)

type RTPInterface struct {
	gs                         *GroupSock
	owner                      interface{}
	nextTCPReadStreamSocketNum int
}

func NewRTPInterface(owner interface{}, gs *GroupSock) *RTPInterface {
	rtpInterface := new(RTPInterface)
	rtpInterface.gs = gs
	rtpInterface.owner = owner
	return rtpInterface
}

func (this *RTPInterface) startNetworkReading( /*handlerProc interface*/ ) {
}

func (this *RTPInterface) stopNetworkReading() {
}

func (this *RTPInterface) GS() *GroupSock {
	return this.gs
}

func (this *RTPInterface) sendPacket(packet []byte, packetSize uint) bool {
	return this.gs.Output(packet, packetSize, this.gs.TTL())
}

func (this *RTPInterface) handleRead() bool {
	return true
}
