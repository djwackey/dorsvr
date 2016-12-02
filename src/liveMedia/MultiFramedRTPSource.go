package liveMedia

import (
	"fmt"
	. "groupsock"
	"utils"
)

type MultiFramedRTPSource struct {
	RTPSource
	needDelivery                bool
	areDoingNetworkReads        bool
	currentPacketBeginsFrame    bool
	currentPacketCompletesFrame bool
	packetLossInFragmentedFrame bool
	packetReadInProgress        *BufferedPacket
	h264VideoRTPSource          *H264VideoRTPSource
	reOrderingBuffer            *ReorderingPacketBuffer
}

func (source *MultiFramedRTPSource) InitMultiFramedRTPSource(isource IFramedSource,
	RTPgs *GroupSock, rtpPayloadFormat, rtpTimestampFrequency uint) {
	packetFactory := NewBufferedPacketFactory()

	source.h264VideoRTPSource = isource.(*H264VideoRTPSource)

	source.reOrderingBuffer = NewReorderingPacketBuffer(packetFactory)
	source.InitRTPSouce(isource, RTPgs, rtpPayloadFormat, rtpTimestampFrequency)
}

func (source *MultiFramedRTPSource) doGetNextFrame() {
	source.rtpInterface.startNetworkReading(source.networkReadHandler)
}

func (source *MultiFramedRTPSource) doGetNextFrame1() {
	source.needDelivery = true
	for source.needDelivery {
		var packetLossPrecededThis bool
		var nextPacket *BufferedPacket

		nextPacket, packetLossPrecededThis = source.reOrderingBuffer.getNextCompletedPacket()
		if nextPacket == nil {
			fmt.Println("failed to get next completed packet.")
			break
		}

		source.needDelivery = false

		if nextPacket.UseCount() == 0 {
			// Before using the packet, check whether it has a special header
			// that needs to be processed:
			specialHeaderSize, processOK := source.processSpecialHeader(nextPacket)
			if !processOK {
				// Something's wrong with the header; reject the packet:
				source.reOrderingBuffer.releaseUsedPacket(nextPacket)
				source.needDelivery = true

				fmt.Println("failed to process special header.")
				break
			}
			nextPacket.skip(specialHeaderSize)
		}

		if source.currentPacketBeginsFrame {
			source.packetLossInFragmentedFrame = false
		} else if packetLossPrecededThis {
			source.packetLossInFragmentedFrame = true
		}

		if source.packetLossInFragmentedFrame {
			source.reOrderingBuffer.releaseUsedPacket(nextPacket)
			source.needDelivery = true
			break
		}

		var frameSize uint
		nextPacket.use(source.buffTo, source.maxSize)

		source.frameSize += frameSize

		if !nextPacket.hasUsableData() {
			source.reOrderingBuffer.releaseUsedPacket(nextPacket)
		}

		if source.currentPacketCompletesFrame {
			source.afterGetting()
		} else {
			source.needDelivery = true
		}

		fmt.Println("MultiFramedRTPSource::doGetNextFrame1")
	}
}

func (source *MultiFramedRTPSource) processSpecialHeader(packet *BufferedPacket) (uint, bool) {
	return source.h264VideoRTPSource.processSpecialHeader(packet)
}

func (source *MultiFramedRTPSource) packetIsUsableInJitterCalculation(packet []byte, packetSize uint) bool {
	return true
}

