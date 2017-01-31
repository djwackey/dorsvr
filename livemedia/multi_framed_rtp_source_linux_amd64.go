package livemedia

import (
	"fmt"
	sys "syscall"

	gs "github.com/djwackey/dorsvr/groupsock"
)

type MultiFramedRTPSource struct {
	RTPSource
	needDelivery                bool
	areDoingNetworkReads        bool
	currentPacketBeginsFrame    bool
	currentPacketCompletesFrame bool
	packetLossInFragmentedFrame bool
	packetReadInProgress        IBufferedPacket
	reOrderingBuffer            *ReorderingPacketBuffer
	specialHeaderHandler        interface{}
	videoRTPSource              interface{}
}

func (source *MultiFramedRTPSource) InitMultiFramedRTPSource(isource IFramedSource,
	RTPgs *gs.GroupSock, rtpPayloadFormat, rtpTimestampFrequency uint,
	packetFactory IBufferedPacketFactory) {

	source.reset()

	source.videoRTPSource = isource
	source.reOrderingBuffer = NewReorderingPacketBuffer(packetFactory)
	source.InitRTPSouce(isource, RTPgs, rtpPayloadFormat, rtpTimestampFrequency)
}

func (source *MultiFramedRTPSource) reset() {
	source.packetLossInFragmentedFrame = false
	source.currentPacketCompletesFrame = true
	source.currentPacketBeginsFrame = true
	source.areDoingNetworkReads = false
	source.needDelivery = false
}

func (source *MultiFramedRTPSource) doGetNextFrame() {
	source.rtpInterface.startNetworkReading(source.networkReadHandler)

	source.frameSize = 0
}

func (source *MultiFramedRTPSource) doGetNextFrame1() {
	source.needDelivery = true
	for source.needDelivery {
		var packetLossPrecededThis bool
		var nextPacket IBufferedPacket

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
			fmt.Println("Packet Loss In Fragmented Frame.")
			break
		}

		packetInfo := nextPacket.use(source.buffTo, source.maxSize)
		source.presentationTime = packetInfo.presentationTime
		source.numTruncatedBytes = packetInfo.bytesTruncated
		source.curPacketRTPTimestamp = packetInfo.rtpTimestamp
		source.curPacketMarkerBit = packetInfo.rtpMarkerBit
		source.curPacketRTPSeqNum = packetInfo.rtpSeqNo
		source.curPacketSyncUsingRTCP = packetInfo.hasBeenSyncedUsingRTCP

		source.frameSize += packetInfo.bytesUsed

		if !nextPacket.hasUsableData() {
			source.reOrderingBuffer.releaseUsedPacket(nextPacket)
			fmt.Println("reOrderingBuffer release Used Packet.")
		}

		if source.currentPacketCompletesFrame {
			source.afterGetting()
		} else {
			source.needDelivery = true
		}
	}
}

func (source *MultiFramedRTPSource) setSpecialHeaderHandler(handler interface{}) {
	source.specialHeaderHandler = handler
}

func (source *MultiFramedRTPSource) processSpecialHeader(packet IBufferedPacket) (uint, bool) {
	if source.specialHeaderHandler != nil {
		return source.specialHeaderHandler.(func(packet IBufferedPacket) (uint, bool))(packet)
	}
	return 0, true
}

func (source *MultiFramedRTPSource) packetIsUsableInJitterCalculation(packet []byte, packetSize uint) bool {
	return true
}

