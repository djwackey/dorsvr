package liveMedia

type RTPSource struct {
	FramedSource
	rtpInterface *RTCPInstance
}

func NewRTPSource() *RTPSource {
	return new(RTPSource)
}

func (source *RTPSource) setStreamSocket() {
}
