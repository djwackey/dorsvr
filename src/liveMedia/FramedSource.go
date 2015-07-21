package liveMedia

import (
//"fmt"
)

type IFramedSource interface {
	getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{})
}

type FramedSource struct {
	source  IFramedSource
	buffTo  []byte
	maxSize uint
}

func (this *FramedSource) InitFramedSource(source IFramedSource) {
	this.source = source
}
