package livemedia

import (
	sys "syscall"
	"time"

	gs "github.com/djwackey/dorsvr/groupsock"
)

const rtpHeaderSize uint = 12

type MultiFramedRTPSink struct {
	RTPSink
	outBuf                          *OutPacketBuffer
	nextSendTime                    sys.Timeval
	noFramesLeft                    bool
	isFirstPacket                   bool
	currentTimestamp                uint32
	ourMaxPacketSize                uint
	timestampPosition               uint
	specialHeaderSize               uint
	numFramesUsedSoFar              uint
	specialHeaderPosition           uint
	curFragmentationOffset          uint
	curFrameSpecificHeaderSize      uint
	totalFrameSpecificHeaderSizes   uint
	curFrameSpecificHeaderPosition  uint
	previousFrameEndedFragmentation bool
	onSendErrorFunc                 interface{}
}

func (s *MultiFramedRTPSink) InitMultiFramedRTPSink(rtpSink IMediaSink,
	rtpGroupSock *gs.GroupSock, rtpPayloadType, rtpTimestampFrequency uint32, rtpPayloadFormatName string) {
	// Default max packet size (1500, minus allowance for IP, UDP, UMTP headers)
	// (Also, make it a multiple of 4 bytes, just in case that matters.)
	s.setPacketSizes(1000, 1448)
	s.InitRTPSink(rtpSink, rtpGroupSock, rtpPayloadType, rtpTimestampFrequency, rtpPayloadFormatName)
}

func (s *MultiFramedRTPSink) setPacketSizes(preferredPacketSize, maxPacketSize uint) {
	s.outBuf = newOutPacketBuffer(preferredPacketSize, maxPacketSize)
	s.ourMaxPacketSize = maxPacketSize
}

func (s *MultiFramedRTPSink) multiFramedPlaying() {
	s.buildAndSendPacket(true)
}

func (s *MultiFramedRTPSink) buildAndSendPacket(isFirstPacket bool) {
	s.isFirstPacket = isFirstPacket

	// Set up the RTP header:
	var rtpHdr uint32 = 0x80000000
	rtpHdr |= s._rtpPayloadType << 16
	rtpHdr |= s.seqNo
	s.outBuf.enqueueWord(rtpHdr)

	s.timestampPosition = s.outBuf.curPacketSize()
	s.outBuf.skipBytes(4)

	s.outBuf.enqueueWord(s._ssrc)

	// Allow for a special, payload-format-specific header following the RTP header:
	s.specialHeaderPosition = s.outBuf.curPacketSize()
	s.specialHeaderSize = s.SpecialHeaderSize()
	s.outBuf.skipBytes(s.specialHeaderSize)

	// Begin packing as many (complete) frames into the packet as we can:
	s.noFramesLeft = false
	s.numFramesUsedSoFar = 0
	s.totalFrameSpecificHeaderSizes = 0

	s.packFrame()
}

func (s *MultiFramedRTPSink) packFrame() {
	if s.outBuf.haveOverflowData() {
		// Use this frame before reading a new one from the source
		frameSize := s.outBuf.overflowDataSize
		presentationTime := s.outBuf.overflowPresentationTime
		durationInMicroseconds := s.outBuf.overflowDurationInMicroseconds

		s.outBuf.useOverflowData()
		s.afterGettingFrame(frameSize, durationInMicroseconds, presentationTime)
	} else {
		// Normal case: we need to read a new frame from the source
		if s.Source == nil {
			return
		}

		s.curFrameSpecificHeaderPosition = s.outBuf.curPacketSize()
		s.curFrameSpecificHeaderSize = s.frameSpecificHeaderSize()
		s.outBuf.skipBytes(s.curFrameSpecificHeaderSize)
		s.totalFrameSpecificHeaderSizes += s.curFrameSpecificHeaderSize

		// H264FUAFragmenter
		s.Source.GetNextFrame(s.outBuf.curPtr(), s.outBuf.totalBytesAvailable(),
			s.afterGettingFrame, s.ourHandlerClosure)
	}
}

