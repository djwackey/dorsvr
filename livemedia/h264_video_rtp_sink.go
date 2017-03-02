package livemedia

import (
	"encoding/base64"
	"fmt"
	sys "syscall"

	gs "github.com/djwackey/dorsvr/groupsock"
)

//////// H264VideoRTPSink ////////
type H264VideoRTPSink struct {
	VideoRTPSink
	ourFragmenter *H264FUAFragmenter
	sps           string
	pps           string
	spsSize       int
	ppsSize       int
}

func newH264VideoRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint) *H264VideoRTPSink {
	sink := new(H264VideoRTPSink)
	sink.initVideoRTPSink(sink, rtpGroupSock, rtpPayloadType, 90000, "H264")
	return sink
}

func (s *H264VideoRTPSink) ContinuePlaying() {
	if s.ourFragmenter == nil {
		s.ourFragmenter = newH264FUAFragmenter(s.Source, OutPacketBufferMaxSize)
	} else {
		s.ourFragmenter.reAssignInputSource(s.Source)
	}

	s.Source = s.ourFragmenter
	s.multiFramedPlaying()
}

func (s *H264VideoRTPSink) AuxSDPLine() string {
	sps := s.sps
	pps := s.pps
	//spsSize := s.spsSize
	//ppsSize := s.ppsSize
	if sps == "" || pps == "" {
		if s.ourFragmenter == nil {
			return ""
		}

		framerSource := s.ourFragmenter.InputSource()
		if framerSource == nil {
			return ""
		}

		//framerSource.getSPSandPPS()
	}

	spsBase64 := base64.StdEncoding.EncodeToString([]byte(sps))
	ppsBase64 := base64.StdEncoding.EncodeToString([]byte(pps))

	var profileLevelID uint8
	if s.spsSize >= 4 {
		profileLevelID = (sps[1] << 16) | (sps[2] << 8) | sps[3] // profile_idc|constraint_setN_flag|level_idc
	}

	fmt.Sprintf("a=fmtp:%d packetization-mode=1;profile-level-id=%06X;sprop-parameter-sets=%s,%s\r\n",
		s.RtpPayloadType(), profileLevelID, spsBase64, ppsBase64)
	return ""
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

func newH264FUAFragmenter(inputSource IFramedSource, inputBufferMax uint) *H264FUAFragmenter {
	fragment := new(H264FUAFragmenter)
	fragment.numValidDataBytes = 1
	fragment.inputBufferSize = inputBufferMax + 1
	fragment.inputBuffer = make([]byte, fragment.inputBufferSize)
	fragment.InitFramedFilter(inputSource)
	fragment.InitFramedSource(fragment)
	return fragment
}

func (f *H264FUAFragmenter) doGetNextFrame() {
	if f.numValidDataBytes == 1 {
		// H264VideoStreamFramer
		// We have no NAL unit data currently in the buffer.  Read a new one:
		f.inputSource.GetNextFrame(f.inputBuffer[1:], f.inputBufferSize-1,
			f.afterGettingFrame, f.handleClosure)
	} else {
		if f.maxSize < f.maxOutputPacketSize {
			fmt.Printf("H264FUAFragmenter::doGetNextFrame(): maxSize (%d) is smaller than expected\n", f.maxSize)
		} else {
			f.maxSize = f.maxOutputPacketSize
		}

		f.lastFragmentCompletedNALUnit = true
		if f.curDataOffset == 1 {
			if f.numValidDataBytes-1 <= f.maxSize { // case 1
				copy(f.buffTo, f.inputBuffer[1:f.numValidDataBytes-1])
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
			copy(f.buffTo, f.inputBuffer[f.curDataOffset-2:numBytesToSend])
			f.frameSize = numBytesToSend
			f.curDataOffset += numBytesToSend - 2
		}

		if f.curDataOffset >= f.numValidDataBytes {
			// We're done with this data.  Reset the pointers for receiving new data:
			f.numValidDataBytes = 1
			f.curDataOffset = 1
		}

		// Complete delivery to the client:
		f.inputSource.afterGetting()
	}
}

func (f *H264FUAFragmenter) afterGettingFrame(frameSize, numTruncatedBytes uint, presentationTime sys.Timeval) {
	f.numValidDataBytes += frameSize

	f.doGetNextFrame()
}
