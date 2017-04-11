package livemedia

type MPEGVideoStreamParser struct {
	StreamParser
	buffTo                 []byte
	saveTo                 []byte
	startOfFrame           []byte
	numLimitBytes          uint
	numSavedBytes          uint
	numTruncatedBytes      uint
	savedCurFrameSize      uint
	savedNumTruncatedBytes uint
	usingSource            IFramedSource
}

func (p *MPEGVideoStreamParser) initMPEGVideoStreamParser(usingSource, inputSource IFramedSource,
	clientOnInputCloseFunc interface{}) {
	p.usingSource = usingSource

	p.initStreamParser(inputSource, clientOnInputCloseFunc, p.restoreSavedParserState)
}

func (p *MPEGVideoStreamParser) restoreSavedParserState() {
	p.restore()
	p.buffTo = p.saveTo
	p.numSavedBytes = p.savedCurFrameSize
	p.numTruncatedBytes = p.savedNumTruncatedBytes
}

func (p *MPEGVideoStreamParser) registerReadInterest(buffTo []byte, maxSize uint) {
	p.buffTo = buffTo
	p.saveTo = buffTo
	p.numLimitBytes = maxSize
	p.numSavedBytes = 0
	p.numTruncatedBytes = 0
	p.savedCurFrameSize = 0
	p.savedNumTruncatedBytes = 0
	p.startOfFrame = p.buffTo
}

func (p *MPEGVideoStreamParser) saveByte(b uint) {
	if p.numSavedBytes >= p.numLimitBytes {
		p.numTruncatedBytes += 1
		return
	}
	p.buffTo[p.numSavedBytes:][0] = byte(b)
	p.numSavedBytes += 1
}

func (p *MPEGVideoStreamParser) save4Bytes(word uint) {
	if p.numSavedBytes+4 > p.numLimitBytes {
		p.numTruncatedBytes += 4
		return
	}

	p.buffTo[p.numSavedBytes:][0] = byte(word >> 24)
	p.buffTo[p.numSavedBytes:][1] = byte(word >> 16)
	p.buffTo[p.numSavedBytes:][2] = byte(word >> 8)
	p.buffTo[p.numSavedBytes:][3] = byte(word)
	p.numSavedBytes += 4
}

func (p *MPEGVideoStreamParser) curFrameSize() uint {
	return p.numSavedBytes
}

func (p *MPEGVideoStreamParser) setParseState() {
	p.saveTo = p.buffTo
	p.savedCurFrameSize = p.numSavedBytes
	p.savedNumTruncatedBytes = p.numTruncatedBytes
	p.saveParserState()
}
