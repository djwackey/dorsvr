package livemedia

import (
	"bytes"
	"encoding/binary"
	"net"
	sys "syscall"

	"github.com/djwackey/gitea/log"
)

// default
var OutPacketBufferMaxSize uint = 2000000

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

func newOutPacketBuffer(preferredPacketSize, maxPacketSize uint) *OutPacketBuffer {
	maxNumPackets := (OutPacketBufferMaxSize + (maxPacketSize - 1)) / maxPacketSize
	limit := maxNumPackets * maxPacketSize

	b := &OutPacketBuffer{
		limit:         limit,
		maxPacketSize: maxPacketSize,
		preferred:     preferredPacketSize,
		buff:          make([]byte, limit),
	}

	b.resetOffset()
	b.resetPacketStart()
	b.resetOverflowData()
	return b
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

func (b *OutPacketBuffer) setOverflowData(overflowDataOffset, overflowDataSize,
	durationInMicroseconds uint, presentationTime sys.Timeval) {
	b.overflowDataSize = overflowDataSize
	b.overflowDataOffset = overflowDataOffset
	b.overflowPresentationTime = presentationTime
	b.overflowDurationInMicroseconds = durationInMicroseconds
}

func (b *OutPacketBuffer) useOverflowData() {
	b.enqueue(b.buff[(b.packetStart+b.overflowDataOffset):], b.overflowDataSize)
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
		log.Warn("OutPacketBuffer::enqueue() warning: %d > %d\n", numBytes, b.totalBytesAvailable())
		numBytes = b.totalBytesAvailable()
	}

	if !bytes.Equal(b.curPtr()[:numBytes], from[:numBytes]) {
		copy(b.curPtr(), from[:numBytes])
	}
	b.increment(numBytes)
}

func (b *OutPacketBuffer) enqueueWord(word uint32) {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, word)
	if err != nil {
		log.Error(0, "[OutPacketBuffer::enqueueWord] Failed to enqueueWord.%s", err.Error())
		return
	}
	b.enqueue(buff.Bytes(), 4)
}

func (b *OutPacketBuffer) insert(from []byte, numBytes, toPosition uint) {
	realToPosition := b.packetStart + toPosition
	if realToPosition+numBytes > b.limit {
		if realToPosition > b.limit {
			return // we can't do this
		}
		numBytes = b.limit - realToPosition
	}

	copy(b.buff[realToPosition:], from[:numBytes])
	if toPosition+numBytes > b.curOffset {
		b.curOffset = toPosition + numBytes
	}
}

func (b *OutPacketBuffer) insertWord(word uint32, toPosition uint) {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, word)
	if err == nil {
		b.insert(buff.Bytes(), 4, toPosition)
	}
}

func (b *OutPacketBuffer) wouldOverflow(numBytes uint) bool {
	return (b.curOffset + numBytes) > b.maxPacketSize
}

func (b *OutPacketBuffer) numOverflowBytes(numBytes uint) uint {
	return (b.curOffset + numBytes) - b.maxPacketSize
}

func (b *OutPacketBuffer) isTooBigForAPacket(numBytes uint) bool {
	return numBytes > b.maxPacketSize
}

func (b *OutPacketBuffer) extract(to []byte, numBytes, fromPosition uint) {
	realFromPosition := b.packetStart + fromPosition
	if realFromPosition+numBytes > b.limit {
		if realFromPosition > b.limit {
			return
		}
		numBytes = b.limit - realFromPosition
	}
	copy(to, b.buff[realFromPosition:realFromPosition+numBytes])
}

func (b *OutPacketBuffer) extractWord(fromPosition uint) uint32 {
	word := make([]byte, 4)
	b.extract(word, 4, fromPosition)

	return binary.BigEndian.Uint32(word)
}