func (source *MultiFramedRTPSource) networkReadHandler() {
	for {
		var packet *BufferedPacket = source.packetReadInProgress
		if packet == nil {
			packet = source.reOrderingBuffer.getFreePacket()
		}

		//var readSuccess bool
		for {
			err := packet.fillInData(source.rtpInterface)
			if err != nil {
				break
			}

			data := packet.data()
			size := packet.dataSize()

			// Check for the 12-byte RTP header:
			if size < 12 {
				break
			}

			rtpHdr, _ := ntohl(data)
			data, size = ADVANCE(data, size, 4)

			var rtpMarkerBit bool
			rtpMarkerBit = (rtpHdr & 0x00800000) != 0

			rtpTimestamp, _ := ntohl(data)
			data, size = ADVANCE(data, size, 4)

			rtpSSRC, _ := ntohl(data)
			data, size = ADVANCE(data, size, 4)

			// Check the RTP version number (it should be 2):
			if (rtpHdr & 0xC0000000) != 0x80000000 {
				fmt.Println("failed to check the RTP version number.")
				break
			}

			// Skip over any CSRC identifiers in the header:
			cc := (rtpHdr >> 24) & 0xF

			if size < uint(cc) {
				fmt.Println("error CSRC identifiers size in the header.")
				break
			}
			data, size = ADVANCE(data, size, 4)

			// Check for (& ignore) any RTP header extension
			if rtpHdr&0x10000000 != 0 {
				if size < 4 {
					break
				}

				extHdr, _ := ntohl(data)
				data, size = ADVANCE(data, size, 4)

				remExtSize := 4 * (extHdr & 0xFFFF)

				if size < uint(remExtSize) {
					fmt.Println("error RTP header extension size.")
					break
				}

				data, size = ADVANCE(data, size, 4)
			}

			// Discard any padding bytes:
			if rtpHdr&0x20000000 != 0 {
				if size == 0 {
					fmt.Println("The packet size equal zero.")
					break
				}
				numPaddingBytes := uint(data[size-1])
				if size < numPaddingBytes {
					fmt.Println("error padding bytes size.")
					break
				}
				packet.removePadding(numPaddingBytes)
			}

			// Check the Payload Type.
			rtpPayloadFormat := source.RTPPayloadFormat()
			if uint((rtpHdr&0x007F0000)>>16) != rtpPayloadFormat {
				fmt.Println("error RTP Payload format.")
				break
			}

			// The rest of the packet is the usable data.  Record and save it:
			if uint(rtpSSRC) != source.lastReceivedSSRC {
				source.lastReceivedSSRC = uint(rtpSSRC)
				source.reOrderingBuffer.resetHaveSeenFirstPacket()
			}

			rtpSeqNo := rtpHdr & 0xFFFF

			usableInJitterCalculation := source.packetIsUsableInJitterCalculation(data, size)

			var presentationTime utils.Timeval
			var hasBeenSyncedUsingRTCP bool

			timestampFrequency := uint32(source.TimestampFrequency())
			source.ReceptionStatsDB().noteIncomingPacket(rtpSSRC, rtpSeqNo, rtpTimestamp,
				timestampFrequency, uint32(size), usableInJitterCalculation)

			// Fill in the rest of the packet descriptor, and store it:
			var timeNow utils.Timeval
			utils.GetTimeOfDay(&timeNow)
			packet.assignMiscParams(rtpSeqNo, rtpTimestamp, presentationTime, timeNow,
				hasBeenSyncedUsingRTCP, rtpMarkerBit)

			if !source.reOrderingBuffer.storePacket(packet) {
				fmt.Println("failed to store packet.")
				break
			}

			//readSuccess = true
			break
		}

		source.doGetNextFrame1()
	}
}

////////// BufferedPacket definition //////////

const MAX_PACKET_SIZE = 20000

type BufferedPacket struct {
	buffer                 []byte
	head                   uint
	tail                   uint
	useCount               uint
	packetSize             uint
	RTPSeqNo               uint32
	RTPTimestamp           uint32
	nextPacket             *BufferedPacket
	timeReceived           utils.Timeval
	presentationTime       utils.Timeval
	hasBeenSyncedUsingRTCP bool
	RTPMarkerBit           bool
	isFirstPacket          bool
}

func NewBufferedPacket() *BufferedPacket {
	packet := new(BufferedPacket)
	packet.packetSize = MAX_PACKET_SIZE
	packet.buffer = make([]byte, MAX_PACKET_SIZE)
	return packet
}

