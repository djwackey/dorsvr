package liveMedia

type BasicUDPSource struct {
	FramedSource
}

func NewBasicUDPSource() *BasicUDPSource {
	return new(BasicUDPSource)
}

func (this *BasicUDPSource) getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}, onCloseFunc interface{}) {
}

func (this *BasicUDPSource) doGetNextFrame() {
}
