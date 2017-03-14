package livemedia

import (
	"fmt"
	sys "syscall"
)

var bankSize uint = 150000

type StreamParser struct {
	curBankNum                 uint
	curParserIndex             uint
	saveParserIndex            uint
	totNumValidBytes           uint
	savedParserIndex           uint
	remainingUnparsedBits      uint
	savedRemainingUnparsedBits uint
	haveSeenEOF                bool
	inputSource                IFramedSource
	bank                       [2][]byte
	curBank                    []byte
	clientContinueFunc         interface{}
	clientOnInputCloseFunc     interface{}
	lastSeenPresentationTime   sys.Timeval
}

func (p *StreamParser) initStreamParser(inputSource IFramedSource,
	clientOnInputCloseFunc, clientContinueFunc interface{}) {
	p.inputSource = inputSource
	p.clientContinueFunc = clientContinueFunc
	p.clientOnInputCloseFunc = clientOnInputCloseFunc

	p.bank[0] = make([]byte, bankSize)
	p.bank[1] = make([]byte, bankSize)

	p.curBank = p.bank[p.curBankNum]
}

func (p *StreamParser) restoreSavedParserState() {
	p.curParserIndex = p.savedParserIndex
	p.remainingUnparsedBits = p.savedRemainingUnparsedBits
}

func (p *StreamParser) get4Bytes() (uint, bool) {
	result, ok := p.test4Bytes()
	if !ok {
		return 0, false
	}

	p.curParserIndex += 4
	p.remainingUnparsedBits = 0
	return result, true
}

func (p *StreamParser) get2Bytes() (uint, bool) {
	if p.ensureValidBytes(2) {
		return 0, false
	}

	ptr := p.nextToParse()
	result := (ptr[0] << 8) | ptr[1]

	p.curParserIndex += 2
	p.remainingUnparsedBits = 0
	return uint(result), true
}

func (p *StreamParser) get1Byte() (uint, bool) {
	if p.ensureValidBytes(1) {
		return 0, false
	}

	p.curParserIndex++
	return uint(p.curBank[p.curParserIndex]), true
}

func (p *StreamParser) test4Bytes() (uint, bool) {
	if p.ensureValidBytes(4) {
		return 0, false
	}

	ptr := p.nextToParse()
	return uint((ptr[0] << 24) | (ptr[1] << 16) | (ptr[2] << 8) | ptr[3]), true
}

func (p *StreamParser) testBytes(to []byte, numBytes uint) bool {
	if p.ensureValidBytes(numBytes) {
		return false
	}

	to = p.nextToParse()[:numBytes]
	return true
}

func (p *StreamParser) skipBytes(numBytes uint) bool {
	if p.ensureValidBytes(numBytes) {
		return false
	}

	p.curParserIndex += numBytes
	return true
}

func (p *StreamParser) nextToParse() []byte {
	return p.curBank[p.curParserIndex:]
}

func (p *StreamParser) curOffset() uint {
	return p.curParserIndex
}

func (p *StreamParser) saveParserState() {
	p.savedParserIndex = p.curParserIndex
	p.savedRemainingUnparsedBits = p.remainingUnparsedBits
}

func (p *StreamParser) ensureValidBytes(numBytesNeeded uint) bool {
	if p.curParserIndex+numBytesNeeded <= p.totNumValidBytes {
		return false
	}

	return p.ensureValidBytes1(numBytesNeeded)
}

func (p *StreamParser) ensureValidBytes1(numBytesNeeded uint) bool {
	maxInputFrameSize := p.inputSource.maxFrameSize()
	if maxInputFrameSize > numBytesNeeded {
		numBytesNeeded = maxInputFrameSize
	}

	if p.curParserIndex+numBytesNeeded > bankSize {
		numBytesToSave := p.totNumValidBytes + p.savedParserIndex

		p.curBankNum = (p.curBankNum + 1) % 2
		p.curBank = p.bank[p.curBankNum]
		p.curBank = p.curBank[p.saveParserIndex : p.saveParserIndex+numBytesToSave]

		p.curParserIndex -= p.savedParserIndex
		p.savedParserIndex = 0
		p.totNumValidBytes = numBytesToSave
	}

	if p.curParserIndex+numBytesNeeded > bankSize {
		panic("StreamParser Internal error")
	}

	// Try to read as many new bytes as will fit in the current bank:
	maxNumBytesToRead := bankSize - p.totNumValidBytes
	p.inputSource.GetNextFrame(p.curBank[p.totNumValidBytes:], maxNumBytesToRead, p.afterGettingBytes, p.onInputClosure)
	// no more buffered input
	return true
}

func (p *StreamParser) afterGettingBytes(numBytesRead, numTruncatedBytes uint, presentationTime sys.Timeval) {
	if p.totNumValidBytes+numBytesRead > bankSize {
		fmt.Printf("StreamParser::afterGettingBytes() "+
			"warning: read %d bytes; expected no more than %d\n", numBytesRead, bankSize-p.totNumValidBytes)
	}

	p.lastSeenPresentationTime = presentationTime

	p.totNumValidBytes += numBytesRead

	// Continue our original calling source where it left off:
	p.restoreSavedParserState()

	p.clientContinueFunc.(func())()
}

func (p *StreamParser) onInputClosure() {
	if !p.haveSeenEOF {
		p.haveSeenEOF = true
		p.afterGettingBytes(0, 0, p.lastSeenPresentationTime)
	} else {
		fmt.Println("StreamParser::onInputClosure")
		// We're hitting EOF for the second time.  Now, we handle the source input closure:
		p.haveSeenEOF = false
		if p.clientOnInputCloseFunc != nil {
			p.clientOnInputCloseFunc.(func())()
		}
	}
}
