package livemedia

import s "syscall"

type IFramedSource interface {
	GetNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}, onCloseFunc interface{})
	doGetNextFrame()
	afterGetting()
	maxFrameSize() uint
	handleClosure()
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
	presentationTime        s.Timeval
}

func (f *FramedSource) InitFramedSource(source IFramedSource) {
	f.source = source
}

func (f *FramedSource) GetNextFrame(buffTo []byte, maxSize uint,
	afterGettingFunc interface{}, onCloseFunc interface{}) {
	if f.isCurrentlyAwaitingData {
		panic("FramedSource::GetNextFrame(): attempting to read more than once at the same time!")
	}

	f.buffTo = buffTo
	f.maxSize = maxSize
	f.onCloseFunc = onCloseFunc
	f.afterGettingFunc = afterGettingFunc
	f.isCurrentlyAwaitingData = true

	f.source.doGetNextFrame()
}

func (f *FramedSource) afterGetting() {
	f.isCurrentlyAwaitingData = false

	if f.afterGettingFunc != nil {
		f.afterGettingFunc.(func(frameSize, durationInMicroseconds uint,
			presentationTime s.Timeval))(f.frameSize,
			f.durationInMicroseconds, f.presentationTime)
	}
}

func (f *FramedSource) handleClosure() {
	f.isCurrentlyAwaitingData = false

	if f.onCloseFunc != nil {
		f.onCloseFunc.(func())()
	}
}

func (f *FramedSource) stopGettingFrames() {
	f.isCurrentlyAwaitingData = false
}

func (f *FramedSource) maxFrameSize() uint {
	return 0
}
