package liveMedia

type RTPSource struct {
	FramedSource
}

func NewRTPSource() *RTPSource {
	return new(RTPSource)
}
