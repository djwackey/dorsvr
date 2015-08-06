package liveMedia

import (
//"fmt"
    . "include"
)

type IFramedSource interface {
	getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{})
	afterGetting()
	//stopGettingFrames()
}

type FramedSource struct {
	afterGettingFunc       interface{}
	onCloseFunc            interface{}
	source                 IFramedSource
	buffTo                 []byte
	maxSize                uint
    frameSize              uint
    numTruncatedBytes      uint
	durationInMicroseconds uint
    presentationTime       Timeval
}

func (this *FramedSource) InitFramedSource(source IFramedSource) {
	this.source = source
}

func (this *FramedSource) afterGetting() {
	if this.afterGettingFunc != nil {
	}
}

func (this *FramedSource) stopGettingFrames() {
}
