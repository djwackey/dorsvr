package liveMedia

import "fmt"

type RTSPClientSession struct {
	isMulticast      bool
	numStreamStates  int
	TCPStreamIdCount int
	ourSessionId     uint32
	rtspServer       *RTSPServer
	rtspClientConn   *RTSPClientConnection
}

func NewRTSPClientSession(rtspClientConn *RTSPClientConnection, sessionId uint32) *RTSPClientSession {
	rtspClientSession := new(RTSPClientSession)
	rtspClientSession.ourSessionId = sessionId
	rtspClientSession.rtspClientConn = rtspClientConn
	rtspClientSession.rtspServer = rtspClientConn.GetRTSPServer()
	return rtspClientSession
}

func (this *RTSPClientSession) HandleCommandSetup(cmdName, urlPreSuffix, urlSuffix, reqStr string) {
	this.noteLiveness()

	var streamingMode int

	//sms := this.rtspServer.LookupServerMediaSession(urlTotalSuffix)

	transportHeader, _ := parseTransportHeader(reqStr)
	if transportHeader.streamingMode == RTP_TCP && transportHeader.rtpChannelId == 0xFF {
		//rtpChannelId = this.TCPStreamIdCount
		//rtcpChannelId = this.TCPStreamIdCount + 1
	}
	if transportHeader.streamingMode == RTP_TCP {
		//rtcpChannelId := this.TCPStreamIdCount + 2
	}

	//parseRangeHeader()
	//parsePlayNowHeader()
	//var subsession ServerMediaSubSession
	var destAddrStr string
	var sourceAddrStr string
	var serverRTPPort int
	var serverRTCPPort int

	//subsession.GetStreamParameters()

	if this.isMulticast {
		switch streamingMode {
		case RTP_UDP:
			this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP;multicast;destination=%s;source=%s;port=%d-%d;ttl=%d\r\n"+
				"Session: %08X\r\n\r\n", this.rtspClientConn.currentCSeq,
				DateHeader(),
				destAddrStr,
				sourceAddrStr,
				serverRTPPort,
				serverRTCPPort,
				transportHeader.destinationTTL,
				this.ourSessionId)
		case RTP_TCP:
			// multicast streams can't be sent via TCP
			this.rtspClientConn.HandleCommandUnsupportedTransport()
		case RAW_UDP:
			this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: %s;multicast;destination=%s;source=%s;port=%d;ttl=%d\r\n"+
				"Session: %08X\r\n\r\n", this.rtspClientConn.currentCSeq,
				DateHeader(),
				destAddrStr,
				sourceAddrStr,
				serverRTPPort,
				serverRTCPPort,
				transportHeader.destinationTTL,
				this.ourSessionId)
		default:
		}
	} else {
		switch streamingMode {
		case RTP_UDP:
			this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n" +
				"CSeq: %s\r\n" +
				"%s" +
				"Transport: RTP/AVP;unicast;destination=%s;source=%s;client_port=%d-%d;server_port=%d-%d\r\n" +
				"Session: %08X\r\n\r\n")
		case RTP_TCP:
			this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n" +
				"CSeq: %s\r\n" +
				"%s" +
				"Transport: RTP/AVP/TCP;unicast;destination=%s;source=%s;interleaved=%d-%d\r\n" +
				"Session: %08X\r\n\r\n")
		case RAW_UDP:
			this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n" +
				"CSeq: %s\r\n" +
				"%s" +
				"Transport: %s;unicast;destination=%s;source=%s;client_port=%d;server_port=%d\r\n" +
				"Session: %08X\r\n\r\n")
		}
	}
}

func (this *RTSPClientSession) HandleCommandWithinSession(cmdName string) {
	//var subsession ServerMediaSubSession
	switch cmdName {
	case "TEARDOWN":
		//this.HandleCommandTearDown(this.rtspClientConn, subsession)
	case "PLAY":
		//this.HandleCommandPlay(this.rtspClientConn, subsession, fullRequestStr)
	case "PAUSE":
		//this.HandleCommandPause(this.rtspClientConn, subsession)
	case "GET_PARAMETER":
		//this.HandleCommandGetParameter(this.rtspClientConn, subsession, fullRequestStr)
	case "SET_PARAMETER":
		//this.HandleCommandSetParameter(this.rtspClientConn, subsession, fullRequestStr)
	}
}

func (this *RTSPClientSession) HandleCommandPlay() {
	for i := 0; i < this.numStreamStates; i++ {
		//this.streamStates[i].subsession.startStream()
	}
}

func (this *RTSPClientSession) HandleCommandPause() {
	for i := 0; i < this.numStreamStates; i++ {
		//this.streamStates[i].subsession.pauseStream()
	}

	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) HandleCommandGetParameter() {
	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) HandleCommandSetParameter() {
	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) HandleCommandTearDown() {
	for i := 0; i < this.numStreamStates; i++ {
		//this.streamStates[i].subsession.deleteStream()
	}
}

func (this *RTSPClientSession) noteLiveness() {
}
