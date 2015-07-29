package liveMedia

import (
//"fmt"
)

type IFramedSource interface {
	getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{})
    afterGetting()
	//stopGettingFrames()
}

type FramedSource struct {
	afterGettingFunc interface{}
	onCloseFunc      interface{}
	source           IFramedSource
	buffTo           []byte
	maxSize          uint
    durationInMicroseconds uint
}

func (this *FramedSource) InitFramedSource(source IFramedSource) {
	this.source = source
}

func (this *FramedSource) afterGetting() {
}

func (this *FramedSource) stopGettingFrames() {
}
