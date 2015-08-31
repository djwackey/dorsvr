package liveMedia

import (
	"fmt"
	. "include"
)

type IFramedSource interface {
	getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}, onCloseFunc interface{})
	doGetNextFrame()
	afterGetting()
	maxFrameSize() uint
	//getSPSandPPS()
	//stopGettingFrames()
}

type FramedSource struct {
	afterGettingFunc        interface{}
	onCloseFunc             interface{}
	source                  IFramedSource
	buffTo                  []byte
	maxSize                 uint
	frameSize               uint
	numTruncatedBytes       uint
	durationInMicroseconds  uint
	isCurrentlyAwaitingData bool
	presentationTime        Timeval
}

func (this *FramedSource) InitFramedSource(source IFramedSource) {
	this.source = source
}

func (this *FramedSource) getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}, onCloseFunc interface{}) {
	if this.isCurrentlyAwaitingData {
		panic("FramedSource::getNextFrame(): attempting to read more than once at the same time!")
	}

	fmt.Println(fmt.Sprintf("FramedSource::getNextFrame -> %p", this.source))

	this.buffTo = buffTo
	this.maxSize = maxSize
	this.onCloseFunc = onCloseFunc
	this.afterGettingFunc = afterGettingFunc
	this.isCurrentlyAwaitingData = true

	this.source.doGetNextFrame()
}

func (this *FramedSource) afterGetting() {
	this.isCurrentlyAwaitingData = false

	if this.afterGettingFunc != nil {
	}
}

func (this *FramedSource) handleClosure() {
	this.isCurrentlyAwaitingData = false

	if this.onCloseFunc != nil {
	}
}

func (this *FramedSource) stopGettingFrames() {
	this.isCurrentlyAwaitingData = false
}

func (this *FramedSource) maxFrameSize() uint {
	return 0
}
