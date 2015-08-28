package liveMedia

type MPEGVideoStreamParser struct {
	StreamParser
    limit                  []byte
	buffTo                 []byte
    saveTo                 []byte
	startOfFrame           []byte
	numTruncatedBytes      uint
    savedNumTruncatedBytes uint
	usingSource            *H264VideoStreamFramer
}

func (this *MPEGVideoStreamParser) registerReadInterest(buffTo []byte, maxSize uint) {
	this.startOfFrame = this.buffTo
    this.buffTo = buffTo
	this.savedTo = buffTo
    this.limit = buffTo[maxSize:]
    this.numTruncatedBytes = 0
    this.savedNumTruncatedBytes = 0
}

func (this *MPEGVideoStreamParser) NumTruncatedBytes() uint {
	return this.numTruncatedBytes
}

func (this *MPEGVideoStreamParser) saveByte(ubyte uint) {
}

func (this *MPEGVideoStreamParser) save4Bytes(word uint) {
}

func (this *MPEGVideoStreamParser) curFrameSize() uint {
	return 0
}

func (this *MPEGVideoStreamParser) setParseState() {
	this.saveParserState()
}
