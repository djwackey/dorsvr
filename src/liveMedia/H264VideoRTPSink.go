package liveMedia

import (
	"fmt"
	. "groupsock"
)

//////// H264VideoRTPSink ////////
type H264VideoRTPSink struct {
	VideoRTPSink
	ourFragmenter *H264FUAFragmenter
	SPS           int
	PPS           int
}

func NewH264VideoRTPSink(rtpGroupSock *GroupSock, rtpPayloadType uint) *H264VideoRTPSink {
	h264VideoRTPSink := new(H264VideoRTPSink)
	h264VideoRTPSink.InitVideoRTPSink(h264VideoRTPSink, rtpGroupSock, rtpPayloadType, 90000, "H264")
	return h264VideoRTPSink
}

func (this *H264VideoRTPSink) continuePlaying() {
	fmt.Println("H264VideoRTPSink::continuePlaying")
	if this.ourFragmenter == nil {
		this.ourFragmenter = NewH264FUAFragmenter(this.source, OutPacketBufferMaxSize)
	} else {
		this.ourFragmenter.reAssignInputSource(this.source)
	}

	this.source = this.ourFragmenter

	this.multiFramedPlaying()
}

//////// H264FUAFragmenter ////////
type H264FUAFragmenter struct {
	FramedFilter
	maxOutputPacketSize uint
	numValidDataBytes   uint
	inputBufferSize     uint
	inputBuffer         []byte
}

func NewH264FUAFragmenter(inputSource IFramedSource, inputBufferMax uint) *H264FUAFragmenter {
	fragment := new(H264FUAFragmenter)
	fragment.numValidDataBytes = 1
	fragment.inputBufferSize = inputBufferMax + 1
	fragment.inputBuffer = make([]byte, fragment.inputBufferSize)
	fragment.InitFramedFilter(inputSource)
	return fragment
}

func (this *H264FUAFragmenter) doGetNextFrame() {
	if this.numValidDataBytes == 1 {
		this.inputSource.getNextFrame(this.buffTo, this.maxSize, nil)
	} else {
		if this.maxSize < this.maxOutputPacketSize {
		} else {
			this.maxSize = this.maxOutputPacketSize
		}
	}
}

func (this *H264FUAFragmenter) getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}) {
}

func (this *H264FUAFragmenter) afterGettingFrame(frameSize uint) {
	this.numValidDataBytes += frameSize
}
