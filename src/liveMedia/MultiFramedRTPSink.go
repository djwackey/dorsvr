package liveMedia

import (
	"fmt"
	. "groupsock"
	//"time"
	"utils"
)

var rtpHeaderSize uint = 12

type MultiFramedRTPSink struct {
	RTPSink
	outBuf                          *OutPacketBuffer
	nextSendTime                    utils.Timeval
	noFramesLeft                    bool
	isFirstPacket                   bool
	currentTimestamp                uint
	ourMaxPacketSize                uint
	timestampPosition               uint
	specialHeaderSize               uint
	numFramesUsedSoFar              uint
	specialHeaderPosition           uint
	curFragmentationOffset          uint
	totalFrameSpecificHeaderSizes   uint
	previousFrameEndedFragmentation bool
	onSendErrorFunc                 interface{}
}

func (this *MultiFramedRTPSink) InitMultiFramedRTPSink(rtpSink IRTPSink,
	rtpGroupSock *GroupSock, rtpPayloadType, rtpTimestampFrequency uint, rtpPayloadFormatName string) {
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
	fmt.Println("MultiFramedRTPSink::ContinuePlaying")
	this.buildAndSendPacket(true)
}

func (this *MultiFramedRTPSink) buildAndSendPacket(isFirstPacket bool) {
	this.isFirstPacket = isFirstPacket

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

	// Begin packing as many (complete) frames into the packet as we can:
	this.noFramesLeft = false
	this.numFramesUsedSoFar = 0
	this.totalFrameSpecificHeaderSizes = 0

	this.packFrame()
}

func (this *MultiFramedRTPSink) packFrame() {
	if this.outBuf.haveOverflowData() {
		// Use this frame before reading a new one from the source
		frameSize := this.outBuf.OverflowDataSize()
		presentationTime := this.outBuf.OverflowPresentationTime()
		durationInMicroseconds := this.outBuf.OverflowDurationInMicroseconds()
		this.outBuf.useOverflowData()
		this.afterGettingFrame(frameSize, durationInMicroseconds, presentationTime)
	} else {
		// Normal case: we need to read a new frame from the source
		if this.Source == nil {
			return
		}
		fmt.Println("packFrame", this.afterGettingFrame)
		this.Source.GetNextFrame(this.outBuf.curPtr(), this.outBuf.totalBytesAvailable(),
			this.afterGettingFrame, this.ourHandlerClosure)
	}
}

func (this *MultiFramedRTPSink) afterGettingFrame(frameSize, durationInMicroseconds uint, presentationTime utils.Timeval) {
	fmt.Println("MultiFramedRTPSink::afterGettingFrame")
	if this.isFirstPacket {
		// Record the fact that we're starting to play now:
		utils.GetTimeOfDay(&this.nextSendTime)
	}

	curFragmentationOffset := this.curFragmentationOffset
	numFrameBytesToUse := frameSize
	var overflowBytes uint

	if numFrameBytesToUse == 0 && frameSize > 0 {
		// Send our packet now, because we have filled it up:
		this.sendPacketIfNecessary()
	} else {
		// Use this frame in our outgoing packet:
		frameStart := this.outBuf.curPtr()
		this.outBuf.increment(numFrameBytesToUse)
		// do this now, in case "doSpecialFrameHandling()" calls "setFramePadding()" to append padding bytes

		// Here's where any payload format specific processing gets done:
		this.doSpecialFrameHandling(curFragmentationOffset, numFrameBytesToUse, overflowBytes, string(frameStart), presentationTime)

		this.numFramesUsedSoFar++

		// Update the time at which the next packet should be sent, based
		// on the duration of the frame that we just packed into it.
		// However, if this frame has overflow data remaining, then don't
		// count its duration yet.
		if overflowBytes == 0 {
			this.nextSendTime.Tv_usec += int64(durationInMicroseconds)
			this.nextSendTime.Tv_sec += this.nextSendTime.Tv_usec / 1000000
			this.nextSendTime.Tv_usec %= 1000000
		}

		// Send our packet now if (i) it's already at our preferred size, or
		// (ii) (heuristic) another frame of the same size as the one we just
		//      read would overflow the packet, or
		// (iii) it contains the last fragment of a fragmented frame, and we
		//      don't allow anything else to follow this or
		// (iv) one frame per packet is allowed:
		if this.outBuf.isPreferredSize() ||
			this.outBuf.wouldOverflow(numFrameBytesToUse) ||
			this.previousFrameEndedFragmentation && !this.allowOtherFramesAfterLastFragment() ||
			!this.frameCanAppearAfterPacketStart(this.outBuf.curPtr(), frameSize) {
			// The packet is ready to be sent now
			this.sendPacketIfNecessary()
		} else {
			// There's room for more frames; try getting another:
			this.packFrame()
		}
	}
}

