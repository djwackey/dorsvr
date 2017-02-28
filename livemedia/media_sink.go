package livemedia

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	sys "syscall"
)

var OutPacketBufferMaxSize uint = 60000 // default

//////// OutPacketBuffer ////////
type OutPacketBuffer struct {
	buff                           []byte
	limit                          uint
	preferred                      uint
	curOffset                      uint
	packetStart                    uint
	maxPacketSize                  uint
	overflowDataSize               uint
	overflowDataOffset             uint
	overflowDurationInMicroseconds uint
	overflowPresentationTime       sys.Timeval
}

func NewOutPacketBuffer(preferredPacketSize, maxPacketSize uint) *OutPacketBuffer {
	outPacketBuffer := new(OutPacketBuffer)
	outPacketBuffer.preferred = preferredPacketSize
	outPacketBuffer.maxPacketSize = maxPacketSize

	maxNumPackets := (OutPacketBufferMaxSize - (maxPacketSize - 1)) / maxPacketSize
	outPacketBuffer.limit = maxNumPackets * maxPacketSize
	outPacketBuffer.buff = make([]byte, outPacketBuffer.limit)
	outPacketBuffer.resetOffset()
	outPacketBuffer.resetPacketStart()
	outPacketBuffer.resetOverflowData()
	return outPacketBuffer
}

func (b *OutPacketBuffer) packet() []byte {
	return b.buff[b.packetStart:]
}

func (b *OutPacketBuffer) curPtr() []byte {
	return b.buff[(b.packetStart + b.curOffset):]
}

func (b *OutPacketBuffer) curPacketSize() uint {
	return b.curOffset
}

func (b *OutPacketBuffer) totalBufferSize() uint {
	return b.limit
}

func (b *OutPacketBuffer) increment(numBytes uint) {
	b.curOffset += numBytes
}

func (b *OutPacketBuffer) haveOverflowData() bool {
	return b.overflowDataSize > 0
}

func (b *OutPacketBuffer) isPreferredSize() bool {
	return b.curOffset >= b.preferred
}

func (b *OutPacketBuffer) useOverflowData() {
	b.enqueue(b.buff[(b.packetStart+b.overflowDataOffset):], b.overflowDataSize)
}

func (b *OutPacketBuffer) OverflowDataSize() uint {
	return b.overflowDataSize
}

func (b *OutPacketBuffer) OverflowPresentationTime() sys.Timeval {
	return b.overflowPresentationTime
}

func (b *OutPacketBuffer) OverflowDurationInMicroseconds() uint {
	return b.overflowDurationInMicroseconds
}

func (b *OutPacketBuffer) adjustPacketStart(numBytes uint) {
	b.packetStart += numBytes
	if b.overflowDataOffset >= numBytes {
		b.overflowDataOffset -= numBytes
	} else {
		b.overflowDataOffset = 0
		b.overflowDataSize = 0
	}
}

func (b *OutPacketBuffer) totalBytesAvailable() uint {
	return b.limit - (b.packetStart + b.curOffset)
}

func (b *OutPacketBuffer) enqueue(from []byte, numBytes uint) {
	if numBytes > b.totalBytesAvailable() {
		fmt.Printf("OutPacketBuffer::enqueue() warning: %d > %d\n", numBytes, b.totalBytesAvailable())
		numBytes = b.totalBytesAvailable()
	}

	if string(b.curPtr()) != string(from) {
		//b.curPtr() = from
	}
	b.increment(numBytes)
}

func (b *OutPacketBuffer) enqueueWord(word uint) {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf, binary.BigEndian, word)
	b.enqueue(buf.Bytes(), 4)
}

func (b *OutPacketBuffer) insert(from []byte, numBytes, toPosition uint) {
	realToPosition := b.packetStart + toPosition
	if realToPosition+numBytes > b.limit {
		if realToPosition > b.limit {
			return // we can't do this
		}
		numBytes = b.limit - realToPosition
	}

	//memmove(&fBuf[realToPosition], from, numBytes)
	if toPosition+numBytes > b.curOffset {
		b.curOffset = toPosition + numBytes
	}
}

func (b *OutPacketBuffer) insertWord(word byte, toPosition uint) {
}

func (b *OutPacketBuffer) wouldOverflow(numBytes uint) bool {
	return (b.curOffset + numBytes) > b.maxPacketSize
}

func (b *OutPacketBuffer) skipBytes(numBytes uint) {
}

func (b *OutPacketBuffer) resetPacketStart() {
	if b.overflowDataSize > 0 {
		b.overflowDataOffset += b.packetStart
	}
	b.packetStart = 0
}

func (b *OutPacketBuffer) resetOffset() {
	b.curOffset = 0
}

func (b *OutPacketBuffer) resetOverflowData() {
	b.overflowDataSize = 0
	b.overflowDataOffset = 0
}

//////// MediaSink ////////
type IMediaSink interface {
	StartPlaying(source IFramedSource) bool
}

type MediaSink struct {
	Source  IFramedSource
	rtpSink IRTPSink
}

func (s *MediaSink) InitMediaSink(rtpSink IRTPSink) {
	s.rtpSink = rtpSink
}

func (s *MediaSink) StartPlaying(source IFramedSource) bool {
	if s.Source != nil {
		fmt.Println("This sink is already being played")
		return false
	}

	if s.rtpSink == nil {
		fmt.Println("This RTP Sink is nil")
		return false
	}

	s.Source = source
	s.rtpSink.ContinuePlaying()
	return true
}

func (s *MediaSink) StopPlaying() {
	// First, tell the source that we're no longer interested:
	if s.Source != nil {
		s.Source.stopGettingFrames()
	}
}

func (s *MediaSink) AuxSDPLine() string {
	return ""
}

func (s *MediaSink) RtpPayloadType() uint {
	return 0
}

func (s *MediaSink) RtpmapLine() string {
	return ""
}

func (s *MediaSink) SdpMediaType() string {
	return ""
}

func (s *MediaSink) OnSourceClosure() {
}

func (s *MediaSink) addStreamSocket(sockNum net.Conn, streamChannelID uint) {
	return
}

func (s *MediaSink) delStreamSocket() {
}

func (s *MediaSink) currentSeqNo() uint {
	return 0
}

func (s *MediaSink) presetNextTimestamp() uint {
	return 0
}
