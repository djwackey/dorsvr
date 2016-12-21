package livemedia

import (
	"fmt"
	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/dorsvr/utils"
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
	prevReportTime       int64
	nextReportTime       int64
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

func NewSDESItem(tag int, value string) *SDESItem {
	item := new(SDESItem)

	length := len(value)
	if length > 0xFF {
		length = 0xFF // maximum data length for a SDES item
	}

	item.data = []byte{byte(tag), byte(length)}
	return item
}

func dTimeNow() int64 {
	var timeNow utils.Timeval
	utils.GetTimeOfDay(&timeNow)
	return timeNow.Tv_sec + timeNow.Tv_usec/1000000.0
}

func (this *SDESItem) totalSize() uint {
	return 2 + uint(this.data[1])
}

func NewRTCPInstance(rtcpGS *gs.GroupSock, totSessionBW uint, cname string) *RTCPInstance {
	rtcp := new(RTCPInstance)
	rtcp.typeOfEvent = EVENT_REPORT
	rtcp.totSessionBW = totSessionBW
	rtcp.CNAME = NewSDESItem(RTCP_SDES_CNAME, cname)

	rtcp.prevReportTime = dTimeNow()
	rtcp.nextReportTime = rtcp.prevReportTime

	rtcp.inBuf = make([]byte, maxRTCPPacketSize)
	rtcp.outBuf = NewOutPacketBuffer(preferredPacketSize, maxRTCPPacketSize)

	rtcp.rtcpInterface = NewRTPInterface(rtcp, rtcpGS)
	rtcp.rtcpInterface.startNetworkReading(rtcp.incomingReportHandler)

	rtcp.onExpire()
	return rtcp
}

func (instance *RTCPInstance) numMembers() uint {
	return 0
}

func (instance *RTCPInstance) setSpecificRRHandler() {
}

func (this *RTCPInstance) SetByeHandler(handlerTask interface{}, clientData interface{}) {
	this.byeHandlerTask = handlerTask
	this.byeHandlerClientData = clientData
}

func (instance *RTCPInstance) setSRHandler(handlerTask interface{}, clientData interface{}) {
	instance.SRHandlerTask = handlerTask
}

func (instance *RTCPInstance) setRRHandler(handlerTask interface{}, clientData interface{}) {
	instance.RRHandlerTask = handlerTask
}

func (this *RTCPInstance) incomingReportHandler() {
	var callByeHandler bool
	for {
		readBytes, err := this.rtcpInterface.handleRead(this.inBuf)
		if err != nil {
			fmt.Println("RTCP Interface failed to handle read.", err.Error())
			break
		}

		packet := this.inBuf[:readBytes]
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

					if this.Source != nil {
						receptionStats := this.Source.ReceptionStatsDB()
						receptionStats.noteIncomingSR(reportSenderSSRC, NTPmsw, NTPlsm, rtpTimestamp)
					}

					packet, packetSize = ADVANCE(packet, packetSize, 8)

					// If a 'SR handler' was set, call it now:
					if this.SRHandlerTask != nil {
						//this.SRHandlerTask()
					}
				}

				fallthrough
			case RTCP_PT_RR:
				reportBlocksSize := uint(rc * (6 * 4))
				if length >= reportBlocksSize {
					length -= reportBlocksSize

					if this.Sink != nil {
					} else {
						packet, packetSize = ADVANCE(packet, packetSize, reportBlocksSize)
					}

					if pt == RTCP_PT_RR {
						fmt.Println("RTCP_PT_RR")
						if this.RRHandlerTask != nil {
							//this.RRHandlerTask()
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
				fmt.Println("extraneous %d bytes at end of RTCP packet!\n", packetSize)
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

		this.onReceive(typeOfPacket, totPacketSize, uint(reportSenderSSRC))

		if callByeHandler && this.byeHandlerTask != nil {
			this.byeHandlerTask.(func(subsession *MediaSubSession))(this.byeHandlerClientData.(*MediaSubSession))
		}
	}
}

func (this *RTCPInstance) onReceive(typeOfPacket int, totPacketSize, ssrc uint) {
	OnReceive()
}

func (this *RTCPInstance) sendReport() {
	// Begin by including a SR and/or RR report:
	this.addReport()

	// Then, include a SDES:
	this.addSDES()

	// Send the report:
	this.sendBuiltPacket()
}

func (this *RTCPInstance) sendBuiltPacket() {
	reportSize := this.outBuf.curPacketSize()
	this.rtcpInterface.sendPacket(this.outBuf.packet(), reportSize)
	this.outBuf.resetOffset()

	this.lastSentSize = uint(IP_UDP_HDR_SIZE) + reportSize
	this.haveJustSentPacket = true
	this.lastPacketSentSize = reportSize
}

func (this *RTCPInstance) addReport() {
	if this.Sink != nil {
		if this.Sink.EnableRTCPReports() {
			return
		}

		if this.Sink.NextTimestampHasBeenPreset() {
			return
		}

		this.addSR()
	} else if this.Source != nil {
		this.addRR()
	}
}

func (this *RTCPInstance) addSDES() {
	numBytes := 4
	//numBytes += this.CNAME.totalSize()
	numBytes += 1

	num4ByteWords := (numBytes + 3) / 4

	var rtcpHdr int64 = 0x81000000 // version 2, no padding, 1 SSRC chunk
	rtcpHdr |= (RTCP_PT_SDES << 16)
	rtcpHdr |= int64(num4ByteWords)
	this.outBuf.enqueueWord(uint(rtcpHdr))
}

func (instance *RTCPInstance) addSR() {
	//this.enqueueCommonReportPrefix(RTCP_PT_SR, this.Source.SSRC(), 0)
	instance.enqueueCommonReportSuffix()
}

func (instance *RTCPInstance) addRR() {
	//this.enqueueCommonReportPrefix(RTCP_PT_RR, this.Source.SSRC(), 0)
	instance.enqueueCommonReportSuffix()
}

func (instance *RTCPInstance) onExpire() {
	// Note: totsessionbw is kbits per second
	var rtcpBW float32 = 0.05 * float32(instance.totSessionBW) * 1024 / 8

	var senders uint
	if instance.Sink != nil {
		senders = 1
	}

	OnExpire(instance, instance.numMembers(), senders, rtcpBW)
}

func (instance *RTCPInstance) unsetSpecificRRHandler() {
}

func (instance *RTCPInstance) enqueueCommonReportPrefix(packetType, SSRC, numExtraWords uint) {
}

func (instance *RTCPInstance) enqueueCommonReportSuffix() {
}
