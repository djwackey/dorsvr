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

func (s *MultiFramedRTPSource) initMultiFramedRTPSource(source IFramedSource,
	RTPgs *gs.GroupSock, rtpPayloadFormat, rtpTimestampFrequency uint32,
	packetFactory IBufferedPacketFactory) {

	s.reset()

	s.videoRTPSource = source
	s.reOrderingBuffer = newReorderingPacketBuffer(packetFactory)
	s.initRTPSouce(source, RTPgs, rtpPayloadFormat, rtpTimestampFrequency)
}

func (s *MultiFramedRTPSource) reset() {
	s.packetLossInFragmentedFrame = false
	s.currentPacketCompletesFrame = true
	s.currentPacketBeginsFrame = true
	s.areDoingNetworkReads = false
	s.needDelivery = false
}

func (s *MultiFramedRTPSource) doGetNextFrame() error {
	s.rtpInterface.startNetworkReading(s.networkReadHandler)

	s.frameSize = 0
	return nil
}

func (s *MultiFramedRTPSource) doGetNextFrame1() {
	s.needDelivery = true
	for s.needDelivery {
		var packetLossPrecededThis bool
		var nextPacket IBufferedPacket

		nextPacket, packetLossPrecededThis = s.reOrderingBuffer.getNextCompletedPacket()
		if nextPacket == nil {
			fmt.Println("failed to get next completed packet.")
			break
		}

		s.needDelivery = false

		if nextPacket.UseCount() == 0 {
			// Before using the packet, check whether it has a special header
			// that needs to be processed:
			specialHeaderSize, processOK := s.processSpecialHeader(nextPacket)
			if !processOK {
				// Something's wrong with the header; reject the packet:
				s.reOrderingBuffer.releaseUsedPacket(nextPacket)
				s.needDelivery = true

				fmt.Println("failed to process special header.")
				break
			}
			nextPacket.skip(specialHeaderSize)
		}

		if s.currentPacketBeginsFrame {
			s.packetLossInFragmentedFrame = false
		} else if packetLossPrecededThis {
			s.packetLossInFragmentedFrame = true
		}

		if s.packetLossInFragmentedFrame {
			s.reOrderingBuffer.releaseUsedPacket(nextPacket)
			s.needDelivery = true
			fmt.Println("Packet Loss In Fragmented Frame.")
			break
		}

		packetInfo := nextPacket.use(s.buffTo, uint32(s.maxSize))
		s.presentationTime = packetInfo.presentationTime
		s.numTruncatedBytes = uint(packetInfo.bytesTruncated)
		s.curPacketRTPTimestamp = packetInfo.rtpTimestamp
		s.curPacketMarkerBit = packetInfo.rtpMarkerBit
		s.curPacketRTPSeqNum = packetInfo.rtpSeqNo
		s.curPacketSyncUsingRTCP = packetInfo.hasBeenSyncedUsingRTCP

		s.frameSize += uint(packetInfo.bytesUsed)

		if !nextPacket.hasUsableData() {
			s.reOrderingBuffer.releaseUsedPacket(nextPacket)
			fmt.Println("reOrderingBuffer release Used Packet.")
		}

		if s.currentPacketCompletesFrame {
			s.afterGetting()
		} else {
			s.needDelivery = true
		}
	}
}

func (s *MultiFramedRTPSource) setSpecialHeaderHandler(handler interface{}) {
	s.specialHeaderHandler = handler
}

func (s *MultiFramedRTPSource) processSpecialHeader(packet IBufferedPacket) (uint32, bool) {
	if s.specialHeaderHandler != nil {
		return s.specialHeaderHandler.(func(packet IBufferedPacket) (uint32, bool))(packet)
	}
	return 0, true
}

func (s *MultiFramedRTPSource) packetIsUsableInJitterCalculation(packet []byte, packetSize uint32) bool {
	return true
}

