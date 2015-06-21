package liveMedia

import "fmt"

type H264VideoStreamFramer struct {
	mParser    *H264VideoStreamParser
	mFrameRate float64
}

type H264VideoStreamParser struct {
	StreamParser
	mOutputStartCodeSize        int
	mHaveSeenFirstStartCode     bool
	mHaveSeenFirstByteOfNALUnit bool
}

func NewH264VideoStreamFramer() *H264VideoStreamFramer {
	parser := NewH264VideoStreamParser()
	frameRate := 25.0
	return &H264VideoStreamFramer{parser, frameRate}
}

func NewH264VideoStreamParser() *H264VideoStreamParser {
	return &H264VideoStreamParser{}
}

func (this *H264VideoStreamParser) Parse(data []byte) {
	if !this.mHaveSeenFirstStartCode {
		first4Bytes := this.test4Bytes(data)

		if first4Bytes == 0x00000001 {
			fmt.Println("parse", first4Bytes)
		}

		this.skipBytes(4)
		this.mHaveSeenFirstStartCode = true
	}
}
