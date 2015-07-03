package liveMedia

import (
    . "groupsock"
)

type RTPInterface struct {
    gs *GroupSock
}

func NewRTPInterface(gs *GroupSock) *RTPInterface {
    rtpInterface := new(RTPInterface)
    rtpInterface.gs = gs
    return rtpInterface
}

func (this *RTPInterface) startNetworkReading( /*handlerProc interface*/ ) {
}

func (this *RTPInterface) stopNetworkReading() {
}

func (this *RTPInterface) sendPacket(packet []byte, packetSize uint) {
    this.gs.Output(packet, packetSize, this.gs.ttl())
}
