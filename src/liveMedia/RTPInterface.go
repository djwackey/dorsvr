package liveMedia

import (
	. "groupsock"
)

type RTPInterface struct {
	gs *GroupSock
    owner interface{}
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

func (this *RTPInterface) sendPacket(packet []byte, packetSize uint) {
	this.gs.Output(string(packet), packetSize, this.gs.TTL())
}