func (source *MultiFramedRTPSource) networkReadHandler() {
	for {
		var packet IBufferedPacket = source.packetReadInProgress
		if packet == nil {
			packet = source.reOrderingBuffer.getFreePacket(source)
		}

		for {
			err := packet.fillInData(source.rtpInterface)
			if err != nil {
				break
			}

			// Check for the 12-byte RTP header:
			if packet.dataSize() < 12 {
				break
			}

			rtpHdr, _ := gs.Ntohl(packet.data())
			packet.skip(4)

			var rtpMarkerBit bool = (rtpHdr & 0x00800000) != 0

			rtpTimestamp, _ := gs.Ntohl(packet.data())
			packet.skip(4)

			rtpSSRC, _ := gs.Ntohl(packet.data())
			packet.skip(4)

			// Check the RTP version number (it should be 2):
			if (rtpHdr & 0xC0000000) != 0x80000000 {
				fmt.Println("failed to check the RTP version number.")
				break
			}

			// Skip over any CSRC identifiers in the header:
			cc := uint((rtpHdr >> 24) & 0xF)

			if packet.dataSize() < cc {
				fmt.Println("error CSRC identifiers size in the header.")
				break
			}
			packet.skip(cc * 4)

			// Check for (& ignore) any RTP header extension
			if rtpHdr&0x10000000 != 0 {
				if packet.dataSize() < 4 {
					break
				}

				extHdr, _ := gs.Ntohl(packet.data())
				packet.skip(4)

				remExtSize := uint(4 * (extHdr & 0xFFFF))

				if packet.dataSize() < remExtSize {
					fmt.Println("error RTP header extension size.")
					break
				}

				packet.skip(remExtSize)
			}

			// Discard any padding bytes:
			if rtpHdr&0x20000000 != 0 {
				if packet.dataSize() == 0 {
					fmt.Println("The packet size equal zero.")
					break
				}
				numPaddingBytes := uint(packet.data()[packet.dataSize()-1])
				if packet.dataSize() < numPaddingBytes {
					fmt.Println("error padding bytes size.")
					break
				}
				packet.removePadding(numPaddingBytes)
			}

			// Check the Payload Type.
			if uint((rtpHdr&0x007F0000)>>16) != source.rtpPayloadFormat {
				fmt.Println("error RTP Payload format.")
				break
			}

			// The rest of the packet is the usable data.  Record and save it:
			if uint(rtpSSRC) != source.lastReceivedSSRC {
				source.lastReceivedSSRC = uint(rtpSSRC)
				source.reOrderingBuffer.resetHaveSeenFirstPacket()
			}

			rtpSeqNo := rtpHdr & 0xFFFF

			usableInJitterCalculation := source.packetIsUsableInJitterCalculation(packet.data(), packet.dataSize())

			presentationTime, hasBeenSyncedUsingRTCP :=
				source.receptionStatsDB.noteIncomingPacket(rtpSSRC, rtpSeqNo, rtpTimestamp,
					uint32(source.timestampFrequency), uint32(packet.dataSize()), usableInJitterCalculation)

			// Fill in the rest of the packet descriptor, and store it:
			var timeNow sys.Timeval
			sys.Gettimeofday(&timeNow)
			packet.assignMiscParams(rtpSeqNo, rtpTimestamp, presentationTime, timeNow,
				hasBeenSyncedUsingRTCP, rtpMarkerBit)

			if !source.reOrderingBuffer.storePacket(packet) {
				fmt.Println("failed to store packet.")
				break
			}

			break
		}

		source.doGetNextFrame1()
	}
}

////////// BufferedPacket definition //////////

const MAX_PACKET_SIZE = 20000

type IBufferedPacket interface {
	data() []byte
	dataSize() uint
	rtpSeqNo() uint
	UseCount() uint
	skip(numBytes uint)
	isFirstPacket() bool
	hasUsableData() bool
	markFirstPacket(flag bool)
	removePadding(numBytes uint)
	TimeReceived() sys.Timeval
	NextPacket() IBufferedPacket
	use(buff []byte, size uint) *PacketInfo
	setNextPacket(nextPacket IBufferedPacket)
	fillInData(rtpInterface *RTPInterface) error
	assignMiscParams(rtpSeqNo, rtpTimestamp uint32,
		presentationTime, timeReceived sys.Timeval,
		hasBeenSyncedUsingRTCP, rtpMarkerBit bool)
}

type BufferedPacket struct {
	buffer                 []byte
	head                   uint
	tail                   uint
	useCount               uint
	packetSize             uint
	RTPSeqNo               uint32
	RTPTimestamp           uint32
	nextPacket             IBufferedPacket
	timeReceived           sys.Timeval
	presentationTime       sys.Timeval
	nextEnclosedFrameProc  interface{}
	hasBeenSyncedUsingRTCP bool
	firstPacketFlag        bool
	RTPMarkerBit           bool
}

type PacketInfo struct {
	bytesUsed              uint
	bytesTruncated         uint
	rtpSeqNo               uint32
	rtpTimestamp           uint32
	presentationTime       sys.Timeval
	hasBeenSyncedUsingRTCP bool
	rtpMarkerBit           bool
}

func NewBufferedPacket() *BufferedPacket {
	packet := new(BufferedPacket)
	packet.InitBufferedPacket()
	return packet
}

