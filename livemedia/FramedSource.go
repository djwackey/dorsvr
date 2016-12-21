package livemedia

import "github.com/djwackey/dorsvr/utils"

type IFramedSource interface {
	GetNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}, onCloseFunc interface{})
	doGetNextFrame()
	afterGetting()
	maxFrameSize() uint
	stopGettingFrames()
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
	presentationTime        utils.Timeval
}

func (this *FramedSource) InitFramedSource(source IFramedSource) {
	this.source = source
}

func (this *FramedSource) GetNextFrame(buffTo []byte, maxSize uint,
	afterGettingFunc interface{}, onCloseFunc interface{}) {
	if this.isCurrentlyAwaitingData {
		panic("FramedSource::GetNextFrame(): attempting to read more than once at the same time!")
	}

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
		this.afterGettingFunc.(func(frameSize, durationInMicroseconds uint,
			presentationTime utils.Timeval))(this.frameSize, this.durationInMicroseconds, this.presentationTime)
	}
}

func (this *FramedSource) handleClosure() {
	this.isCurrentlyAwaitingData = false

	if this.onCloseFunc != nil {
		this.onCloseFunc.(func())()
	}
}

func (this *FramedSource) stopGettingFrames() {
	this.isCurrentlyAwaitingData = false
}

func (this *FramedSource) maxFrameSize() uint {
	return 0
}
