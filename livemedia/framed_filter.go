package livemedia

type FramedFilter struct {
	FramedSource
	inputSource IFramedSource
}

func (f *FramedFilter) InitFramedFilter(inputSource IFramedSource) {
	f.inputSource = inputSource
}

func (f *FramedFilter) InputSource() IFramedSource {
	return f.inputSource
}

func (f *FramedFilter) reAssignInputSource(newInputSource IFramedSource) {
	f.inputSource = newInputSource
}

func (f *FramedFilter) detachInputSource() {
	f.reAssignInputSource(nil)
}