func (s *MultiFramedRTPSink) afterGettingFrame(frameSize, durationInMicroseconds uint, presentationTime sys.Timeval) {
	if s.isFirstPacket {
		// Record the fact that we're starting to play now:
		sys.Gettimeofday(&s.nextSendTime)
	}

	curFragmentationOffset := s.curFragmentationOffset
	numFrameBytesToUse := frameSize
	var overflowBytes uint

	if s.numFramesUsedSoFar > 0 {
		if s.previousFrameEndedFragmentation &&
			!s.allowOtherFramesAfterLastFragment() &&
			!s.rtpSink.frameCanAppearAfterPacketStart(s.outBuf.curPtr(), frameSize) {
			numFrameBytesToUse = 0
			s.outBuf.setOverflowData(s.outBuf.curPacketSize(), frameSize, durationInMicroseconds, presentationTime)
		}
	}
	s.previousFrameEndedFragmentation = false

	if numFrameBytesToUse > 0 {
		// Check whether this frame overflows the packet
		if s.outBuf.wouldOverflow(frameSize) {
			if s.isTooBigForAPacket(frameSize) && (s.numFramesUsedSoFar == 0 || s.allowOtherFramesAfterLastFragment()) {
				overflowBytes = s.computeOverflowForNewFrame(frameSize)
				numFrameBytesToUse -= overflowBytes
				s.curFragmentationOffset += numFrameBytesToUse
			} else {
				overflowBytes = frameSize
				numFrameBytesToUse = 0
			}
			s.outBuf.setOverflowData(s.outBuf.curPacketSize()+numFrameBytesToUse, overflowBytes, durationInMicroseconds, presentationTime)
		} else if s.curFragmentationOffset > 0 {
			s.curFragmentationOffset = 0
			s.previousFrameEndedFragmentation = true
		}
	}

	if numFrameBytesToUse == 0 && frameSize > 0 {
		// Send our packet now, because we have filled it up:
		s.sendPacketIfNecessary()
	} else {
		// Use this frame in our outgoing packet:
		frameStart := s.outBuf.curPtr()
		s.outBuf.increment(numFrameBytesToUse)
		// do this now, in case "doSpecialFrameHandling()" calls "setFramePadding()" to append padding bytes

		// Here's where any payload format specific processing gets done:
		s.rtpSink.doSpecialFrameHandling(curFragmentationOffset, numFrameBytesToUse, overflowBytes, frameStart, presentationTime)

		s.numFramesUsedSoFar++

		// Update the time at which the next packet should be sent, based
		// on the duration of the frame that we just packed into it.
		// However, if this frame has overflow data remaining, then don't
		// count its duration yet.
		if overflowBytes == 0 {
			s.nextSendTime.Usec += int64(durationInMicroseconds)
			s.nextSendTime.Sec += s.nextSendTime.Usec / 1000000
			s.nextSendTime.Usec %= 1000000
		}

		// Send our packet now if (i) it's already at our preferred size, or
		// (ii) (heuristic) another frame of the same size as the one we just
		//      read would overflow the packet, or
		// (iii) it contains the last fragment of a fragmented frame, and we
		//      don't allow anything else to follow this or
		// (iv) one frame per packet is allowed:
		if s.outBuf.isPreferredSize() ||
			s.outBuf.wouldOverflow(numFrameBytesToUse) ||
			s.previousFrameEndedFragmentation && !s.allowOtherFramesAfterLastFragment() ||
			!s.rtpSink.frameCanAppearAfterPacketStart(s.outBuf.curPtr(), frameSize) {
			// The packet is ready to be sent now
			s.sendPacketIfNecessary()
		} else {
			// There's room for more frames; try getting another:
			s.packFrame()
		}
	}
}

func (s *MultiFramedRTPSink) isTooBigForAPacket(numBytes uint) bool {
	numBytes += rtpHeaderSize + s.SpecialHeaderSize() + s.frameSpecificHeaderSize()
	return s.outBuf.isTooBigForAPacket(numBytes)
}

