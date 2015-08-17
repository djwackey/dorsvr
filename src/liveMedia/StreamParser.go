package liveMedia

import (
//"fmt"
)

var BANK_SIZE uint = 150000
var NO_MORE_BUFFERED_INPUT uint = 1

type StreamParser struct {
	curBankNum                 uint
	curParserIndex             uint
	totNumValidBytes           uint
	savedParserIndex           uint
	remainingUnparsedBits      uint
	savedRemainingUnparsedBits uint
	haveSeenEOF                bool
	inputSource                *FramedSource
	bank                       []byte
	curBank                    []byte
}

func (sp *StreamParser) InitStreamParser() {
}

func (sp *StreamParser) get4Bytes() uint {
	result := sp.test4Bytes()
	sp.curParserIndex += 4
	sp.remainingUnparsedBits = 0
	return result
}

func (sp *StreamParser) get2Bytes() uint {
	sp.ensureValidBytes(2)

	ptr := sp.nextToParse()
	result := (ptr[0] << 8) | ptr[1]

	sp.curParserIndex += 2
	sp.remainingUnparsedBits = 0
	return uint(result)
}

func (sp *StreamParser) get1Byte() uint {
	sp.ensureValidBytes(1)
	sp.curParserIndex++
	return uint(sp.CurBank()[sp.curParserIndex])
}

func (sp *StreamParser) test4Bytes() uint {
	sp.ensureValidBytes(4)

	ptr := sp.nextToParse()
	return uint((ptr[0] << 24) | (ptr[1] << 16) | (ptr[2] << 8) | ptr[3])
}

func (sp *StreamParser) testBytes(to []byte, numBytes uint) {
	sp.ensureValidBytes(numBytes)
	to = sp.nextToParse()[:numBytes]
}

func (sp *StreamParser) skipBytes(numBytes uint) {
	sp.ensureValidBytes(numBytes)
	sp.curParserIndex += numBytes
}

func (sp *StreamParser) CurBank() []byte {
	return sp.curBank
}

func (sp *StreamParser) nextToParse() []byte {
	return sp.CurBank()[sp.curParserIndex:]
}

func (sp *StreamParser) curOffset() uint {
	return sp.curParserIndex
}

func (sp *StreamParser) HaveSeenEOF() bool {
	return sp.haveSeenEOF
}

func (sp *StreamParser) saveParserState() {
	sp.savedParserIndex = sp.curParserIndex
	sp.savedRemainingUnparsedBits = sp.remainingUnparsedBits
}

func (sp *StreamParser) TotNumValidBytes() uint {
	return sp.totNumValidBytes
}

func (sp *StreamParser) ensureValidBytes(numBytesNeeded uint) {
	if sp.curParserIndex+numBytesNeeded <= sp.totNumValidBytes {
		return
	}

	sp.ensureValidBytes1(numBytesNeeded)
}

func (sp *StreamParser) ensureValidBytes1(numBytesNeeded uint) uint {
	maxInputFrameSize := sp.inputSource.maxFrameSize()
	if maxInputFrameSize > numBytesNeeded {
		numBytesNeeded = maxInputFrameSize
	}

	if sp.curParserIndex+numBytesNeeded > BANK_SIZE {
		numBytesToSave := sp.totNumValidBytes + sp.savedParserIndex
		//from := sp.CurBank()

		sp.curBankNum = (sp.curBankNum + 1) % 2
		sp.curBank = sp.bank[sp.curBankNum:]

		sp.curParserIndex -= sp.savedParserIndex
		sp.savedParserIndex = 0
		sp.totNumValidBytes = numBytesToSave
	}

	if sp.curParserIndex+numBytesNeeded > BANK_SIZE {
		panic("StreamParser Internal error")
	}

	// Try to read as many new bytes as will fit in the current bank:
	//maxNumBytesToRead := BANK_SIZE - sp.totNumValidBytes
	//sp.inputSource.getNextFrame()
	return NO_MORE_BUFFERED_INPUT
}
