package liveMedia

import (
	. "groupsock"
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
)

type SDESItem struct {
    data []byte
}

type RTCPInstance struct {
    typeOfEvent int
    totSessionBW uint
    CNAME SDESItem
    Sink *RTPSink
    Source *RTPSource
    outBuf *OutPacketBuffer
    rtcpInterface *RTCPInterface
}

func NewSDESItem(tag byte, value string) *SDESItem {
    length := len(value)
    if length > 0xFF {
        length = 0xFF   // maximum data length for a SDES item
    }

    this.data[0] = tag
    this.data[1] = (byte) length
    return new(SDESItem)
}

func (this *SDESItem) totalSize() uint {
    return 2 + (uint) this.data[1]
}

func NewRTCPInstance(RTCPgs *GroupSock, totSessionBW uint, cname string) *RTCPInstance {
    rtcp := new(RTCPInstance)
    rtcp.typeOfEvent = EVENT_REPORT
    rtcp.totSessionBW = totSessionBW
    rtcp.outBuf = NewOutPacketBuffer()
    rtcp.CNAME = NewSDESItem(RTCP_SDES_CNAME, cname)

    rtcp.rtcpInterface = NewRTCPInterface(rtcp, RTCPgs)
    rtcp.rtcpInterface.startNetworkReading()

    go rtcp.incomingReportHandler()
    //this.onExpire(rtcp)
    return rtcp
}

func (this *RTCPInstance) setByeHandler(handlerTask interface{}, clientData interface{}) {
    //this.byeHandlerTask = handlerTask
    //this.byeHandlerClientData = clientData
}

func (this *RTCPInstance) setSRHandler() {
}

func (this *RTCPInstance) setRRHandler() {
}

func (this *RTCPInstance) incomingReportHandler() {
}