func (this *MultiFramedRTPSink) sendPacketIfNecessary() {
	//fmt.Println("sendPacketIfNecessary", this.outBuf.packet(), this.outBuf.curPacketSize())
	if this.numFramesUsedSoFar > 0 {
		if !this.rtpInterface.sendPacket(this.outBuf.packet(), this.outBuf.curPacketSize()) {
			// if failure handler has been specified, call it
			if this.onSendErrorFunc != nil {
			}
		}

		this.packetCount++
		this.totalOctetCount += this.outBuf.curPacketSize()
		this.octetCount += this.outBuf.curPacketSize() - uint(rtpHeaderSize) - this.specialHeaderSize - this.totalFrameSpecificHeaderSizes

		this.seqNo++ // for next time
	}

	if this.outBuf.haveOverflowData() &&
		this.outBuf.totalBytesAvailable() > this.outBuf.totalBufferSize()/2 {
		// Efficiency hack: Reset the packet start pointer to just in front of
		// the overflow data (allowing for the RTP header and special headers),
		// so that we probably don't have to "memmove()" the overflow data
		// into place when building the next packet:
		newPacketStart := this.outBuf.curPacketSize() - (rtpHeaderSize + this.specialHeaderSize + this.frameSpecificHeaderSize())
		this.outBuf.adjustPacketStart(newPacketStart)
	} else {
		// Normal case: Reset the packet start pointer back to the start:
		this.outBuf.resetPacketStart()
	}

	this.outBuf.resetOffset()
	this.numFramesUsedSoFar = 0

	if this.noFramesLeft {
		// We're done:
		this.OnSourceClosure()
	} else {
		// We have more frames left to send.  Figure out when the next frame
		// is due to start playing, then make sure that we wait this long before
		// sending the next packet.
		var timeNow utils.Timeval
		utils.GetTimeOfDay(&timeNow)
		secsDiff := this.nextSendTime.Tv_sec - timeNow.Tv_sec
		uSecondsToGo := secsDiff*1000000 + (this.nextSendTime.Tv_usec - timeNow.Tv_usec)
		if uSecondsToGo < 0 || secsDiff < 0 { // sanity check: Make sure that the time-to-delay is non-negative:
			uSecondsToGo = 0
		}

		// Delay this amount of time:
		this.sendNext()
	}
}

func (this *MultiFramedRTPSink) sendNext() {
	this.buildAndSendPacket(false)
}

func (this *MultiFramedRTPSink) ourHandlerClosure() {
	fmt.Println("MultiFramedRTPSink::ourHandlerClosure")
	this.noFramesLeft = true
}

func (this *MultiFramedRTPSink) isFirstFrameInPacket() bool {
	return this.numFramesUsedSoFar == 0
}

func (this *MultiFramedRTPSink) setTimestamp(framePresentationTime utils.Timeval) {
	// First, convert the presentation time to a 32-bit RTP timestamp:
	this.currentTimestamp = this.convertToRTPTimestamp(framePresentationTime)

	// Then, insert it into the RTP packet:
	this.outBuf.insertWord(byte(this.currentTimestamp), this.timestampPosition)
}

func (this *MultiFramedRTPSink) SpecialHeaderSize() uint {
	// default implementation: Assume no special header:
	return 0
}

func (this *MultiFramedRTPSink) frameSpecificHeaderSize() uint {
	// default implementation: Assume no frame-specific header:
	return 0
}

func (this *MultiFramedRTPSink) allowOtherFramesAfterLastFragment() bool {
	return false
}

func (this *MultiFramedRTPSink) frameCanAppearAfterPacketStart(frameStart []byte, numBytesInFrame uint) bool {
	return true
}

func (this *MultiFramedRTPSink) doSpecialFrameHandling(fragmentationOffset, numBytesInFrame, numRemainingBytes uint,
	frameStart string, framePresentationTime utils.Timeval) {
	if this.isFirstFrameInPacket() {
		this.setTimestamp(framePresentationTime)
	}
}
