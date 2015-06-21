package liveMedia

import (
    "fmt"
)

type StreamParser struct {
    mCurParserIndex uint
}

func (sp *StreamParser) get4Bytes() {
    sp.mCurParserIndex += 4
}

func (sp *StreamParser) get2Bytes() {
    sp.mCurParserIndex += 2
}

func (sp *StreamParser) get1Bytes() {
    sp.mCurParserIndex++
    fmt.Println("get1Bytes")
}

func (sp *StreamParser) test4Bytes(bytes []byte) byte {
    return (bytes[0]<<24) | (bytes[1]<<16) | (bytes[2]<<8) | bytes[3]
}

func (sp *StreamParser) skipBytes(numBytes uint) {
    sp.mCurParserIndex += numBytes
}
