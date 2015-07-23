package liveMedia

import (
	"fmt"
	. "groupsock"
	//"time"
)

var rtpHeaderSize int = 12

type MultiFramedRTPSink struct {
	RTPSink
	outBuf                *OutPacketBuffer
	ourMaxPacketSize      uint
	timestampPosition     uint
	specialHeaderSize     uint
	specialHeaderPosition uint
}

func (this *MultiFramedRTPSink) InitMultiFramedRTPSink(rtpSink IRTPSink, rtpGroupSock *GroupSock, rtpPayloadType, rtpTimestampFrequency uint, rtpPayloadFormatName string) {
	// Default max packet size (1500, minus allowance for IP, UDP, UMTP headers)
	// (Also, make it a multiple of 4 bytes, just in case that matters.)
	this.setPacketSizes(1000, 1448)
	this.InitRTPSink(rtpSink, rtpGroupSock, rtpPayloadType, rtpTimestampFrequency, rtpPayloadFormatName)
}

func (this *MultiFramedRTPSink) setPacketSizes(preferredPacketSize, maxPacketSize uint) {
	this.outBuf = NewOutPacketBuffer(preferredPacketSize, maxPacketSize)
	this.ourMaxPacketSize = maxPacketSize
}

func (this *MultiFramedRTPSink) multiFramedPlaying() {
	fmt.Println("MultiFramedRTPSink::continuePlaying")
	this.buildAndSendPacket()
}

func (this *MultiFramedRTPSink) buildAndSendPacket() {
	var rtpHdr uint = 0x80000000
	rtpHdr |= this.rtpPayloadType << 16
	rtpHdr |= this.seqNo
	this.outBuf.enqueueWord(rtpHdr)

	this.timestampPosition = this.outBuf.curPacketSize()
	this.outBuf.skipBytes(4)

	this.outBuf.enqueueWord(this.SSRC())

	// Allow for a special, payload-format-specific header following the
	// RTP header:
	this.specialHeaderPosition = this.outBuf.curPacketSize()
	this.specialHeaderSize = this.SpecialHeaderSize()
	this.outBuf.skipBytes(this.specialHeaderSize)

	this.packFrame()
}

func (this *MultiFramedRTPSink) packFrame() {
	if this.outBuf.haveOverflowData() {
		this.afterGettingFrame()
	} else {
		// Normal case: we need to read a new frame from the source
		if this.source == nil {
			return
		}
		this.source.getNextFrame(this.outBuf.curPtr(), this.outBuf.totalBytesAvailable(), this.afterGettingFrame)
	}
}

func (this *MultiFramedRTPSink) afterGettingFrame() {
	this.sendPacketIfNecessary()
}

func (this *MultiFramedRTPSink) sendPacketIfNecessary() {
	for {
		if !this.rtpInterface.sendPacket(this.outBuf.packet(), this.outBuf.curPacketSize()) {
			// if failure handler has been specified, call it
		}

		fmt.Println("sendPacketIfNecessary")
		//time.Sleep(2 * time.Second)
	}
}

func (this *MultiFramedRTPSink) SpecialHeaderSize() uint {
	// default implementation: Assume no special header:
	return 0
}
