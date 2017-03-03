package livemedia

import (
	"fmt"
	sys "syscall"

	gs "github.com/djwackey/dorsvr/groupsock"
)

const (
	// RTCP packet types:
	RTCP_PT_SR   = 200
	RTCP_PT_RR   = 201
	RTCP_PT_SDES = 202
	RTCP_PT_BYE  = 203
	RTCP_PT_APP  = 204

	// SDES tags:
	RTCP_SDES_END   = 0
	RTCP_SDES_CNAME = 1
	RTCP_SDES_NAME  = 2
	RTCP_SDES_EMAIL = 3
	RTCP_SDES_PHONE = 4
	RTCP_SDES_LOC   = 5
	RTCP_SDES_TOOL  = 6
	RTCP_SDES_NOTE  = 7
	RTCP_SDES_PRIV  = 8

	// overhead (bytes) of IP and UDP hdrs
	IP_UDP_HDR_SIZE = 28

	PACKET_UNKNOWN_TYPE = 0
	PAKCET_RTP          = 1
	PACKET_RTCP_REPORT  = 2
	PACKET_BYE          = 3
)

type SDESItem struct {
	data []byte
}

// bytes, (1500, minus some allowance for IP, UDP, UMTP headers)
var maxRTCPPacketSize uint = 1450
var preferredPacketSize uint = 1000 // bytes

type RTCPInstance struct {
	typeOfEvent          uint
	lastSentSize         uint
	totSessionBW         uint
	lastPacketSentSize   uint
	haveJustSentPacket   bool
	prevReportTime       int32
	nextReportTime       int32
	inBuf                []byte
	CNAME                *SDESItem
	Sink                 *RTPSink
	Source               *RTPSource
	outBuf               *OutPacketBuffer
	rtcpInterface        *RTPInterface
	byeHandlerTask       interface{}
	SRHandlerTask        interface{}
	RRHandlerTask        interface{}
	byeHandlerClientData interface{}
}

func newSDESItem(tag int, value string) *SDESItem {
	item := new(SDESItem)

	length := len(value)
	if length > 0xFF {
		length = 0xFF // maximum data length for a SDES item
	}

	item.data = []byte{byte(tag), byte(length)}
	return item
}

func dTimeNow() int32 {
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)
	return timeNow.Sec + timeNow.Usec/1000000.0
}

func (s *SDESItem) totalSize() uint {
	return 2 + uint(s.data[1])
}

func newRTCPInstance(rtcpGS *gs.GroupSock, totSessionBW uint, cname string) *RTCPInstance {
	rtcp := new(RTCPInstance)
	rtcp.typeOfEvent = EVENT_REPORT
	rtcp.totSessionBW = totSessionBW
	rtcp.CNAME = newSDESItem(RTCP_SDES_CNAME, cname)

	rtcp.prevReportTime = dTimeNow()
	rtcp.nextReportTime = rtcp.prevReportTime

	rtcp.inBuf = make([]byte, maxRTCPPacketSize)
	rtcp.outBuf = newOutPacketBuffer(preferredPacketSize, maxRTCPPacketSize)

	rtcp.rtcpInterface = newRTPInterface(rtcp, rtcpGS)
	rtcp.rtcpInterface.startNetworkReading(rtcp.incomingReportHandler)

	rtcp.onExpire()
	return rtcp
}

func (r *RTCPInstance) numMembers() uint {
	return 0
}

func (r *RTCPInstance) setSpecificRRHandler() {
}

func (r *RTCPInstance) SetByeHandler(handlerTask interface{}, clientData interface{}) {
	r.byeHandlerTask = handlerTask
	r.byeHandlerClientData = clientData
}

func (r *RTCPInstance) setSRHandler(handlerTask interface{}, clientData interface{}) {
	r.SRHandlerTask = handlerTask
}

func (r *RTCPInstance) setRRHandler(handlerTask interface{}, clientData interface{}) {
	r.RRHandlerTask = handlerTask
}

