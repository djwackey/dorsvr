package liveMedia

import (
	"fmt"
	"time"
)

type RTSPClientSession struct {
	isMulticast          bool
	isTimerRunning       bool
	numStreamStates      int
	TCPStreamIdCount     int
	ourSessionId         uint32
	streamStates         *streamState
	rtspServer           *RTSPServer
	rtspClientConn       *RTSPClientConnection
	serverMediaSession   *ServerMediaSession
	livenessTimeoutTimer *time.Timer
}

type streamState struct {
	subsession  *ServerMediaSubSession
	streamToken interface{}
}

func NewRTSPClientSession(rtspClientConn *RTSPClientConnection, sessionId uint32) *RTSPClientSession {
	rtspClientSession := new(RTSPClientSession)
	rtspClientSession.ourSessionId = sessionId
	rtspClientSession.rtspClientConn = rtspClientConn
	rtspClientSession.rtspServer = rtspClientConn.GetRTSPServer()
	rtspClientSession.noteLiveness()
	return rtspClientSession
}

func (this *RTSPClientSession) HandleCommandSetup(urlPreSuffix, urlSuffix, reqStr string) {
	streamName := urlPreSuffix
	//trackId := urlSuffix

	//this.noteLiveness()

	var rtpChannelId int
	var rtcpChannelId int
	var streamingMode int

	sms := this.rtspServer.LookupServerMediaSession(streamName)
	if sms == nil {
		if this.serverMediaSession == nil {
			this.rtspClientConn.handleCommandNotFound()
		} else {
			this.rtspClientConn.handleCommandBad()
		}
		return
	}

	if this.serverMediaSession == nil {
		this.serverMediaSession = sms
	} else if sms != this.serverMediaSession {
		this.rtspClientConn.handleCommandBad()
		return
	}

	streamStates := new(streamState)
	streamStates.subsession = nil

	transportHeader, _ := parseTransportHeader(reqStr)
	if transportHeader.streamingMode == RTP_TCP && transportHeader.rtpChannelId == 0xFF {
		rtpChannelId = this.TCPStreamIdCount
		rtcpChannelId = this.TCPStreamIdCount + 1
	}
	if transportHeader.streamingMode == RTP_TCP {
		rtcpChannelId = this.TCPStreamIdCount + 2
	}

	parseRangeHeader()
	parsePlayNowHeader()

	//var subsession ServerMediaSubSession
	var destAddrStr string
	var sourceAddrStr string
	var streamingModeStr string
	var serverRTPPort int
	var serverRTCPPort int

	//streamParameter := "" //subsession.GetStreamParameters()
	clientRTPPort := "streamParameter.clientRTPPort"
	clientRTCPPort := "streamParameter.clientRTCPPort"

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
			this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP;unicast;destination=%s;source=%s;client_port=%d-%d;server_port=%d-%d\r\n"+
				"Session: %08X\r\n\r\n", this.rtspClientConn.currentCSeq,
				DateHeader(),
				destAddrStr,
				sourceAddrStr,
				clientRTPPort,
				clientRTCPPort,
				serverRTPPort,
				serverRTCPPort,
				this.ourSessionId)
		case RTP_TCP:
			this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP/TCP;unicast;destination=%s;source=%s;interleaved=%d-%d\r\n"+
				"Session: %08X\r\n\r\n", this.rtspClientConn.currentCSeq,
				DateHeader(),
				destAddrStr,
				sourceAddrStr,
				rtpChannelId,
				rtcpChannelId,
				this.ourSessionId)
		case RAW_UDP:
			this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: %s;unicast;destination=%s;source=%s;client_port=%d;server_port=%d\r\n"+
				"Session: %08X\r\n\r\n", this.rtspClientConn.currentCSeq,
				DateHeader(),
				streamingModeStr,
				destAddrStr,
				sourceAddrStr,
				clientRTPPort,
				serverRTPPort,
				this.ourSessionId)
		}
	}
}

func (this *RTSPClientSession) HandleCommandWithinSession(cmdName string) {
	this.noteLiveness()

	var subsession *ServerMediaSubSession
	switch cmdName {
	case "TEARDOWN":
		this.HandleCommandTearDown(subsession)
	case "PLAY":
		//this.HandleCommandPlay(subsession, fullRequestStr)
	case "PAUSE":
		this.HandleCommandPause(subsession)
	case "GET_PARAMETER":
		this.handleCommandGetParameter()
	case "SET_PARAMETER":
		this.handleCommandSetParameter()
	}
}

func (this *RTSPClientSession) HandleCommandPlay(subsession *ServerMediaSubSession, fullRequestStr string) {
	//this.rtspServer.RtspURL()

	parseScaleHeader()

	for i := 0; i < this.numStreamStates; i++ {
		//this.streamStates[i].subsession.startStream()
	}
}

func (this *RTSPClientSession) HandleCommandPause(subsession *ServerMediaSubSession) {
	for i := 0; i < this.numStreamStates; i++ {
		//this.streamStates[i].subsession.pauseStream()
	}

	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) handleCommandGetParameter() {
	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) handleCommandSetParameter() {
	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) HandleCommandTearDown(subsession *ServerMediaSubSession) {
	for i := 0; i < this.numStreamStates; i++ {
		//this.streamStates[i].subsession.deleteStream()
	}
}

func (this *RTSPClientSession) noteLiveness() {
	if !this.isTimerRunning {
		go this.livenessTimeoutTask(time.Second * this.rtspServer.reclamationTestSeconds)
		this.isTimerRunning = true
	} else {
        fmt.Println("noteLiveness", this.livenessTimeoutTimer)
		this.livenessTimeoutTimer.Reset(time.Second * this.rtspServer.reclamationTestSeconds)
	}
}

func (this *RTSPClientSession) livenessTimeoutTask(d time.Duration) {
	this.livenessTimeoutTimer = time.NewTimer(d)

	for {
		select {
		case <-this.livenessTimeoutTimer.C:
			fmt.Println("livenessTimeoutTask")
		}
	}
}
