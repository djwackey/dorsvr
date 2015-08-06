package liveMedia

import (
	"fmt"
)

type StreamParser struct {
	curParserIndex uint
    haveSeenEOF bool
    bank []byte
}

func (sp *StreamParser) get4Bytes() {
	sp.curParserIndex += 4
}

func (sp *StreamParser) get2Bytes() {
	sp.curParserIndex += 2
}

func (sp *StreamParser) get1Bytes() {
	sp.curParserIndex++
	fmt.Println("get1Bytes")
}

func (sp *StreamParser) test4Bytes(bytes []byte) byte {
	return (bytes[0] << 24) | (bytes[1] << 16) | (bytes[2] << 8) | bytes[3]
}

func (sp *StreamParser) skipBytes(numBytes uint) {
	sp.curParserIndex += numBytes
}

func (sp *StreamParser) curBank() []byte {
    return sp.bank
}

func (sp *StreamParser) haveSeenEOF() bool {
    return sp.haveSeenEOF
}

func (sp *StreamParser) saveParserState() {
}
