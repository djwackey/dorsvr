package liveMedia

type RTPSink struct {
	MediaSink
}

func (this *RTPSink) AuxSDPLine() string {
	return ""
}

func (this *RTPSink) RtpPayloadType() string {
	return ""
}

func (this *RTPSink) SdpMediaType() string {
	return ""
}
