package liveMedia

type MPEGVideoStreamParser struct {
	StreamParser
    numTruncatedBytes uint
}

func (this *MPEGVideoStreamParser) registerReadInterest(to []byte, maxSize uint) {
	//this.StartOfFrame = this.buffTo
	//this.SavedTo = to
	//fLimit = to + maxSize
}

func (this *MPEGVideoStreamParser) numTruncatedBytes() uint {
    return this.numTruncatedBytes
}

func (this *MPEGVideoStreamParser) save4Bytes(word uint) {
}

func (this *MPEGVideoStreamParser) curFrameSize() uint {
    return 0
}

func (this *MPEGVideoStreamParser) setParseState() {
    this.saveParserState()
}