func (s *MultiFramedRTPSource) networkReadHandler() {
	for {
		var packet IBufferedPacket = s.packetReadInProgress
		if packet == nil {
			packet = s.reOrderingBuffer.getFreePacket(s)
		}

		for {
			err := packet.fillInData(s.rtpInterface)
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
			cc := (rtpHdr >> 24) & 0xF

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

				remExtSize := 4 * (extHdr & 0xFFFF)

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
				numPaddingBytes := uint32(packet.data()[packet.dataSize()-1])
				if packet.dataSize() < numPaddingBytes {
					fmt.Println("error padding bytes size.")
					break
				}
				packet.removePadding(numPaddingBytes)
			}

			// Check the Payload Type.
			if (rtpHdr&0x007F0000)>>16 != s.rtpPayloadFormat {
				fmt.Println("error RTP Payload format.")
				break
			}

			// The rest of the packet is the usable data.  Record and save it:
			if rtpSSRC != s.lastReceivedSSRC {
				s.lastReceivedSSRC = rtpSSRC
				s.reOrderingBuffer.resetHaveSeenFirstPacket()
			}

			rtpSeqNo := rtpHdr & 0xFFFF

			usableInJitterCalculation := s.packetIsUsableInJitterCalculation(packet.data(), packet.dataSize())

			presentationTime, hasBeenSyncedUsingRTCP :=
				s.receptionStatsDB.noteIncomingPacket(rtpSSRC, rtpSeqNo, rtpTimestamp,
					uint32(s.timestampFrequency), uint32(packet.dataSize()), usableInJitterCalculation)

			// Fill in the rest of the packet descriptor, and store it:
			var timeNow sys.Timeval
			sys.Gettimeofday(&timeNow)
			packet.assignMiscParams(rtpSeqNo, rtpTimestamp, presentationTime, timeNow,
				hasBeenSyncedUsingRTCP, rtpMarkerBit)

			if !s.reOrderingBuffer.storePacket(packet) {
				fmt.Println("failed to store packet.")
				break
			}

			break
		}

		s.doGetNextFrame1()
	}
}

////////// BufferedPacket definition //////////

const maxPacketSize = 20000

type IBufferedPacket interface {
	data() []byte
	dataSize() uint32
	rtpSeqNo() uint
	UseCount() uint
	skip(numBytes uint32)
	isFirstPacket() bool
	hasUsableData() bool
	markFirstPacket(flag bool)
	removePadding(numBytes uint32)
	TimeReceived() sys.Timeval
	NextPacket() IBufferedPacket
	use(buff []byte, size uint32) *PacketInfo
	setNextPacket(nextPacket IBufferedPacket)
	fillInData(rtpInterface *RTPInterface) error
	assignMiscParams(rtpSeqNo, rtpTimestamp uint32,
		presentationTime, timeReceived sys.Timeval,
		hasBeenSyncedUsingRTCP, rtpMarkerBit bool)
}

type BufferedPacket struct {
	buffer                 []byte
	head                   uint32
	tail                   uint32
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
	bytesUsed              uint32
	bytesTruncated         uint32
	rtpSeqNo               uint32
	rtpTimestamp           uint32
	presentationTime       sys.Timeval
	hasBeenSyncedUsingRTCP bool
	rtpMarkerBit           bool
}

func newBufferedPacket() *BufferedPacket {
	p := new(BufferedPacket)
	p.initBufferedPacket()
	return p
}

func (p *BufferedPacket) initBufferedPacket() {
	p.packetSize = maxPacketSize
	p.buffer = make([]byte, maxPacketSize)
}

