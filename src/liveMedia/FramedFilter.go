package liveMedia

type FramedFilter struct {
	FramedSource
	inputSource IFramedSource
}

func (this *FramedFilter) InitFramedFilter(inputSource IFramedSource) {
	this.inputSource = inputSource
}

func (this *FramedFilter) InputSource() IFramedSource {
	return this.inputSource
}

func (this *FramedFilter) reAssignInputSource(newInputSource IFramedSource) {
	this.inputSource = newInputSource
}