func (packet *BufferedPacket) use(buff []byte, size uint) {
	frameSize := packet.tail - packet.head

	var bytesUsed, bytesTruncated uint
	var frameDurationInMicroseconds int64

	if frameSize > size {
		bytesTruncated += frameSize - size
		bytesUsed = size
	} else {
		bytesTruncated = 0
		bytesUsed = frameSize
	}

	packet.useCount += 1

	// Update "fPresentationTime" for the next enclosed frame (if any):
	packet.presentationTime.Tv_usec += frameDurationInMicroseconds
	if packet.presentationTime.Tv_usec >= 1000000 {
		packet.presentationTime.Tv_sec += packet.presentationTime.Tv_usec / 1000000
		packet.presentationTime.Tv_usec = packet.presentationTime.Tv_usec % 1000000
	}
}

func (packet *BufferedPacket) skip(numBytes uint) {
	packet.head += numBytes
	if packet.head > packet.tail {
		packet.head = packet.tail
	}
}

func (packet *BufferedPacket) rtpSeqNo() uint {
	return uint(packet.RTPSeqNo)
}

func (packet *BufferedPacket) UseCount() uint {
	return packet.useCount
}

func (packet *BufferedPacket) hasUsableData() bool {
	return (packet.tail - packet.head) != 0
}

func (packet *BufferedPacket) IsFirstPacket() bool {
	return packet.isFirstPacket
}

func (packet *BufferedPacket) NextPacket() *BufferedPacket {
	return packet.nextPacket
}

func (packet *BufferedPacket) removePadding(numBytes uint) {
	if numBytes > packet.tail-packet.head {
		numBytes = packet.tail - packet.head
	}

	packet.tail -= numBytes
}

func (packet *BufferedPacket) TimeReceived() utils.Timeval {
	return packet.timeReceived
}

func (packet *BufferedPacket) assignMiscParams(rtpSeqNo, rtpTimestamp uint32,
	presentationTime, timeReceived utils.Timeval, hasBeenSyncedUsingRTCP, rtpMarkerBit bool) {
	packet.RTPSeqNo = rtpSeqNo
	packet.timeReceived = timeReceived
	packet.RTPMarkerBit = rtpMarkerBit
	packet.RTPTimestamp = rtpTimestamp
	packet.presentationTime = presentationTime
	packet.hasBeenSyncedUsingRTCP = hasBeenSyncedUsingRTCP
}

func (packet *BufferedPacket) fillInData(rtpInterface *RTPInterface) error {
	readBytes, err := rtpInterface.handleRead(packet.buffer[packet.tail:])

	packet.tail += uint(readBytes)
	return err
}

func (packet *BufferedPacket) dataSize() uint {
	return packet.tail - packet.head
}

func (packet *BufferedPacket) data() []byte {
	return packet.buffer[:packet.head]
}

////////// BufferedPacketFactory definition //////////

type BufferedPacketFactory struct {
}

func NewBufferedPacketFactory() *BufferedPacketFactory {
	return new(BufferedPacketFactory)
}

func (factory *BufferedPacketFactory) createNewPacket() *BufferedPacket {
	return NewBufferedPacket()
}

////////// ReorderingPacketBuffer definition //////////

type ReorderingPacketBuffer struct {
	headPacket          *BufferedPacket
	tailPacket          *BufferedPacket
	savePacket          *BufferedPacket
	packetFactory       *BufferedPacketFactory
	thresholdTime       int64 // uSeconds
	savedPacketFree     bool
	haveSeenFirstPacket bool
	nextExpectedSeqNo   uint
}

func NewReorderingPacketBuffer(packetFactory *BufferedPacketFactory) *ReorderingPacketBuffer {
	packetBuffer := new(ReorderingPacketBuffer)
	packetBuffer.thresholdTime = 100000 /* default reordering threshold: 100 ms */
	if packetFactory == nil {
		packetBuffer.packetFactory = NewBufferedPacketFactory()
	} else {
		packetBuffer.packetFactory = packetFactory
	}
	return packetBuffer
}