func (r *RTCPInstance) incomingReportHandler() {
	var callByeHandler bool
	for {
		readBytes, err := r.rtcpInterface.handleRead(r.inBuf)
		if err != nil {
			fmt.Println("RTCP Interface failed to handle read.", err.Error())
			break
		}

		packet := r.inBuf[:readBytes]
		packetSize := uint(readBytes)

		var rtcpHdr uint32
		rtcpHdr, _ = gs.Ntohl(packet)

		totPacketSize := IP_UDP_HDR_SIZE + packetSize

		if packetSize < 4 {
			fmt.Println("RTCP Interface packet Size less than 4.")
			continue
		}

		if (rtcpHdr & 0xE0FE0000) != (0x80000000 | (RTCP_PT_SR << 16)) {
			fmt.Printf("rejected bad RTCP packet: header 0x%08x\n", rtcpHdr)
			continue
		}

		typeOfPacket := PACKET_UNKNOWN_TYPE
		var packetOk bool
		var reportSenderSSRC uint32

		for {
			rc := (rtcpHdr >> 24) & 0x1F
			pt := (rtcpHdr >> 16) & 0xFF
			// doesn't count hdr
			length := uint(4 * (rtcpHdr & 0xFFFF))
			// skip over the header
			packet, packetSize = ADVANCE(packet, packetSize, 4)
			if length > packetSize {
				break
			}

			// Assume that each RTCP subpacket begins with a 4-byte SSRC:
			if length < 4 {
				break
			}
			length -= 4

			reportSenderSSRC, _ = gs.Ntohl(packet)

			packet, packetSize = ADVANCE(packet, packetSize, 4)

			var subPacketOk bool
			switch pt {
			case RTCP_PT_SR:
				if length >= 20 {
					length -= 20

					// Extract the NTP timestamp, and note this:
					NTPmsw, _ := gs.Ntohl(packet)
					packet, packetSize = ADVANCE(packet, packetSize, 4)

					NTPlsm, _ := gs.Ntohl(packet)
					packet, packetSize = ADVANCE(packet, packetSize, 4)

					rtpTimestamp, _ := gs.Ntohl(packet)
					packet, packetSize = ADVANCE(packet, packetSize, 4)

					if r.Source != nil {
						r.Source.receptionStatsDB.noteIncomingSR(reportSenderSSRC, NTPmsw, NTPlsm, rtpTimestamp)
					}

					packet, packetSize = ADVANCE(packet, packetSize, 8)

					// If a 'SR handler' was set, call it now:
					if r.SRHandlerTask != nil {
						//r.SRHandlerTask()
					}
				}

				fallthrough
			case RTCP_PT_RR:
				reportBlocksSize := uint(rc * (6 * 4))
				if length >= reportBlocksSize {
					length -= reportBlocksSize

					if r.Sink != nil {
					} else {
						packet, packetSize = ADVANCE(packet, packetSize, reportBlocksSize)
					}

					if pt == RTCP_PT_RR {
						fmt.Println("RTCP_PT_RR")
						if r.RRHandlerTask != nil {
							//r.RRHandlerTask()
						}
					}

					subPacketOk = true
					typeOfPacket = PACKET_RTCP_REPORT
				}

			case RTCP_PT_BYE:
				fmt.Println("RTCP_PT_BYE")
				callByeHandler = true

				subPacketOk = true
				typeOfPacket = PACKET_BYE
			default:
				subPacketOk = true
			}

			if !subPacketOk {
				break
			}

			packet, packetSize = ADVANCE(packet, packetSize, length)

			if packetSize == 0 {
				packetOk = true
				break
			} else if packetSize < 4 {
				fmt.Printf("extraneous %d bytes at end of RTCP packet!\n", packetSize)
				break
			}

			rtcpHdr, _ = gs.Ntohl(packet)

			if (rtcpHdr & 0xC0000000) != 0x80000000 {
				fmt.Printf("bad RTCP subpacket: header 0x%08x\n", rtcpHdr)
				break
			}
		}

		if !packetOk {
			fmt.Printf("rejected bad RTCP subpacket: header 0x%08x\n", rtcpHdr)
			continue
		} else {
			fmt.Println("validated entire RTCP packet")
		}

		r.onReceive(typeOfPacket, totPacketSize, uint(reportSenderSSRC))

		if callByeHandler && r.byeHandlerTask != nil {
			r.byeHandlerTask.(func(subsession *MediaSubSession))(r.byeHandlerClientData.(*MediaSubSession))
		}
	}
}

func (r *RTCPInstance) onReceive(typeOfPacket int, totPacketSize, ssrc uint) {
	OnReceive()
}

func (r *RTCPInstance) sendReport() {
	// Begin by including a SR and/or RR report:
	r.addReport()

	// Then, include a SDES:
	r.addSDES()

	// Send the report:
	r.sendBuiltPacket()
}

func (r *RTCPInstance) sendBuiltPacket() {
	reportSize := r.outBuf.curPacketSize()
	r.rtcpInterface.sendPacket(r.outBuf.packet(), reportSize)
	r.outBuf.resetOffset()

	r.lastSentSize = uint(IP_UDP_HDR_SIZE) + reportSize
	r.haveJustSentPacket = true
	r.lastPacketSentSize = reportSize
}

func (r *RTCPInstance) addReport() {
	if r.Sink != nil {
		if r.Sink.enableRTCPReports {
			return
		}

		if r.Sink.nextTimestampHasBeenPreset {
			return
		}

		r.addSR()
	} else if r.Source != nil {
		r.addRR()
	}
}

func (r *RTCPInstance) addSDES() {
	numBytes := 4
	//numBytes += r.CNAME.totalSize()
	numBytes += 1

	num4ByteWords := (numBytes + 3) / 4

	var rtcpHdr int64 = 0x81000000 // version 2, no padding, 1 SSRC chunk
	rtcpHdr |= (RTCP_PT_SDES << 16)
	rtcpHdr |= int64(num4ByteWords)
	r.outBuf.enqueueWord(uint(rtcpHdr))
}

func (r *RTCPInstance) addSR() {
	//r.enqueueCommonReportPrefix(RTCP_PT_SR, r.Source.SSRC(), 0)
	r.enqueueCommonReportSuffix()
}

func (r *RTCPInstance) addRR() {
	//r.enqueueCommonReportPrefix(RTCP_PT_RR, r.Source.SSRC(), 0)
	r.enqueueCommonReportSuffix()
}

func (r *RTCPInstance) onExpire() {
	// Note: totsessionbw is kbits per second
	var rtcpBW float32 = 0.05 * float32(r.totSessionBW) * 1024 / 8

	var senders uint
	if r.Sink != nil {
		senders = 1
	}

	OnExpire(r, r.numMembers(), senders, rtcpBW)
}

func (r *RTCPInstance) unsetSpecificRRHandler() {
}

func (r *RTCPInstance) enqueueCommonReportPrefix(packetType, SSRC, numExtraWords uint) {
}

func (r *RTCPInstance) enqueueCommonReportSuffix() {
}
