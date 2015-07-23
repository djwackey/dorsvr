package liveMedia

import (
	. "groupsock"
)

//////// H264VideoRTPSink ////////
type H264VideoRTPSink struct {
	VideoRTPSink
    ourFragmenter *H264FUAFragmenter
	SPS int
	PPS int
}

func NewH264VideoRTPSink(rtpGroupSock *GroupSock, rtpPayloadType uint) *H264VideoRTPSink {
	h264VideoRTPSink := new(H264VideoRTPSink)
	h264VideoRTPSink.InitVideoRTPSink(h264VideoRTPSink, rtpGroupSock, rtpPayloadType, 90000, "H264")
	return h264VideoRTPSink
}

func (this *H264VideoRTPSink) continuePlaying() {
    if this.ourFragmenter == nil {
        this.ourFragmenter = NewH264FUAFragmenter(this.source)
    } else {
        this.ourFragmenter.reAssignInputSource(this.source)
    }

    this.source = this.ourFragmenter
}


//////// H264FUAFragmenter ////////
type H264FUAFragmenter struct {
    FramedFilter
    numValidDataBytes uint
    inputBufferSize uint
    inputBuffer []byte
}

func NewH264FUAFragmenter(inputSource IFramedSource, inputBufferMax uint) *H264FUAFragmenter {
    fragment := new(H264FUAFragmenter)
    fragment.numValidDataBytes = 1
    fragment.inputBufferSize = inputBufferMax + 1
    fragment.inputBuffer = make([]byte, this.inputBufferSize)
    fragment.InitFramedFilter(inputSource)
    return fragment
}

func (this *H264FUAFragmenter) doGetNextFrame() {
    if this.numValidDataBytes == 1 {
        this.inputSource.getNextFrame()
    } else {
        if this.maxSize < this.maxOutputPacketSize {
        } else {
            this.maxSize = this.maxOutputPacketSize
        }
    }
}

func (this *H264FUAFragmenter) afterGettingFrame(frameSize uint) {
    this.numValidDataBytes += frameSize
}
