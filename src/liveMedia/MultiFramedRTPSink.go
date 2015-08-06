package liveMedia

import (
	"fmt"
	. "groupsock"
	. "include"
	//"time"
)

var rtpHeaderSize int = 12

type MultiFramedRTPSink struct {
	RTPSink
	outBuf                *OutPacketBuffer
	nextSendTime          Timeval
    noFramesLeft          bool
	isFirstPacket         bool
	ourMaxPacketSize      uint
	timestampPosition     uint
	specialHeaderSize     uint
    numFramesUsedSoFar    uint
	specialHeaderPosition uint
    totalFrameSpecificHeaderSizes uint
	onSendErrorFunc       interface{}
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
		this.afterGettingFrame()
	} else {
		// Normal case: we need to read a new frame from the source
		if this.source == nil {
			return
		}
		fmt.Println("packFrame", this.afterGettingFrame)
		this.source.getNextFrame(this.outBuf.curPtr(), this.outBuf.totalBytesAvailable(), this.afterGettingFrame)
	}
}

func (this *MultiFramedRTPSink) afterGettingFrame(frameSize uint) {
	fmt.Println("MultiFramedRTPSink::afterGettingFrame")
	if this.isFirstPacket {
		// Record the fact that we're starting to play now:
		GetTimeOfDay(&this.nextSendTime)
	}

    numFrameBytesToUse := frameSize

	   if numFrameBytesToUse == 0 && frameSize > 0 {
	       // Send our packet now, because we have filled it up:
	       this.sendPacketIfNecessary()
	   } else {
	       // Use this frame in our outgoing packet:
	       frameStart = this.outBuf.curPtr()
	       this.outBuf.increment(numFrameBytesToUse)
	       // do this now, in case "doSpecialFrameHandling()" calls "setFramePadding()" to append padding bytes

	       // Here's where any payload format specific processing gets done:
	       this.doSpecialFrameHandling(curFragmentationOffset, numFrameBytesToUse, overflowBytes, frameStart, presentationTime)

	       this.numFramesUsedSoFar++

	       // Update the time at which the next packet should be sent, based
	       // on the duration of the frame that we just packed into it.
	       // However, if this frame has overflow data remaining, then don't
	       // count its duration yet.
	       if overflowBytes == 0 {
	           this.nextSendTime.tv_usec += this.durationInMicroseconds
	           this.nextSendTime.tv_sec += this.nextSendTime.tv_usec/1000000
	           this.nextSendTime.tv_usec %= 1000000
	       }

	       // Send our packet now if (i) it's already at our preferred size, or
	       // (ii) (heuristic) another frame of the same size as the one we just
	       //      read would overflow the packet, or
	       // (iii) it contains the last fragment of a fragmented frame, and we
	       //      don't allow anything else to follow this or
	       // (iv) one frame per packet is allowed:
	       //if this.outBuf.isPreferredSize() ||
	          //this.outBuf.wouldOverflow(numFrameBytesToUse) ||
	          //this.previousFrameEndedFragmentation && !allowOtherFramesAfterLastFragment() || !frameCanAppearAfterPacketStart(fOutBuf->curPtr() - frameSize, frameSize) {
	           // The packet is ready to be sent now
	           //this.sendPacketIfNecessary()
	       //} else {
	           // There's room for more frames; try getting another:
	           //this.packFrame()
	       //}
	   }
}

func (this *MultiFramedRTPSink) sendPacketIfNecessary() {
	//fmt.Println("sendPacketIfNecessary", this.outBuf.packet(), this.outBuf.curPacketSize())
	    if this.numFramesUsedSoFar > 0 {
	        if !this.rtpInterface.sendPacket(this.outBuf.packet(), this.outBuf.curPacketSize()) {
			    // if failure handler has been specified, call it
	            if this.onSendErrorFunc != nil {}
	        }

	        this.packetCount++
	        this.totalOctetCount += this.outBuf.curPacketSize()
	        this.octetCount += this.outBuf.curPacketSize() - rtpHeaderSize - this.specialHeaderSize - this.totalFrameSpecificHeaderSizes

	        this.seqNo++ // for next time
	    }

	    if this.outBuf.haveOverflowData() &&
	       this.outBuf.totalBytesAvailable() > this.outBuf.totalBufferSize()/2 {
	       // Efficiency hack: Reset the packet start pointer to just in front of
	       // the overflow data (allowing for the RTP header and special headers),
	       // so that we probably don't have to "memmove()" the overflow data
	       // into place when building the next packet:
	       newPacketStart = this.outBuf.curPacketSize() - (rtpHeaderSize + this.specialHeaderSize + this.frameSpecificHeaderSize())
	       this.outBuf.adjustPacketStart(newPacketStart)
	   } else {
	       // Normal case: Reset the packet start pointer back to the start:
	       this.outBuf.resetPacketStart()
	   }

	   this.outBuf.resetOffset()
	   this.numFramesUsedSoFar = 0

	   if this.noFramesLeft {
	       // We're done:
	       this.onSourceClosure()
	   } else {
	       // We have more frames left to send.  Figure out when the next frame
	       // is due to start playing, then make sure that we wait this long before
	       // sending the next packet.
	       var timeNow Timeval
	       GetTimeOfDay(&timeNow)
	       secsDiff := this.nextSendTime.tv_sec - timeNow.tv_sec
	       uSecondsToGo = secsDiff*1000000 + (this.nextSendTime.tv_usec - timeNow.tv_usec)
	       if uSecondsToGo < 0 || secsDiff < 0 { // sanity check: Make sure that the time-to-delay is non-negative:
	           uSecondsToGo = 0
	       }

	       // Delay this amount of time:
	       sendNext()
	   }
}

func (this *MultiFramedRTPSink) sendNext() {
	this.buildAndSendPacket(false)
}

func (this *MultiFramedRTPSink) SpecialHeaderSize() uint {
	// default implementation: Assume no special header:
	return 0
}

func (this *MultiFramedRTPSink) frameSpecificHeaderSize() uint {
    // default implementation: Assume no frame-specific header:
    return 0
}

func (this *MultiFramedRTPSink) doSpecialFrameHandling(fragmentationOffset, numBytesInFrame, numRemainingBytes uint, frameStart string, framePresentationTime Timeval) {
}