func (p *BufferedPacket) use(buff []byte, size uint32) (info *PacketInfo) {
	origFramePtr, dataSize := p.data(), p.dataSize()

	var frameSize, frameDurationInMicroseconds uint32
	frameSize, frameDurationInMicroseconds = p.getNextEnclosedFrameParameters(origFramePtr, dataSize)

	var bytesUsed, bytesTruncated uint32
	if frameSize > size {
		bytesTruncated += frameSize - size
		bytesUsed = size
	} else {
		bytesTruncated = 0
		bytesUsed = frameSize
	}

	p.useCount += 1

	info = &PacketInfo{
		bytesUsed:              bytesUsed,
		rtpSeqNo:               p.RTPSeqNo,
		rtpTimestamp:           p.RTPTimestamp,
		rtpMarkerBit:           p.RTPMarkerBit,
		bytesTruncated:         bytesTruncated,
		presentationTime:       p.presentationTime,
		hasBeenSyncedUsingRTCP: p.hasBeenSyncedUsingRTCP,
	}

	// Update "presentationTime" for the next enclosed frame (if any):
	p.presentationTime.Usec += int64(frameDurationInMicroseconds)
	if p.presentationTime.Usec >= 1000000 {
		p.presentationTime.Sec += p.presentationTime.Usec / 1000000
		p.presentationTime.Usec = p.presentationTime.Usec % 1000000
	}

	return info
}

func (p *BufferedPacket) skip(numBytes uint32) {
	p.head += numBytes
	if p.head > p.tail {
		p.head = p.tail
	}
}

func (p *BufferedPacket) rtpSeqNo() uint {
	return uint(p.RTPSeqNo)
}

func (p *BufferedPacket) UseCount() uint {
	return p.useCount
}

func (p *BufferedPacket) setNextPacket(nextPacket IBufferedPacket) {
	p.nextPacket = nextPacket
}

func (p *BufferedPacket) hasUsableData() bool {
	return (p.tail - p.head) != 0
}

func (p *BufferedPacket) isFirstPacket() bool {
	return p.firstPacketFlag
}

func (p *BufferedPacket) markFirstPacket(flag bool) {
	p.firstPacketFlag = flag
}

func (p *BufferedPacket) NextPacket() IBufferedPacket {
	return p.nextPacket
}

func (p *BufferedPacket) removePadding(numBytes uint32) {
	if numBytes > p.tail-p.head {
		numBytes = p.tail - p.head
	}

	p.tail -= numBytes
}

func (p *BufferedPacket) TimeReceived() sys.Timeval {
	return p.timeReceived
}

func (p *BufferedPacket) nextEnclosedFrameSize(buff []byte, size uint32) uint32 {
	if p.nextEnclosedFrameProc != nil {
		p.nextEnclosedFrameProc.(func(buff []byte, size uint32) uint32)(buff, size)
	}
	return size
}

func (p *BufferedPacket) getNextEnclosedFrameParameters(framePtr []byte, dataSize uint32) (frameSize,
	frameDurationInMicroseconds uint32) {
	frameSize = p.nextEnclosedFrameSize(framePtr, dataSize)
	frameDurationInMicroseconds = 0
	return
}

func (p *BufferedPacket) assignMiscParams(rtpSeqNo, rtpTimestamp uint32,
	presentationTime, timeReceived sys.Timeval, hasBeenSyncedUsingRTCP, rtpMarkerBit bool) {
	p.RTPSeqNo = rtpSeqNo
	p.timeReceived = timeReceived
	p.RTPMarkerBit = rtpMarkerBit
	p.RTPTimestamp = rtpTimestamp
	p.presentationTime = presentationTime
	p.hasBeenSyncedUsingRTCP = hasBeenSyncedUsingRTCP
}

func (p *BufferedPacket) fillInData(i *RTPInterface) error {
	readBytes, err := i.handleRead(p.buffer[p.tail:])

	p.tail += uint32(readBytes)
	return err
}

func (p *BufferedPacket) dataSize() uint32 {
	return p.tail - p.head
}

func (p *BufferedPacket) data() []byte {
	return p.buffer[p.head:]
}

////////// BufferedPacketFactory definition //////////

type IBufferedPacketFactory interface {
	createNewPacket(source interface{}) IBufferedPacket
}

type BufferedPacketFactory struct {
}

func newBufferedPacketFactory() IBufferedPacketFactory {
	return new(BufferedPacketFactory)
}

