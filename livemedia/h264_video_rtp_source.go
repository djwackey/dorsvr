package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

type H264VideoRTPSource struct {
	MultiFramedRTPSource
	curPacketNALUnitType        uint
	currentPacketBeginsFrame    bool
	currentPacketCompletesFrame bool
}

func newH264VideoRTPSource(RTPgs *gs.GroupSock,
	rtpPayloadFormat, rtpTimestampFrequency uint32) *H264VideoRTPSource {
	source := new(H264VideoRTPSource)

	source.initMultiFramedRTPSource(source, RTPgs,
		rtpPayloadFormat, rtpTimestampFrequency, newH264BufferedPacketFactory())
	source.setSpecialHeaderHandler(source.processSpecialHeader)
	return source
}

func (s *H264VideoRTPSource) processSpecialHeader(packet IBufferedPacket) (
	resultSpecialHeaderSize uint32, processOK bool) {
	headerStart, packetSize := packet.data(), packet.dataSize()

	var expectedHeaderSize uint32

	// Check if the type field is 28 (FU-A) or 29 (FU-B)
	s.curPacketNALUnitType = uint(headerStart[0]) & 0x1F

	switch s.curPacketNALUnitType {
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
			s.currentPacketBeginsFrame = true
		} else {
			expectedHeaderSize = 2

			if packetSize < expectedHeaderSize {
				return
			}

			s.currentPacketBeginsFrame = false
		}

		s.currentPacketCompletesFrame = (endBit != 0)
	default:
		s.currentPacketBeginsFrame = true
		s.currentPacketCompletesFrame = true
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

type H264BufferedPacket struct {
	BufferedPacket
	source *H264VideoRTPSource
}

type H264BufferedPacketFactory struct {
	BufferedPacketFactory
}

func newH264BufferedPacket(source *H264VideoRTPSource) *H264BufferedPacket {
	packet := new(H264BufferedPacket)
	packet.initBufferedPacket()
	packet.source = source
	packet.nextEnclosedFrameProc = packet.nextEnclosedFrameSize
	return packet
}

func (p *H264BufferedPacket) nextEnclosedFrameSize(buff []byte, size uint) uint32 {
	framePtr, dataSize := buff[p.head:], p.tail-p.head

	var resultNALUSize, frameSize uint32

	switch p.source.curPacketNALUnitType {
	case 24, 25: // STAP-A or STAP-B
		// The first two bytes are NALU size:
		if dataSize >= 2 {
			resultNALUSize = (uint32(framePtr[0]) << 8) | uint32(framePtr[1])
			framePtr = framePtr[2:]
		}
	case 26: // MTAP16
		// The first two bytes are NALU size.
		// The next three are the DOND and TS offset:
		if dataSize >= 5 {
			resultNALUSize = (uint32(framePtr[0]) << 8) | uint32(framePtr[1])
			framePtr = framePtr[5:]
		}
	case 27: // MTAP24
		// The first two bytes are NALU size.
		// The next four are the DOND and TS offset:
		if dataSize >= 6 {
			resultNALUSize = (uint32(framePtr[0]) << 8) | uint32(framePtr[1])
			framePtr = framePtr[6:]
		}
	default:
		// Common case: We use the entire packet data:
		return dataSize
	}

	if resultNALUSize <= dataSize {
		frameSize = resultNALUSize
	} else {
		frameSize = dataSize
	}

	return frameSize
}

func newH264BufferedPacketFactory() IBufferedPacketFactory {
	return new(H264BufferedPacketFactory)
}

func (f *H264BufferedPacketFactory) createNewPacket(source interface{}) IBufferedPacket {
	var h264VideoRTPSource *H264VideoRTPSource
	if source != nil {
		h264VideoRTPSource = source.(*H264VideoRTPSource)
	}
	return newH264BufferedPacket(h264VideoRTPSource)
}
