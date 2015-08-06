package liveMedia

import (
	"fmt"
	"time"
	"net"
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
	subsession  IServerMediaSubSession
	streamToken *StreamState
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

	var rtpChannelId, rtcpChannelId, streamingMode int

	fmt.Println("LookupServerMediaSession", streamName)

	sms := this.rtspServer.LookupServerMediaSession(streamName)
	if sms == nil {
		if this.serverMediaSession == nil {
			this.rtspClientConn.handleCommandNotFound()
		} else {
			this.rtspClientConn.handleCommandBad()
		}
		return
	}
	//fmt.Println("HandleCommandSetup")

	if this.serverMediaSession == nil {
		this.serverMediaSession = sms
	} else if sms != this.serverMediaSession {
		this.rtspClientConn.handleCommandBad()
		return
	}

	if this.streamStates == nil {
		this.numStreamStates = len(this.serverMediaSession.subSessions)

		this.streamStates = new(streamState)
		this.streamStates.subsession = this.serverMediaSession.subSessions[0]
	}

	transportHeader := parseTransportHeader(reqStr)
	if transportHeader.streamingMode == RTP_TCP && transportHeader.rtpChannelId == 0xFF {
		rtpChannelId = this.TCPStreamIdCount
		rtcpChannelId = this.TCPStreamIdCount + 1
	}
	if transportHeader.streamingMode == RTP_TCP {
		rtcpChannelId = this.TCPStreamIdCount + 2
	}

	clientRTPPort := transportHeader.clientRTPPortNum
	clientRTCPPort := transportHeader.clientRTCPPortNum

	parseRangeHeader(reqStr)
	parsePlayNowHeader(reqStr)

	subsession := this.streamStates.subsession

	var destAddrStr, sourceAddrStr, streamingModeStr string
	var serverRTPPort, serverRTCPPort int

	var tcpSocketNum *net.Conn
	if streamingMode == RTP_TCP {
	    tcpSocketNum = &this.rtspClientConn.clientOutputSocket
	}

	streamParameter := subsession.getStreamParameters(tcpSocketNum, clientRTPPort, clientRTCPPort, rtpChannelId, rtcpChannelId)

	fmt.Println("RTSPClientSession::getStreamParameters", streamParameter, transportHeader)

	this.streamStates.streamToken = streamParameter.streamToken

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

func (this *RTSPClientSession) HandleCommandWithinSession(cmdName, urlPreSuffix, urlSuffix, fullRequestStr string) {
	this.noteLiveness()

	var subsession *ServerMediaSubSession

	//if this.ourServerMediaSession == nil { // There wasn't a previous SETUP!
	//    this.rtspClientConn.handleCommandNotSupported()
	//    return
	//} else if (urlSuffix[0] != '\0' && strcmp(this.serverMediaSession.StreamName(), urlPreSuffix) == 0) {
	//    // Non-aggregated operation.
	//    // Look up the media subsession whose track id is "urlSuffix":
	//    ServerMediaSubsessionIterator iter(*fOurServerMediaSession)
	//    for ((subsession = iter.next()) != NULL) {
	//        if (strcmp(subsession.TrackId(), urlSuffix) == 0) {
	//            break // success
	//        }
	//    }

	//    if (subsession == NULL) { // no such track!
	//        this.rtspClientConn.handleCommandNotFound()
	//        return
	//    }
	//} else if strcmp(this.serverMediaSession.StreamName(), urlSuffix) == 0 ||
	//         (urlSuffix[0] == '\0' && strcmp(this.serverMediaSession.StreamName(), urlPreSuffix) == 0) {
	//    // Aggregated operation
	//    subsession = nil
	//} else if (urlPreSuffix[0] != '\0' && urlSuffix[0] != '\0') {
	//    // Aggregated operation, if <urlPreSuffix>/<urlSuffix> is the session (stream) name:
	//    urlPreSuffixLen := strlen(urlPreSuffix)
	//    if strncmp(this.serverMediaSession.StreamName(), urlPreSuffix, urlPreSuffixLen) == 0 &&
	//        this.serverMediaSession.StreamName()[urlPreSuffixLen] == '/' &&
	//        strcmp(&(this.ourServerMediaSession.StreamName())[urlPreSuffixLen+1], urlSuffix) == 0 {
	//        subsession = nil
	//    } else {
	//        this.rtspClientConn.handleCommandNotFound()
	//        return
	//    }
	//} else { // the request doesn't match a known stream and/or track at all!
	//    this.rtspClientConn.handleCommandNotFound()
	//    return
	//}

	switch cmdName {
	case "TEARDOWN":
		this.HandleCommandTearDown(subsession)
	case "PLAY":
		this.HandleCommandPlay(subsession, fullRequestStr)
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

	parseScaleHeader(fullRequestStr)

	this.streamStates.subsession.startStream(this.streamStates.streamToken)

	/*
		for i := 0; i < this.numStreamStates; i++ {
			this.streamStates[i].subsession.startStream()
		}*/

	// Create a "RTP-INFO" line. It will get filled in from each subsession's state:
	/*
	   rtpInfoFmt := "%s"+
	                 "%s"+
	                 "url=%s/%s"+
	                 ";seq=%d"+
	                 ";rtptime=%d"
	*/
	rtpInfo := "RTP-INFO: "

	scaleHeader := ""
	rangeHeader := "" //fmt.Sprintf("Range: clock=%s-\r\n", absStart)

	// Fill in the response:
	this.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
		"CSeq: %s\r\n"+
		"%s"+
		"%s"+
		"%s"+
		"Session: %08X\r\n"+
		"%s\r\n", this.rtspClientConn.currentCSeq,
		DateHeader(),
		scaleHeader,
		rangeHeader,
		this.ourSessionId,
		rtpInfo)
}

func (this *RTSPClientSession) HandleCommandPause(subsession *ServerMediaSubSession) {
	this.streamStates.subsession.pauseStream(this.streamStates.streamToken)
	/*
		for i := 0; i < this.numStreamStates; i++ {
			this.streamStates[i].subsession.pauseStream()
		}*/

	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) handleCommandGetParameter() {
	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) handleCommandSetParameter() {
	this.rtspClientConn.setRTSPResponseWithSessionId("200 OK", this.ourSessionId)
}

func (this *RTSPClientSession) HandleCommandTearDown(subsession *ServerMediaSubSession) {
	this.streamStates.subsession.deleteStream(this.streamStates.streamToken)
	/*
		for i := 0; i < this.numStreamStates; i++ {
			this.streamStates[i].subsession.deleteStream()
		}*/
}

func (this *RTSPClientSession) noteLiveness() {
	if !this.isTimerRunning {
		go this.livenessTimeoutTask(time.Second * this.rtspServer.reclamationTestSeconds)
		this.isTimerRunning = true
	} else {
		//fmt.Println("noteLiveness", this.livenessTimeoutTimer)
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