func (buffer *ReorderingPacketBuffer) getFreePacket() *BufferedPacket {
	if buffer.savePacket == nil {
		buffer.savePacket = buffer.packetFactory.createNewPacket()
		buffer.savedPacketFree = true
	}

	if buffer.savedPacketFree {
		buffer.savedPacketFree = false
		return buffer.savePacket
	} else {
		return buffer.packetFactory.createNewPacket()
	}
}

func (buffer *ReorderingPacketBuffer) getNextCompletedPacket() (*BufferedPacket, bool) {
	var packetLossPreceded bool

	if buffer.headPacket == nil {
		return nil, packetLossPreceded
	}

	if buffer.headPacket.rtpSeqNo() == buffer.nextExpectedSeqNo {
		packetLossPreceded = buffer.headPacket.IsFirstPacket()
		return buffer.headPacket, packetLossPreceded
	}

	var timeThresholdHasBeenExceeded bool
	if buffer.thresholdTime == 0 {
		timeThresholdHasBeenExceeded = true
	} else {
		var timeNow utils.Timeval
		utils.GetTimeOfDay(&timeNow)

		timeReceived := buffer.headPacket.TimeReceived()
		uSecondsSinceReceived := (timeNow.Tv_sec-timeReceived.Tv_sec)*1000000 +
			(timeNow.Tv_usec - timeReceived.Tv_usec)
		timeThresholdHasBeenExceeded = uSecondsSinceReceived > buffer.thresholdTime
	}

	if timeThresholdHasBeenExceeded {
		buffer.nextExpectedSeqNo = buffer.headPacket.rtpSeqNo()
		// we've given up on earlier packets now
		packetLossPreceded = true
		return buffer.headPacket, packetLossPreceded
	}

	return nil, packetLossPreceded
}

func (buffer *ReorderingPacketBuffer) releaseUsedPacket(packet *BufferedPacket) {
	buffer.nextExpectedSeqNo++

	buffer.headPacket = buffer.headPacket.NextPacket()
	if buffer.headPacket != nil {
		//buffer.tailPacket = nil
	}
}

func (buffer *ReorderingPacketBuffer) resetHaveSeenFirstPacket() {
	buffer.haveSeenFirstPacket = false
}

func (buffer *ReorderingPacketBuffer) storePacket(packet *BufferedPacket) bool {
	rtpSeqNo := packet.rtpSeqNo()

	fmt.Println("ReorderingPacketBuffer::storePacket", rtpSeqNo)

	if !buffer.haveSeenFirstPacket {
		buffer.nextExpectedSeqNo = rtpSeqNo
		packet.isFirstPacket = true
		buffer.haveSeenFirstPacket = true
		fmt.Println("IsFirstPacket")
	}

	if seqNumLT(int(rtpSeqNo), int(buffer.nextExpectedSeqNo)) {
		fmt.Println("seqNumLT")
		return false
	}

	if buffer.tailPacket == nil {
		packet.nextPacket = nil
		buffer.headPacket = packet
		buffer.tailPacket = packet
	}

	tailPacketRTPSeqNo := int(buffer.tailPacket.rtpSeqNo())

	if seqNumLT(tailPacketRTPSeqNo, int(rtpSeqNo)) {
		packet.nextPacket = nil
		buffer.tailPacket.nextPacket = packet
		buffer.tailPacket = packet
	}

	if int(rtpSeqNo) == tailPacketRTPSeqNo {
		return false
	}

	var beforePtr *BufferedPacket
	var afterPtr *BufferedPacket = buffer.headPacket

	for afterPtr != nil {
		if seqNumLT(int(rtpSeqNo), int(afterPtr.rtpSeqNo())) {
			break
		}

		if rtpSeqNo == afterPtr.rtpSeqNo() {
			return false
		}

		beforePtr = afterPtr
		afterPtr = afterPtr.NextPacket()
	}

	packet.nextPacket = afterPtr
	if beforePtr == nil {
		buffer.headPacket = packet
	} else {
		beforePtr.nextPacket = packet
	}

	return true
}
