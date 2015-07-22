package liveMedia

import (
	"fmt"
    "bytes"
    "encoding/binary"
)

// allow for some possibly large H.264 frames
var maxSize uint = 100000

//////// OutPacketBuffer ////////
type OutPacketBuffer struct {
	buff             []byte
	preferred        uint
	curOffset        uint
	maxPacketSize    uint
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
	return 1024
}

func (this *OutPacketBuffer) enqueue(from []byte, numBytes uint) {
    if numBytes > this.totalBytesAvailable() {
        fmt.Println("OutPacketBuffer::enqueue() warning: %d > %d", numBytes, this.totalBytesAvailable())
        numBytes = this.totalBytesAvailable()
    }
}

func (this *OutPacketBuffer) enqueueWord(word uint) {
    buf := bytes.NewBuffer([]byte{})
    binary.Write(buf, binary.BigEndian, word)
    this.enqueue(buf.Bytes(), 4)
}

func (this *OutPacketBuffer) skipBytes(numBytes uint) {
}


//////// MediaSink ////////
type MediaSink struct {
	source  IFramedSource
	rtpSink IRTPSink
}

func (this *MediaSink) InitMediaSink(rtpSink IRTPSink) {
	this.rtpSink = rtpSink
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
