package liveMedia

type RTPSink struct {
	MediaSink
    rtpPayloadType int
    rtpInterface RTPInterface
}

func (this *RTPSink) InitRTPSink(gs *GroupSock, rtpPayloadType int) {
    this.rtpInterface = NewRTPInterface(gs)
    this.rtpPayloadType = rtpPayloadType
}

func (this *RTPSink) AuxSDPLine() string {
	return ""
}

func (this *RTPSink) RtpPayloadType() string {
	return ""
}

func (this *RTPSink) SdpMediaType() string {
	return "data"
}
