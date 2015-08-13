package liveMedia

import (
	"fmt"
)

var BANK_SIZE = 150000
var NO_MORE_BUFFERED_INPUT = 1

type StreamParser struct {
    curBankNum            uint
	curParserIndex        uint
    totNumValidBytes      uint
    savedParserIndex      uint
    remainingUnparsedBits uint
    savedRemainingUnparsedBits uint
	haveSeenEOF           bool
    inputSouce            *FramedSource
	bank                  []byte
}

func (sp *StreamParser) InitStreamParser() {
}

func (sp *StreamParser) get4Bytes() uint {
    result := this.test4Bytes()
	sp.curParserIndex += 4
    sp.remainingUnparsedBits = 0
    return result
}

func (sp *StreamParser) get2Bytes() uint {
    sp.ensureValidBytes(2)

    ptr := sp.nextToParse()
    result := (ptr[0]<<8) | ptr[1]

	sp.curParserIndex += 2
    sp.remainingUnparsedBits = 0
    return result
}

func (sp *StreamParser) get1Byte() byte {
    sp.ensureValidBytes(1)
	sp.curParserIndex++
    return this.curBank()[sp.curParserIndex:]
}

func (sp *StreamParser) test4Bytes() uint {
    sp.ensureValidBytes(4)

    ptr := this.nextToParse()
	return (ptr[0] << 24) | (ptr[1] << 16) | (ptr[2] << 8) | ptr[3]
}

func (sp *StreamParser) testBytes(to []byte, numBytes uint) {
    sp.ensureValidBytes(numBytes)
    to = this.nextToParse()[:numBytes]
}

func (sp *StreamParser) skipBytes(numBytes uint) {
    sp.ensureValidBytes(numBytes)
	sp.curParserIndex += numBytes
}

func (sp *StreamParser) curBank() []byte {
	return sp.bank
}

func (sp *StreamParser) nextToParse() []byte {
    return this.curBank()[this.curParserIndex]
}

func (sp *StreamParser) curOffset() uint {
    return sp.curParserIndex
}

func (sp *StreamParser) HaveSeenEOF() bool {
	return sp.haveSeenEOF
}

func (sp *StreamParser) saveParserState() {
    this.savedParserIndex = this.curParserIndex
    this.savedRemainingUnparsedBits = this.remainingUnparsedBits
}

func (sp *StreamParser) TotNumValidBytes() uint {
    return sp.totNumValidBytes
}

func (sp *StreamParser) ensureValidBytes(numBytesNeeded uint) {
    if this.curParserIndex + numBytesNeeded <= this.totNumValidBytes {
        return
    }

    this.ensureValidBytes1(numBytesNeeded)
}

func (sp *StreamParser) ensureValidBytes1(numBytesNeeded uint) uint {
    maxInputFrameSize := this.inputSource.maxFrameSize()
    if maxInputFrameSize > numBytesNeeded {
        numBytesNeeded = maxInputFrameSize
    }

    if this.curParserIndex + numBytesNeeded > BANK_SIZE {
        numBytesToSave = this.totNumValidBytes + this.savedParserIndex
        from := this.curBank()

        this.curBankNum = (this.curBankNum + 1) % 2
        this.curBank = this.bank[this.curBandNum:]

        this.curParserIndex -= this.savedParserIndex
        this.savedParserIndex = 0
        this.totNumValidBytes = numBytesToSave
    }

    if this.curParserIndex + numBytesNeeded > BANK_SIZE {
        panic("StreamParser Internal error")
    }

    // Try to read as many new bytes as will fit in the current bank:
    maxNumBytesToRead = BANK_SIZE - this.totNumValidBytes
    this.inputSource.getNextFrame()
    return NO_MORE_BUFFERED_INPUT
}
