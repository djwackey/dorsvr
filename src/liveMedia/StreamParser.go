package liveMedia

import (
	"fmt"
)

type StreamParser struct {
	curParserIndex   uint
    totNumValidBytes uint
	haveSeenEOF      bool
	bank             []byte
}

func (sp *StreamParser) get4Bytes() {
	sp.curParserIndex += 4
}

func (sp *StreamParser) get2Bytes() {
	sp.curParserIndex += 2
}

func (sp *StreamParser) get1Byte() uint {
	sp.curParserIndex++
	fmt.Println("get1Byte")
	return 0
}

func (sp *StreamParser) test4Bytes() uint {
	var bytes []uint
	return (bytes[0] << 24) | (bytes[1] << 16) | (bytes[2] << 8) | bytes[3]
}

func (sp *StreamParser) testBytes(to []byte, numBytes uint) {
}

func (sp *StreamParser) skipBytes(numBytes uint) {
	sp.curParserIndex += numBytes
}

func (sp *StreamParser) curBank() []byte {
	return sp.bank
}

func (sp *StreamParser) curOffset() uint {
    return sp.curParserIndex
}

func (sp *StreamParser) HaveSeenEOF() bool {
	return sp.haveSeenEOF
}

func (sp *StreamParser) saveParserState() {
}

func (sp *StreamParser) TotNumValidBytes() uint {
    return sp.totNumValidBytes
}
