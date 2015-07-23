package liveMedia

import (
	"fmt"
    "bytes"
    "encoding/binary"
)

//////// OutPacketBuffer ////////
type OutPacketBuffer struct {
	buff             []byte
    limit            uint
    maxSize          uint
	preferred        uint
	curOffset        uint
    packetStart      uint
	maxPacketSize    uint
	overflowDataSize uint
}

func NewOutPacketBuffer(preferredPacketSize, maxPacketSize uint) *OutPacketBuffer {
	outPacketBuffer := new(OutPacketBuffer)
    outPacketBuffer.maxSize = 60000 // default
	outPacketBuffer.preferred = preferredPacketSize
	outPacketBuffer.maxPacketSize = maxPacketSize

    maxNumPackets := (maxSize - (maxPacketSize - 1)) / maxPacketSize
    outPacketBuffer.limit = maxNumPackets * maxPacketSize
    outPacketBuffer.buff = make([]byte, outPacketBuffer.limit)
    outPacketBuffer.resetOffset()
    outPacketBuffer.resetPacketStart()
    outPacketBuffer.resetOverflowData()
	return outPacketBuffer
}

func (this *OutPacketBuffer) packet() []byte {
	return this.buff[this.packetStart]
}

func (this *OutPacketBuffer) curPtr() []byte {
	return this.buff[this.packetStart + this.curOffset]
}

func (this *OutPacketBuffer) curPacketSize() uint {
	return this.curOffset
}

func (this *OutPacketBuffer) increment(numBytes uint) {
    this.curOffset += numBytes
}

func (this *OutPacketBuffer) haveOverflowData() bool {
	return this.overflowDataSize > 0
}

func (this *OutPacketBuffer) adjustPacketStart(numBytes uint) {
    this.packetStart += numBytes
    if this.overflowDataOffset >= numBytes {
        this.overflowDataOffset -= numBytes
    } else {
        this.overflowDataOffset = 0
        this.overflowDataSize = 0
    }
}

func (this *OutPacketBuffer) totalBytesAvailable() uint {
	return this.limit - (this.packet + this.curOffset)
}

func (this *OutPacketBuffer) enqueue(from []byte, numBytes uint) {
    if numBytes > this.totalBytesAvailable() {
        fmt.Println("OutPacketBuffer::enqueue() warning: %d > %d", numBytes, this.totalBytesAvailable())
        numBytes = this.totalBytesAvailable()
    }

    if string(this.curPtr()) != string(from) {
        this.curPtr() = from
    }
    this.increment(numBytes)
}

func (this *OutPacketBuffer) enqueueWord(word uint) {
    buf := bytes.NewBuffer([]byte{})
    binary.Write(buf, binary.BigEndian, word)
    this.enqueue(buf.Bytes(), 4)
}

func (this *OutPacketBuffer) skipBytes(numBytes uint) {
}

func (this *OutPacketBuffer) resetPacketStart() {
    if this.overflowDataSize > 0 {
        this.overflowDataOffset += this.packetStart
    }
    this.packetStart = 0
}

func (this *OutPacketBuffer) resetOffset() {
    this.curOffset = 0
}

func (this *OutPacketBuffer) resetOverflowData() {
    this.overflowDataSize = 0
    this.overflowDataOffset = 0
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
    // First, tell the source that we're no longer interested:
    if this.source != nil {
        this.source.stopGettingFrames()
    }
}
