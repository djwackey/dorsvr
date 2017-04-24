package livemedia

import (
	"errors"
	sys "syscall"

	"github.com/djwackey/gitea/log"
)

const bankSize uint = 150000

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
	clientOnInputCloseFunc     interface{}
	restoreParserStateFunc     interface{}
	lastSeenPresentationTime   sys.Timeval
}

func (p *StreamParser) initStreamParser(inputSource IFramedSource,
	clientOnInputCloseFunc, restoreParserStateFunc interface{}) {
	p.inputSource = inputSource
	p.clientOnInputCloseFunc = clientOnInputCloseFunc
	p.restoreParserStateFunc = restoreParserStateFunc

	p.bank[0] = make([]byte, bankSize)
	p.bank[1] = make([]byte, bankSize)

	p.curBank = p.bank[p.curBankNum]
}

func (p *StreamParser) restore() {
	p.curParserIndex = p.savedParserIndex
	p.remainingUnparsedBits = p.savedRemainingUnparsedBits
}

func (p *StreamParser) get4Bytes() (n uint, err error) {
	if n, err = p.test4Bytes(); err != nil {
		return
	}

	p.curParserIndex += 4
	p.remainingUnparsedBits = 0
	return
}

func (p *StreamParser) get2Bytes() (n uint, err error) {
	if err = p.ensureValidBytes(2); err != nil {
		return
	}

	ptr := p.nextToParse()
	n = uint((ptr[0] << 8) | ptr[1])

	p.curParserIndex += 2
	p.remainingUnparsedBits = 0
	return
}

func (p *StreamParser) get1Byte() (uint, error) {
	if err := p.ensureValidBytes(1); err != nil {
		return 0, err
	}

	p.curParserIndex++
	return uint(p.curBank[p.curParserIndex]), nil
}

func (p *StreamParser) test4Bytes() (uint, error) {
	if err := p.ensureValidBytes(4); err != nil {
		return 0, err
	}

	ptr := p.nextToParse()
	return uint(Uint32(ptr[:4])), nil
}

func (p *StreamParser) testBytes(to []byte, numBytes uint) error {
	if err := p.ensureValidBytes(numBytes); err != nil {
		return err
	}

	copy(to, p.nextToParse()[:numBytes])
	return nil
}

func (p *StreamParser) skipBytes(numBytes uint) error {
	if err := p.ensureValidBytes(numBytes); err != nil {
		return err
	}

	p.curParserIndex += numBytes
	return nil
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

func (p *StreamParser) ensureValidBytes(numBytesNeeded uint) error {
	if p.curParserIndex+numBytesNeeded <= p.totNumValidBytes {
		return nil
	}

	return p.ensureValidBytes1(numBytesNeeded)
}

func (p *StreamParser) ensureValidBytes1(numBytesNeeded uint) error {
	maxInputFrameSize := p.inputSource.maxFrameSize()
	if maxInputFrameSize > numBytesNeeded {
		numBytesNeeded = maxInputFrameSize
	}

	// First, check whether these new bytes would overflow the current
	// bank.  If so, start using a new bank now.
	if p.curParserIndex+numBytesNeeded > bankSize {
		numBytesToSave := p.totNumValidBytes - p.savedParserIndex
		from := p.curBank[p.saveParserIndex:]

		p.curBankNum = (p.curBankNum + 1) % 2
		p.curBank = p.bank[p.curBankNum]
		copy(p.curBank, from[:numBytesToSave])
		//p.curBank = p.curBank[p.saveParserIndex : p.saveParserIndex+numBytesToSave]

		p.curParserIndex -= p.savedParserIndex
		p.savedParserIndex = 0
		p.totNumValidBytes = numBytesToSave
	}

	if p.curParserIndex+numBytesNeeded > bankSize {
		panic("StreamParser Internal error")
	}

	// Try to read as many new bytes as will fit in the current bank:
	maxNumBytesToRead := bankSize - p.totNumValidBytes
	err := p.inputSource.GetNextFrame(p.curBank[p.totNumValidBytes:], maxNumBytesToRead,
		p.afterGettingBytes, p.onInputClosure)
	if err != nil {
		// maybe haven't more data to read, reached file's end.
		return err
	}

	return errors.New("reading bytes from input source.")
}

func (p *StreamParser) afterGettingBytes(numBytesRead, numTruncatedBytes uint, presentationTime sys.Timeval) {
	if p.totNumValidBytes+numBytesRead > bankSize {
		log.Warn("StreamParser::afterGettingBytes() "+
			"warning: read %d bytes; expected no more than %d\n", numBytesRead, bankSize-p.totNumValidBytes)
	}

	p.lastSeenPresentationTime = presentationTime

	p.totNumValidBytes += numBytesRead
	//log.Debug("[StreamParser::afterGettingBytes] totNumValidBytes: %d", p.totNumValidBytes)

	// Continue our original calling source where it left off:
	p.restoreParserStateFunc.(func())()
}

func (p *StreamParser) onInputClosure() {
	if !p.haveSeenEOF {
		log.Debug("[StreamParser::onInputClosure] haveSeenEOF: true")
		p.haveSeenEOF = true
		//p.afterGettingBytes(0, 0, p.lastSeenPresentationTime)
	} else {
		// We're hitting EOF for the second time.  Now, we handle the source input closure:
		if p.clientOnInputCloseFunc != nil {
			p.clientOnInputCloseFunc.(func())()
		}
	}
}
