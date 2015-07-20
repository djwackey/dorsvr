package liveMedia

import (
	"fmt"
)

// allow for some possibly large H.264 frames
var maxSize uint = 100000

type MediaSink struct {
	source  IFramedSource
	rtpSink IRTPSink
}

func (this *MediaSink) InitMediaSink(rtpSink IRTPSink) {
	this.rtpSink = rtpSink
}

//////// OutPacketBuffer ////////
type OutPacketBuffer struct {
	buff []byte
    preferred uint
	curOffset uint
    maxPacketSize uint
    overflowDataSize uint
}

func NewOutPacketBuffer(preferredPacketSize, maxPacketSize uint) *OutPacketBuffer {
    outPacketBuffer := new(OutPacketBuffer)
    outPacketBuffer.preferred = preferredPacketSize
    outPacketBuffer.maxPacketSize = maxPacketSize
    return outPacketBuffer
}

func (this *OutPacketBuffer) packet() []byte {
	return this.buff
}

func (this *OutPacketBuffer) curPtr() []byte {
	return this.buff
}

func (this *OutPacketBuffer) curPacketSize() uint {
	return this.curOffset
}

func (this *OutPacketBuffer) haveOverflowData() bool {
    return this.overflowDataSize > 0
}

func (this *OutPacketBuffer) totalBytesAvailable() uint {
    return 0
}

func (this *MediaSink) startPlaying(source IFramedSource) bool {
	if this.source != nil {
		fmt.Println("This sink is already being played")
		return false
	}

	this.source = source
	this.rtpSink.continuePlaying()
	return true
}

func (this *MediaSink) stopPlaying() {
}
