package livemedia

type FramedFilter struct {
	FramedSource
	inputSource IFramedSource
}

func (f *FramedFilter) initFramedFilter(inputSource IFramedSource) {
	f.inputSource = inputSource
}
