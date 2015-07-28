package liveMedia

import (
	"fmt"
	. "include"
)

//////// H264VideoStreamParser ////////
type H264VideoStreamParser struct {
	MPEGVideoStreamParser
	outputStartCodeSize        int
	haveSeenFirstStartCode     bool
	haveSeenFirstByteOfNALUnit bool
}

func NewH264VideoStreamParser() *H264VideoStreamParser {
	return new(H264VideoStreamParser)
}

func (this *H264VideoStreamParser) parse(data []byte) {
	if !this.haveSeenFirstStartCode {
		first4Bytes := this.test4Bytes(data)

		if first4Bytes == 0x00000001 {
			fmt.Println("parse", first4Bytes)
		}

		this.skipBytes(4)
		this.haveSeenFirstStartCode = true
	}
}

func (this *H264VideoStreamParser) analyzeSPSData() {
}

//////// H264VideoStreamFramer ////////
type H264VideoStreamFramer struct {
	MPEGVideoStreamFramer
	parser               *H264VideoStreamParser
	nextPresentationTime Timeval
	lastSeenSPS          []byte
	lastSeenPPS          []byte
	lastSeenSPSSize      uint
	lastSeenPPSSize      uint
	frameRate            float64
}

func NewH264VideoStreamFramer(inputSource IFramedSource) *H264VideoStreamFramer {
	h264VideoStreamFramer := new(H264VideoStreamFramer)
	h264VideoStreamFramer.parser = NewH264VideoStreamParser()
	h264VideoStreamFramer.inputSource = inputSource
	h264VideoStreamFramer.frameRate = 25.0
	return h264VideoStreamFramer
}

func (this *H264VideoStreamFramer) getNextFrame(buffTo []byte, maxSize uint, afterGettingFunc interface{}) {
	fmt.Println("H264VideoStreamFramer::getNextFrame")
	this.inputSource.getNextFrame(buffTo, maxSize, afterGettingFunc)
}

func (this *H264VideoStreamFramer) setSPSandPPS(sPropParameterSetsStr string) {
	sPropRecords, numSPropRecords := parseSPropParameterSets(sPropParameterSetsStr)
	var i uint
	for i = 0; i < numSPropRecords; i++ {
		if sPropRecords[i].sPropLength == 0 {
			continue
		}

		nalUnitType := (sPropRecords[i].sPropBytes[0]) & 0x1F
		if nalUnitType == 7 { /* SPS */
			this.saveCopyOfSPS(sPropRecords[i].sPropBytes, sPropRecords[i].sPropLength)
		} else if nalUnitType == 8 { /* PPS */
			this.saveCopyOfPPS(sPropRecords[i].sPropBytes, sPropRecords[i].sPropLength)
		}
	}
}

func (this *H264VideoStreamFramer) saveCopyOfSPS(from []byte, size uint) {
	this.lastSeenSPS = make([]byte, size)
	this.lastSeenSPS = from
	this.lastSeenSPSSize = size
}

func (this *H264VideoStreamFramer) saveCopyOfPPS(from []byte, size uint) {
	this.lastSeenPPS = make([]byte, size)
	this.lastSeenPPS = from
	this.lastSeenPPSSize = size
}
