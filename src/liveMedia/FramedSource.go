package liveMedia

import (
//"fmt"
)

type IFramedSource interface {
	getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{})
    stopGettingFrames()
}

type FramedSource struct {
    afterGettingFunc interface{}
    onCloseFunc interface{}
	source  IFramedSource
	buffTo  []byte
	maxSize uint
}

func (this *FramedSource) InitFramedSource(source IFramedSource) {
	this.source = source
}

func (this *FramedSource) afterGetting() {
}
