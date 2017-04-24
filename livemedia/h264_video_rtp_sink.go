package livemedia

import (
	"encoding/base64"
	"fmt"
	sys "syscall"

	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/gitea/log"
)

//////// H264VideoRTPSink ////////
type H264VideoRTPSink struct {
	VideoRTPSink
	ourFragmenter *H264FUAFragmenter
	sps           []byte
	pps           []byte
	spsSize       uint
	ppsSize       uint
}

func newH264VideoRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint32) *H264VideoRTPSink {
	sink := new(H264VideoRTPSink)
	sink.initVideoRTPSink(sink, rtpGroupSock, rtpPayloadType, 90000, "H264")
	return sink
}

func (s *H264VideoRTPSink) destroy() {
	s.StopPlaying()
}

func (s *H264VideoRTPSink) ContinuePlaying() {
	if s.ourFragmenter == nil {
		s.ourFragmenter = newH264FUAFragmenter(s.Source, OutPacketBufferMaxSize, s.ourMaxPacketSize-12)
	} else {
		// reassign input source
		s.ourFragmenter.initFramedFilter(s.Source)
	}

	s.Source = s.ourFragmenter
	s.multiFramedPlaying()
}

func (s *H264VideoRTPSink) AuxSDPLine() string {
	if len(s.sps) == 0 || len(s.pps) == 0 {
		if s.ourFragmenter == nil {
			return ""
		}

		framerSource := s.ourFragmenter.inputSource.(*H264VideoStreamFramer)
		if framerSource == nil {
			return ""
		}

		s.sps, s.pps, s.spsSize, s.ppsSize = framerSource.getSPSandPPS()
		if len(s.sps) == 0 || len(s.pps) == 0 {
			return ""
		}
	}

	spsBase64 := base64.StdEncoding.EncodeToString(s.sps)
	ppsBase64 := base64.StdEncoding.EncodeToString(s.pps)

	var profileLevelID uint32
	if s.spsSize >= 4 {
		profileLevelID = Uint32(s.sps[1:4])
	}

	return fmt.Sprintf("a=fmtp:%d packetization-mode=1;profile-level-id=%06X;sprop-parameter-sets=%s,%s\r\n",
		s._rtpPayloadType, profileLevelID, spsBase64, ppsBase64)
}

func (s *H264VideoRTPSink) doSpecialFrameHandling(fragmentationOffset, numBytesInFrame, numRemainingBytes uint,
	frameStart []byte, framePresentationTime sys.Timeval) {
	if s.ourFragmenter != nil {
		framerSource := s.ourFragmenter.inputSource.(*H264VideoStreamFramer)
		if s.ourFragmenter.lastFragmentCompletedNALUnit && framerSource != nil && framerSource.pictureEndMarker {
			s.setMarkerBit()
			framerSource.pictureEndMarker = false
		}
	}
	s.setTimestamp(framePresentationTime)
}

func (s *H264VideoRTPSink) frameCanAppearAfterPacketStart(frameStart []byte, numBytesInFrame uint) bool {
	return false
}

//////// H264FUAFragmenter ////////
type H264FUAFragmenter struct {
	FramedFilter
	saveNumTruncatedBytes        uint
	maxOutputPacketSize          uint
	numValidDataBytes            uint
	inputBufferSize              uint
	curDataOffset                uint
	inputBuffer                  []byte
	lastFragmentCompletedNALUnit bool
}

func newH264FUAFragmenter(inputSource IFramedSource,
	inputBufferMax, maxOutputPacketSize uint) *H264FUAFragmenter {
	fragment := new(H264FUAFragmenter)
	fragment.curDataOffset = 1
	fragment.numValidDataBytes = 1
	fragment.inputBufferSize = inputBufferMax + 1
	fragment.maxOutputPacketSize = maxOutputPacketSize
	fragment.inputBuffer = make([]byte, fragment.inputBufferSize)
	fragment.initFramedFilter(inputSource)
	fragment.initFramedSource(fragment)
	return fragment
}

func (f *H264FUAFragmenter) doGetNextFrame() error {
	if f.numValidDataBytes == 1 {
		// We have no NAL unit data currently in the buffer.  Read a new one:
		return f.inputSource.GetNextFrame(f.inputBuffer[1:], f.inputBufferSize-1,
			f.afterGettingFrame, f.handleClosure) // H264VideoStreamFramer
	} else {
		if f.maxSize < f.maxOutputPacketSize {
			log.Warn("H264FUAFragmenter::doGetNextFrame(): maxSize (%d) is smaller than expected\n", f.maxSize)
		} else {
			f.maxSize = f.maxOutputPacketSize
		}

		f.lastFragmentCompletedNALUnit = true
		if f.curDataOffset == 1 {
			if f.numValidDataBytes-1 <= f.maxSize { // case 1
				copy(f.buffTo, f.inputBuffer[1:f.numValidDataBytes])
				f.frameSize = f.numValidDataBytes - 1
				f.curDataOffset = f.numValidDataBytes
			} else { // case 2
				// We need to send the NAL unit data as FU-A packets.  Deliver the first
				// packet now.  Note that we add FU indicator and FU header bytes to the front
				// of the packet (reusing the existing NAL header byte for the FU header).
				f.inputBuffer[0] = (f.inputBuffer[1] & 0xE0) | 28   // FU indicator
				f.inputBuffer[1] = 0x80 | (f.inputBuffer[1] & 0x1F) // FU header (with S bit)
				copy(f.buffTo, f.inputBuffer[:f.maxSize])
				f.frameSize = f.maxSize
				f.curDataOffset += f.maxSize - 1
				f.lastFragmentCompletedNALUnit = false
			}
		} else {
			f.inputBuffer[f.curDataOffset-2] = f.inputBuffer[0]         // FU indicator
			f.inputBuffer[f.curDataOffset-1] = f.inputBuffer[1] &^ 0x80 // FU header (no S bit)
			numBytesToSend := 2 + f.numValidDataBytes - f.curDataOffset
			if numBytesToSend > f.maxSize {
				// We can't send all of the remaining data this time:
				numBytesToSend = f.maxSize
				f.lastFragmentCompletedNALUnit = false
			} else {
				// This is the last fragment:
				f.inputBuffer[f.curDataOffset-1] |= 0x40 // set the E bit in the FU header
				f.numTruncatedBytes = f.saveNumTruncatedBytes
			}
			copy(f.buffTo, f.inputBuffer[f.curDataOffset-2:f.curDataOffset-2+numBytesToSend])
			f.frameSize = numBytesToSend
			f.curDataOffset += numBytesToSend - 2
		}

		if f.curDataOffset >= f.numValidDataBytes {
			// We're done with this data.  Reset the pointers for receiving new data:
			f.numValidDataBytes = 1
			f.curDataOffset = 1
		}

		// Complete delivery to the client:
		f.afterGetting()
	}
	return nil
}

func (f *H264FUAFragmenter) afterGettingFrame(frameSize, durationInMicroseconds uint, presentationTime sys.Timeval) {
	f.numValidDataBytes += frameSize
	f.presentationTime = presentationTime
	f.durationInMicroseconds = durationInMicroseconds

	f.doGetNextFrame()
}