func (s *MultiFramedRTPSink) sendPacketIfNecessary() {
	if s.numFramesUsedSoFar > 0 {
		if !s.rtpInterface.sendPacket(s.outBuf.packet(), s.outBuf.curPacketSize()) {
			// if failure handler has been specified, call it
			if s.onSendErrorFunc != nil {
			}
		}

		s._packetCount++
		s.totalOctetCount += s.outBuf.curPacketSize()
		s._octetCount += s.outBuf.curPacketSize() - rtpHeaderSize - s.specialHeaderSize - s.totalFrameSpecificHeaderSizes

		s.seqNo++ // for next time
	}

	if s.outBuf.haveOverflowData() &&
		s.outBuf.totalBytesAvailable() > s.outBuf.totalBufferSize()/2 {
		// Efficiency hack: Reset the packet start pointer to just in front of
		// the overflow data (allowing for the RTP header and special headers),
		// so that we probably don't have to "memmove()" the overflow data
		// into place when building the next packet:
		newPacketStart := s.outBuf.curPacketSize() - (rtpHeaderSize + s.specialHeaderSize + s.frameSpecificHeaderSize())
		s.outBuf.adjustPacketStart(newPacketStart)
	} else {
		// Normal case: Reset the packet start pointer back to the start:
		s.outBuf.resetPacketStart()
	}

	s.outBuf.resetOffset()
	s.numFramesUsedSoFar = 0

	if s.noFramesLeft {
		// We're done:
		s.OnSourceClosure()
	} else {
		// We have more frames left to send.  Figure out when the next frame
		// is due to start playing, then make sure that we wait this long before
		// sending the next packet.
		var timeNow sys.Timeval
		sys.Gettimeofday(&timeNow)
		secsDiff := s.nextSendTime.Sec - timeNow.Sec
		uSecondsToGo := secsDiff*1000000 + (s.nextSendTime.Usec - timeNow.Usec)
		if uSecondsToGo < 0 || secsDiff < 0 { // sanity check: Make sure that the time-to-delay is non-negative:
			uSecondsToGo = 0
		}

		// Delay this amount of time:
		//log.Debug("[MultiFramedRTPSink::sendPacketIfNecessary] uSecondsToGo: %d", uSecondsToGo)
		time.Sleep(time.Duration(uSecondsToGo) * time.Microsecond)
		s.sendNext()
	}
}

func (s *MultiFramedRTPSink) sendNext() {
	s.buildAndSendPacket(false)
}

func (s *MultiFramedRTPSink) ourHandlerClosure() {
	s.noFramesLeft = true
	s.sendPacketIfNecessary()
}

func (s *MultiFramedRTPSink) isFirstFrameInPacket() bool {
	return s.numFramesUsedSoFar == 0
}

func (s *MultiFramedRTPSink) setTimestamp(framePresentationTime sys.Timeval) {
	// First, convert the presentation time to a 32-bit RTP timestamp:
	s.currentTimestamp = s.convertToRTPTimestamp(framePresentationTime)

	// Then, insert it into the RTP packet:
	s.outBuf.insertWord(s.currentTimestamp, s.timestampPosition)
}

// default implementation: Assume no special header:
func (s *MultiFramedRTPSink) SpecialHeaderSize() uint {
	return 0
}

// default implementation: Assume no frame-specific header:
func (s *MultiFramedRTPSink) frameSpecificHeaderSize() uint {
	return 0
}

func (s *MultiFramedRTPSink) allowOtherFramesAfterLastFragment() bool {
	return false
}

func (s *MultiFramedRTPSink) frameCanAppearAfterPacketStart(frameStart []byte, numBytesInFrame uint) bool {
	return true
}

func (s *MultiFramedRTPSink) computeOverflowForNewFrame(newFrameSize uint) uint {
	return s.outBuf.numOverflowBytes(newFrameSize)
}

func (s *MultiFramedRTPSink) setMarkerBit() {
	rtpHdr := s.outBuf.extractWord(0)
	rtpHdr |= 0x00800000
	s.outBuf.insertWord(rtpHdr, 0)
}

func (s *MultiFramedRTPSink) doSpecialFrameHandling(fragmentationOffset, numBytesInFrame, numRemainingBytes uint,
	frameStart []byte, framePresentationTime sys.Timeval) {
	if s.isFirstFrameInPacket() {
		s.setTimestamp(framePresentationTime)
	}
}