func (packet *BufferedPacket) InitBufferedPacket() {
	packet.packetSize = MAX_PACKET_SIZE
	packet.buffer = make([]byte, MAX_PACKET_SIZE)
}

func (packet *BufferedPacket) use(buff []byte, size uint) *PacketInfo {
	origFramePtr, dataSize := packet.data(), packet.dataSize()

	var frameSize, frameDurationInMicroseconds uint
	frameSize, frameDurationInMicroseconds = packet.getNextEnclosedFrameParameters(origFramePtr, dataSize)

	var bytesUsed, bytesTruncated uint
	if frameSize > size {
		bytesTruncated += frameSize - size
		bytesUsed = size
	} else {
		bytesTruncated = 0
		bytesUsed = frameSize
	}

	packet.useCount += 1

	packetInfo := new(PacketInfo)
	packetInfo.bytesUsed = bytesUsed
	packetInfo.rtpSeqNo = packet.RTPSeqNo
	packetInfo.rtpTimestamp = packet.RTPTimestamp
	packetInfo.bytesTruncated = bytesTruncated
	packetInfo.presentationTime = packet.presentationTime
	packetInfo.hasBeenSyncedUsingRTCP = packet.hasBeenSyncedUsingRTCP
	packetInfo.rtpMarkerBit = packet.RTPMarkerBit

	// Update "presentationTime" for the next enclosed frame (if any):
	packet.presentationTime.Usec += int64(frameDurationInMicroseconds)
	if packet.presentationTime.Usec >= 1000000 {
		packet.presentationTime.Sec += packet.presentationTime.Usec / 1000000
		packet.presentationTime.Usec = packet.presentationTime.Usec % 1000000
	}

	return packetInfo
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

func (packet *BufferedPacket) setNextPacket(nextPacket IBufferedPacket) {
	packet.nextPacket = nextPacket
}

func (packet *BufferedPacket) hasUsableData() bool {
	return (packet.tail - packet.head) != 0
}

func (packet *BufferedPacket) isFirstPacket() bool {
	return packet.firstPacketFlag
}

func (packet *BufferedPacket) markFirstPacket(flag bool) {
	packet.firstPacketFlag = flag
}

func (packet *BufferedPacket) NextPacket() IBufferedPacket {
	return packet.nextPacket
}

func (packet *BufferedPacket) removePadding(numBytes uint) {
	if numBytes > packet.tail-packet.head {
		numBytes = packet.tail - packet.head
	}

	packet.tail -= numBytes
}

func (packet *BufferedPacket) TimeReceived() sys.Timeval {
	return packet.timeReceived
}

func (packet *BufferedPacket) nextEnclosedFrameSize(buff []byte, size uint) uint {
	if packet.nextEnclosedFrameProc != nil {
		packet.nextEnclosedFrameProc.(func(buff []byte, size uint) uint)(buff, size)
	}
	return size
}

func (packet *BufferedPacket) getNextEnclosedFrameParameters(framePtr []byte, dataSize uint) (frameSize,
	frameDurationInMicroseconds uint) {
	frameSize = packet.nextEnclosedFrameSize(framePtr, dataSize)
	frameDurationInMicroseconds = 0
	return
}

func (packet *BufferedPacket) assignMiscParams(rtpSeqNo, rtpTimestamp uint32,
	presentationTime, timeReceived sys.Timeval, hasBeenSyncedUsingRTCP, rtpMarkerBit bool) {
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
	return packet.buffer[packet.head:]
}

////////// BufferedPacketFactory definition //////////

type IBufferedPacketFactory interface {
	createNewPacket(source interface{}) IBufferedPacket
}

type BufferedPacketFactory struct {
}

func NewBufferedPacketFactory() IBufferedPacketFactory {
	return new(BufferedPacketFactory)
}

func (factory *BufferedPacketFactory) createNewPacket(source interface{}) IBufferedPacket {
	return NewBufferedPacket()
}

////////// ReorderingPacketBuffer definition //////////

type ReorderingPacketBuffer struct {
	headPacket          IBufferedPacket
	tailPacket          IBufferedPacket
	savePacket          IBufferedPacket
	packetFactory       IBufferedPacketFactory
	thresholdTime       int64 // uSeconds
	savedPacketFree     bool
	haveSeenFirstPacket bool
	nextExpectedSeqNo   uint
}

func NewReorderingPacketBuffer(packetFactory IBufferedPacketFactory) *ReorderingPacketBuffer {
	packetBuffer := new(ReorderingPacketBuffer)
	packetBuffer.thresholdTime = 100000 /* default reordering threshold: 100 ms */
	if packetFactory == nil {
		packetBuffer.packetFactory = NewBufferedPacketFactory()
	} else {
		packetBuffer.packetFactory = packetFactory
	}
	return packetBuffer
}

func (buffer *ReorderingPacketBuffer) getFreePacket(source *MultiFramedRTPSource) IBufferedPacket {
	if buffer.savePacket == nil {
		buffer.savePacket = buffer.packetFactory.createNewPacket(source.videoRTPSource)
		buffer.savedPacketFree = true
	}

	if buffer.savedPacketFree {
		buffer.savedPacketFree = false
		return buffer.savePacket
	} else {
		return buffer.packetFactory.createNewPacket(source.videoRTPSource)
	}
}

func (buffer *ReorderingPacketBuffer) getNextCompletedPacket() (IBufferedPacket, bool) {
	var packetLossPreceded bool

	if buffer.headPacket == nil {
		fmt.Println("ReorderingPacketBuffer::getNextCompletedPacket: buffer head packet equal nil")
		return nil, packetLossPreceded
	}

	if buffer.headPacket.rtpSeqNo() == buffer.nextExpectedSeqNo {
		packetLossPreceded = buffer.headPacket.isFirstPacket()
		return buffer.headPacket, packetLossPreceded
	}

	var timeThresholdHasBeenExceeded bool
	if buffer.thresholdTime == 0 {
		timeThresholdHasBeenExceeded = true
	} else {
		var timeNow sys.Timeval
		sys.Gettimeofday(&timeNow)

		timeReceived := buffer.headPacket.TimeReceived()
		uSecondsSinceReceived := (timeNow.Sec-timeReceived.Sec)*1000000 +
			(timeNow.Usec - timeReceived.Usec)
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

func (buffer *ReorderingPacketBuffer) releaseUsedPacket(packet IBufferedPacket) {
	buffer.nextExpectedSeqNo++

	buffer.headPacket = buffer.headPacket.NextPacket()
	if buffer.headPacket != nil {
		buffer.tailPacket = nil
	}
	packet.setNextPacket(nil)
}

func (buffer *ReorderingPacketBuffer) resetHaveSeenFirstPacket() {
	buffer.haveSeenFirstPacket = false
}

func (buffer *ReorderingPacketBuffer) storePacket(packet IBufferedPacket) bool {
	rtpSeqNo := packet.rtpSeqNo()

	if !buffer.haveSeenFirstPacket {
		buffer.nextExpectedSeqNo = rtpSeqNo
		packet.markFirstPacket(true)
		buffer.haveSeenFirstPacket = true
		fmt.Println("IsFirstPacket")
	}

	if seqNumLT(int(rtpSeqNo), int(buffer.nextExpectedSeqNo)) {
		fmt.Println("seqNumLT")
		return false
	}

	if buffer.tailPacket == nil {
		packet.setNextPacket(nil)
		buffer.headPacket = packet
		buffer.tailPacket = packet
		return true
	}

	tailPacketRTPSeqNo := int(buffer.tailPacket.rtpSeqNo())

	if seqNumLT(tailPacketRTPSeqNo, int(rtpSeqNo)) {
		packet.setNextPacket(nil)
		buffer.tailPacket.setNextPacket(packet)
		buffer.tailPacket = packet
		return true
	}

	if int(rtpSeqNo) == tailPacketRTPSeqNo {
		fmt.Printf("rtpSeqNo[%d] unequal to tailPacketRTPSeqNo[%d]\n", rtpSeqNo, tailPacketRTPSeqNo)
		return false
	}

	var beforePtr IBufferedPacket
	var afterPtr IBufferedPacket = buffer.headPacket

	for afterPtr != nil {
		if seqNumLT(int(rtpSeqNo), int(afterPtr.rtpSeqNo())) {
			fmt.Println("afterPtr seqNumLT")
			break
		}

		if rtpSeqNo == afterPtr.rtpSeqNo() {
			fmt.Println("This is a duplicate packet - ignore it", rtpSeqNo, afterPtr.rtpSeqNo())
			return false
		}

		beforePtr = afterPtr
		afterPtr = afterPtr.NextPacket()
	}

	packet.setNextPacket(afterPtr)
	if beforePtr == nil {
		buffer.headPacket = packet
	} else {
		beforePtr.setNextPacket(packet)
	}

	return true
}