func (b *OutPacketBuffer) skipBytes(numBytes uint) {
	if numBytes > b.totalBytesAvailable() {
		numBytes = b.totalBytesAvailable()
	}

	b.increment(numBytes)
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
	AuxSDPLine() string
	rtpmapLine() string
	sdpMediaType() string
	enableRTCPReports() bool
	nextTimestampHasBeenPreset() bool
	StartPlaying(source IFramedSource, afterFunc interface{}) bool
	StopPlaying()
	ContinuePlaying()
	destroy()
	ssrc() uint32
	octetCount() uint
	packetCount() uint
	currentSeqNo() uint32
	rtpPayloadType() uint32
	presetNextTimestamp() uint32
	convertToRTPTimestamp(tv sys.Timeval) uint32
	transmissionStatsDB() *RTPTransmissionStatsDB
	addStreamSocket(socketNum net.Conn, streamChannelID uint)
	delStreamSocket(socketNum net.Conn, streamChannelID uint)
	setServerRequestAlternativeByteHandler(socketNum net.Conn, handler interface{})
	frameCanAppearAfterPacketStart(frameStart []byte, numBytesInFrame uint) bool
	doSpecialFrameHandling(fragmentationOffset, numBytesInFrame, numRemainingBytes uint,
		frameStart []byte, framePresentationTime sys.Timeval)
}

type MediaSink struct {
	Source    IFramedSource
	rtpSink   IMediaSink
	afterFunc interface{}
}

func (s *MediaSink) InitMediaSink(rtpSink IMediaSink) {
	s.rtpSink = rtpSink
}

func (s *MediaSink) StartPlaying(source IFramedSource, afterFunc interface{}) bool {
	if s.Source != nil {
		log.Error(1, "This sink is already being played")
		return false
	}

	if s.rtpSink == nil {
		log.Error(1, "This RTP Sink is nil")
		return false
	}

	s.Source = source
	s.afterFunc = afterFunc
	s.rtpSink.ContinuePlaying()
	return true
}

func (s *MediaSink) StopPlaying() {
	// First, tell the source that we're no longer interested:
	if s.Source != nil {
		s.Source.stopGettingFrames()
	}

	s.Source = nil
	s.afterFunc = nil
}

func (s *MediaSink) OnSourceClosure() {
	if s.afterFunc != nil {
		s.afterFunc.(func())()
	}
}

func (s *MediaSink) addStreamSocket(socketNum net.Conn, streamChannelID uint) {}
func (s *MediaSink) delStreamSocket(socketNum net.Conn, streamChannelID uint) {}
func (s *MediaSink) frameCanAppearAfterPacketStart(frameStart []byte, numBytesInFrame uint) bool {
	return false
}
func (s *MediaSink) doSpecialFrameHandling(fragmentationOffset, numBytesInFrame, numRemainingBytes uint, frameStart []byte, framePresentationTime sys.Timeval) {
}
func (s *MediaSink) setServerRequestAlternativeByteHandler(socketNum net.Conn, handler interface{}) {}

func (s *MediaSink) nextTimestampHasBeenPreset() bool             { return true }
func (s *MediaSink) enableRTCPReports() bool                      { return true }
func (s *MediaSink) AuxSDPLine() string                           { return "" }
func (s *MediaSink) rtpmapLine() string                           { return "" }
func (s *MediaSink) sdpMediaType() string                         { return "" }
func (s *MediaSink) presetNextTimestamp() uint32                  { return 0 }
func (s *MediaSink) convertToRTPTimestamp(tv sys.Timeval) uint32  { return 0 }
func (s *MediaSink) transmissionStatsDB() *RTPTransmissionStatsDB { return nil }
func (s *MediaSink) rtpPayloadType() uint32                       { return 0 }
func (s *MediaSink) currentSeqNo() uint32                         { return 0 }
func (s *MediaSink) packetCount() uint                            { return 0 }
func (s *MediaSink) octetCount() uint                             { return 0 }
func (s *MediaSink) ssrc() uint32                                 { return 0 }
func (s *MediaSink) destroy()                                     {}
