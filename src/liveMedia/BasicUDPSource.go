package liveMedia

type BasicUDPSource struct {
    FramedSource
}

func NewBasicUDPSource() *BasicUDPSource {
	return new(BasicUDPSource)
}

func (this *BasicUDPSource) doGetNextFrame() {
}