func (f *BufferedPacketFactory) createNewPacket(source interface{}) IBufferedPacket {
	return newBufferedPacket()
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

func newReorderingPacketBuffer(packetFactory IBufferedPacketFactory) *ReorderingPacketBuffer {
	packetBuffer := new(ReorderingPacketBuffer)
	packetBuffer.thresholdTime = 100000 /* default reordering threshold: 100 ms */
	if packetFactory == nil {
		packetBuffer.packetFactory = newBufferedPacketFactory()
	} else {
		packetBuffer.packetFactory = packetFactory
	}
	return packetBuffer
}

func (b *ReorderingPacketBuffer) getFreePacket(s *MultiFramedRTPSource) IBufferedPacket {
	if b.savePacket == nil {
		b.savePacket = b.packetFactory.createNewPacket(s.videoRTPSource)
		b.savedPacketFree = true
	}

	if b.savedPacketFree {
		b.savedPacketFree = false
		return b.savePacket
	} else {
		return b.packetFactory.createNewPacket(s.videoRTPSource)
	}
}

func (b *ReorderingPacketBuffer) getNextCompletedPacket() (IBufferedPacket, bool) {
	var packetLossPreceded bool

	if b.headPacket == nil {
		fmt.Println("ReorderingPacketBuffer::getNextCompletedPacket: buffer head packet equal nil")
		return nil, packetLossPreceded
	}

	if b.headPacket.rtpSeqNo() == b.nextExpectedSeqNo {
		packetLossPreceded = b.headPacket.isFirstPacket()
		return b.headPacket, packetLossPreceded
	}

	var timeThresholdHasBeenExceeded bool
	if b.thresholdTime == 0 {
		timeThresholdHasBeenExceeded = true
	} else {
		var timeNow sys.Timeval
		sys.Gettimeofday(&timeNow)

		timeReceived := b.headPacket.TimeReceived()
		uSecondsSinceReceived := (timeNow.Sec-timeReceived.Sec)*1000000 +
			(timeNow.Usec - timeReceived.Usec)
		timeThresholdHasBeenExceeded = uSecondsSinceReceived > b.thresholdTime
	}

	if timeThresholdHasBeenExceeded {
		b.nextExpectedSeqNo = b.headPacket.rtpSeqNo()
		// we've given up on earlier packets now
		packetLossPreceded = true
		return b.headPacket, packetLossPreceded
	}

	return nil, packetLossPreceded
}

func (b *ReorderingPacketBuffer) releaseUsedPacket(packet IBufferedPacket) {
	b.nextExpectedSeqNo++

	b.headPacket = b.headPacket.NextPacket()
	if b.headPacket != nil {
		b.tailPacket = nil
	}
	packet.setNextPacket(nil)
}

func (b *ReorderingPacketBuffer) resetHaveSeenFirstPacket() {
	b.haveSeenFirstPacket = false
}

func (b *ReorderingPacketBuffer) storePacket(packet IBufferedPacket) bool {
	rtpSeqNo := packet.rtpSeqNo()

	if !b.haveSeenFirstPacket {
		b.nextExpectedSeqNo = rtpSeqNo
		packet.markFirstPacket(true)
		b.haveSeenFirstPacket = true
		fmt.Println("IsFirstPacket")
	}

	if seqNumLT(int(rtpSeqNo), int(b.nextExpectedSeqNo)) {
		fmt.Println("seqNumLT")
		return false
	}

	if b.tailPacket == nil {
		packet.setNextPacket(nil)
		b.headPacket = packet
		b.tailPacket = packet
		return true
	}

	tailPacketRTPSeqNo := int(b.tailPacket.rtpSeqNo())

	if seqNumLT(tailPacketRTPSeqNo, int(rtpSeqNo)) {
		packet.setNextPacket(nil)
		b.tailPacket.setNextPacket(packet)
		b.tailPacket = packet
		return true
	}

	if int(rtpSeqNo) == tailPacketRTPSeqNo {
		fmt.Printf("rtpSeqNo[%d] unequal to tailPacketRTPSeqNo[%d]\n", rtpSeqNo, tailPacketRTPSeqNo)
		return false
	}

	var beforePtr IBufferedPacket
	var afterPtr IBufferedPacket = b.headPacket

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
		b.headPacket = packet
	} else {
		beforePtr.setNextPacket(packet)
	}

	return true
}
