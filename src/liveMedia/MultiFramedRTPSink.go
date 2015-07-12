package liveMedia

import (
	. "groupsock"
)

type MultiFramedRTPSink struct {
	RTPSink
}

func (this *MultiFramedRTPSink) InitMultiFramedRTPSink(rtpGroupSock *GroupSock, rtpPayloadType int,
                                                                                rtpTimestampFrequency uint,
                                                                                rtpPayloadFormatName string) {
	this.InitRTPSink(rtpGroupSock, rtpPayloadType, rtpTimestampFrequency, rtpPayloadFormatName)
}

func (this *MultiFramedRTPSink) continuePlaying() {
	this.buildAndSendPacket()
}

func (this *MultiFramedRTPSink) buildAndSendPacket() {
	this.packFrame()
}

func (this *MultiFramedRTPSink) packFrame() {
	this.afterGettingFrame()
}

func (this *MultiFramedRTPSink) afterGettingFrame() {
	this.sendPacketIfNecessary()
}

func (this *MultiFramedRTPSink) sendPacketIfNecessary() {
	//this.rtpInterface.sendPacket()
}
