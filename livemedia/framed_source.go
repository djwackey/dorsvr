package livemedia

import sys "syscall"

type IFramedSource interface {
	GetNextFrame(buffTo []byte, maxSize uint, afterGettingFunc, onCloseFunc interface{}) error
	doGetNextFrame() error
	isAwaitingData() bool
	maxFrameSize() uint
	afterGetting()
	handleClosure()
	stopGettingFrames()
	doStopGettingFrames() error
	destroy()
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
	presentationTime        sys.Timeval
}

func (f *FramedSource) initFramedSource(source IFramedSource) {
	f.source = source
}

func (f *FramedSource) GetNextFrame(buffTo []byte, maxSize uint,
	afterGettingFunc, onCloseFunc interface{}) error {
	if f.isCurrentlyAwaitingData {
		panic("FramedSource::GetNextFrame(): attempting to read more than once at the same time!")
	}

	f.buffTo = buffTo
	f.maxSize = maxSize
	f.onCloseFunc = onCloseFunc
	f.afterGettingFunc = afterGettingFunc
	f.isCurrentlyAwaitingData = true

	return f.source.doGetNextFrame()
}

func (f *FramedSource) afterGetting() {
	f.isCurrentlyAwaitingData = false

	if f.afterGettingFunc != nil {
		f.afterGettingFunc.(func(frameSize, durationInMicroseconds uint,
			presentationTime sys.Timeval))(f.frameSize,
			f.durationInMicroseconds, f.presentationTime)
	}
}

func (f *FramedSource) handleClosure() {
	f.stopGettingFrames()
	//f.isCurrentlyAwaitingData = false

	if f.onCloseFunc != nil {
		f.onCloseFunc.(func())()
	}
}

func (f *FramedSource) stopGettingFrames() {
	f.isCurrentlyAwaitingData = false

	// perform any specialized action
	f.source.doStopGettingFrames()
}

// default implementation: do nothing
func (f *FramedSource) doStopGettingFrames() error {
	return nil
}

// default implementation: do nothing
func (f *FramedSource) maxFrameSize() uint {
	return 0
}

func (f *FramedSource) isAwaitingData() bool {
	return f.isCurrentlyAwaitingData
}

// default implementation: do nothing
func (f *FramedSource) destroy() {
}
