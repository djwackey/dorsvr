package liveMedia

import (
	//"fmt"
	. "groupsock"
)

type H264VideoRTPSource struct {
	MultiFramedRTPSource
	curPacketNALUnitType        uint
	currentPacketBeginsFrame    bool
	currentPacketCompletesFrame bool
}

func NewH264VideoRTPSource(RTPgs *GroupSock,
	rtpPayloadFormat, rtpTimestampFrequency uint) *H264VideoRTPSource {
	source := new(H264VideoRTPSource)
	source.InitMultiFramedRTPSource(source, RTPgs, rtpPayloadFormat, rtpTimestampFrequency)
	return source
}

func (source *H264VideoRTPSource) processSpecialHeader(packet *BufferedPacket) (resultSpecialHeaderSize uint, processOK bool) {
	headerStart, packetSize := packet.data(), packet.dataSize()

	var expectedHeaderSize uint

	// Check if the type field is 28 (FU-A) or 29 (FU-B)
	source.curPacketNALUnitType = uint(headerStart[0]) & 0x1F

	switch source.curPacketNALUnitType {
	case 24: // STAP-A
		expectedHeaderSize = 1
	case 25, 26, 27: // STAP-B, MTAP16, or MTAP24
		expectedHeaderSize = 3
	case 28, 29: // FU-A or FU-B
		startBit := (headerStart[1] & 0x80) != 0
		endBit := headerStart[1] & 0x40

		if startBit {
			expectedHeaderSize = 1

			if packetSize < expectedHeaderSize {
				return
			}

			headerStart[1] = (headerStart[0] & 0xE0) + (headerStart[1] & 0x1F)
			source.currentPacketBeginsFrame = true
		} else {
			expectedHeaderSize = 2

			if packetSize < expectedHeaderSize {
				return
			}

			source.currentPacketBeginsFrame = false
		}

		source.currentPacketCompletesFrame = (endBit != 0)
	default:
		source.currentPacketBeginsFrame = true
		source.currentPacketCompletesFrame = true
	}

	resultSpecialHeaderSize, processOK = expectedHeaderSize, true
	return
}

type SPropRecord struct {
	sPropLength uint
	sPropBytes  []byte
}

func parseSPropParameterSets(sPropParameterSetsStr string) ([]*SPropRecord, uint) {
	return nil, 0
}
