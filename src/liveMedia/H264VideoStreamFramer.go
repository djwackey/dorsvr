package liveMedia

import "fmt"

type H264VideoStreamFramer struct {
	MPEGVideoStreamFramer
	parser    *H264VideoStreamParser
	frameRate float64
}

type H264VideoStreamParser struct {
	MPEGVideoStreamParser
	outputStartCodeSize        int
	haveSeenFirstStartCode     bool
	haveSeenFirstByteOfNALUnit bool
}

func NewH264VideoStreamFramer(inputSource IFramedSource) *H264VideoStreamFramer {
	h264VideoStreamFramer := new(H264VideoStreamFramer)
	h264VideoStreamFramer.parser = NewH264VideoStreamParser()
	h264VideoStreamFramer.inputSource = inputSource
	h264VideoStreamFramer.frameRate = 25.0
	return h264VideoStreamFramer
}

func (this *H264VideoStreamFramer) getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}) {
	this.inputSource.getNextFrame(buffTo, maxSize, afterGettingFunc)
}

func (this *H264VideoStreamFramer) setSPSandPPS(sPropParameterSetsStr string) {
	//sPropRecords := parseSPropParameterSets()
}

func NewH264VideoStreamParser() *H264VideoStreamParser {
	return new(H264VideoStreamParser)
}

func (this *H264VideoStreamParser) Parse(data []byte) {
	if !this.haveSeenFirstStartCode {
		first4Bytes := this.test4Bytes(data)

		if first4Bytes == 0x00000001 {
			fmt.Println("parse", first4Bytes)
		}

		this.skipBytes(4)
		this.haveSeenFirstStartCode = true
	}
}
