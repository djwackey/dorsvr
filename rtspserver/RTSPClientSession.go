package rtspserver

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type RTSPClientSession struct {
	isMulticast          bool
	isTimerRunning       bool
	streamAfterSETUP     bool
	numStreamStates      int
	TCPStreamIDCount     uint
	ourSessionID         uint
	streamStates         *StreamServerState
	rtspServer           *RTSPServer
	rtspClientConn       *RTSPClientConnection
	serverMediaSession   *ServerMediaSession
	livenessTimeoutTimer *time.Timer
}

func NewRTSPClientSession(rtspClientConn *RTSPClientConnection, sessionID uint) *RTSPClientSession {
	session := new(RTSPClientSession)
	session.ourSessionID = sessionID
	session.rtspClientConn = rtspClientConn
	session.rtspServer = rtspClientConn.rtspServer
	session.noteLiveness()
	return session
}

func (s *RTSPClientSession) HandleCommandSetup(urlPreSuffix, urlSuffix, reqStr string) {
	streamName, trackID := urlPreSuffix, urlSuffix

	sms := s.rtspServer.LookupServerMediaSession(streamName)
	if sms == nil {
		if s.serverMediaSession == nil {
			s.rtspClientConn.handleCommandNotFound()
		} else {
			s.rtspClientConn.handleCommandBad()
		}
		return
	}

	if s.serverMediaSession == nil {
		s.serverMediaSession = sms
	} else if sms != s.serverMediaSession {
		s.rtspClientConn.handleCommandBad()
		return
	}

	if s.streamStates == nil {
		s.numStreamStates = s.serverMediaSession.subsessionCounter

		s.streamStates = new(streamState)
		for i := 0; i < s.numStreamStates; i++ {
			s.streamStates.subsession = s.serverMediaSession.subSessions[i]
		}
	}

	// Look up information for the specified subsession (track):
	//var streamNum int
	var subsession IServerMediaSubSession
	if trackId != "" {
		for streamNum := 0; streamNum < s.numStreamStates; streamNum++ {
			subsession = s.streamStates.subsession
			fmt.Println("Look up", subsession)
			if subsession != nil && strings.EqualFold(trackId, subsession.TrackID()) {
				break
			}
		}
	} else {
		if s.numStreamStates != 1 && s.streamStates == nil {
			s.rtspClientConn.handleCommandBad()
			return
		}
		subsession = s.streamStates.subsession
	}

	// Look for a "Transport:" header in the request string, to extract client parameters:
	transportHeader := parseTransportHeader(reqStr)
	rtpChannelID := transportHeader.rtpChannelID
	rtcpChannelID := transportHeader.rtcpChannelID
	streamingMode := transportHeader.streamingMode
	clientRTPPort := transportHeader.clientRTPPortNum
	clientRTCPPort := transportHeader.clientRTCPPortNum
	streamingModeStr := transportHeader.streamingModeStr

	if streamingMode == RTP_TCP && rtpChannelID == 0xFF {
		rtpChannelID = s.TCPStreamIDCount
		rtcpChannelID = s.TCPStreamIDCount + 1
	}
	if streamingMode == RTP_TCP {
		rtcpChannelID = s.TCPStreamIDCount + 2
	}

	_, sawRangeHeader := parseRangeHeader(reqStr)
	if sawRangeHeader {
		s.streamAfterSETUP = true
	} else if parsePlayNowHeader(reqStr) {
		s.streamAfterSETUP = true
	} else {
		s.streamAfterSETUP = false
	}

	sourceAddrStr := s.rtspClientConn.localAddr
	destAddrStr := s.rtspClientConn.remoteAddr

	var tcpSocketNum net.Conn
	if streamingMode == RTP_TCP {
		tcpSocketNum = s.rtspClientConn.clientOutputSocket
	}

	streamParameter := subsession.getStreamParameters(tcpSocketNum,
		destAddrStr,
		string(s.ourSessionID),
		clientRTPPort,
		clientRTCPPort,
		rtpChannelID,
		rtcpChannelID)
	serverRTPPort := streamParameter.serverRTPPort
	serverRTCPPort := streamParameter.serverRTCPPort

	//fmt.Println("RTSPClientSession::getStreamParameters", streamParameter, transportHeader)

	s.streamStates.streamToken = streamParameter.streamToken

	if s.isMulticast {
		switch streamingMode {
		case RTP_UDP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP;multicast;destination=%s;source=%s;port=%d-%d;ttl=%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				DateHeader(),
				destAddrStr,
				sourceAddrStr,
				serverRTPPort,
				serverRTCPPort,
				transportHeader.destinationTTL,
				s.ourSessionID)
		case RTP_TCP:
			// multicast streams can't be sent via TCP
			s.rtspClientConn.HandleCommandUnsupportedTransport()
		case RAW_UDP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: %s;multicast;destination=%s;source=%s;port=%d;ttl=%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				DateHeader(),
				destAddrStr,
				sourceAddrStr,
				serverRTPPort,
				serverRTCPPort,
				transportHeader.destinationTTL,
				s.ourSessionID)
		default:
		}
	} else {
		switch streamingMode {
		case RTP_UDP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP;unicast;destination=%s;source=%s;client_port=%d-%d;server_port=%d-%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				DateHeader(),
				destAddrStr,
				sourceAddrStr,
				clientRTPPort,
				clientRTCPPort,
				serverRTPPort,
				serverRTCPPort,
				s.ourSessionID)
		case RTP_TCP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP/TCP;unicast;destination=%s;source=%s;interleaved=%d-%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				DateHeader(),
				destAddrStr,
				sourceAddrStr,
				rtpChannelID,
				rtcpChannelID,
				s.ourSessionID)
		case RAW_UDP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: %s;unicast;destination=%s;source=%s;client_port=%d;server_port=%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				DateHeader(),
				streamingModeStr,
				destAddrStr,
				sourceAddrStr,
				clientRTPPort,
				serverRTPPort,
				s.ourSessionID)
		}
	}
}

func (s *RTSPClientSession) handleCommandWithinSession(cmdName, urlPreSuffix, urlSuffix, fullRequestStr string) {
	fmt.Println("RTSPClientSession::HandleCommandWithinSession", urlPreSuffix, urlSuffix, s.serverMediaSession.StreamName())

	s.noteLiveness()

	var subsession IServerMediaSubSession
	if s.serverMediaSession == nil { // There wasn't a previous SETUP!
		s.rtspClientConn.handleCommandNotSupported()
		return
	} else if urlSuffix != "" && strings.EqualFold(s.serverMediaSession.StreamName(), urlPreSuffix) {
		// Non-aggregated operation.
		// Look up the media subsession whose track id is "urlSuffix":
		for i := 0; i < s.serverMediaSession.subsessionCounter; i++ {
			subsession = s.serverMediaSession.subSessions[i]

			if strings.EqualFold(subsession.TrackID(), urlSuffix) {
				break
			}
		}

		if subsession == nil { // no such track!
			s.rtspClientConn.handleCommandNotFound()
			return
		}
	} else if strings.EqualFold(s.serverMediaSession.StreamName(), urlSuffix) ||
		urlSuffix == "" && strings.EqualFold(s.serverMediaSession.StreamName(), urlPreSuffix) {
		// Aggregated operation
		subsession = nil
	} else if urlPreSuffix != "" && urlSuffix != "" {
		// Aggregated operation, if <urlPreSuffix>/<urlSuffix> is the session (stream) name:
		if strings.EqualFold(s.serverMediaSession.StreamName(), urlPreSuffix) &&
			s.serverMediaSession.StreamName() == "" &&
			strings.EqualFold(s.serverMediaSession.StreamName(), urlSuffix) {
			subsession = nil
		} else {
			s.rtspClientConn.handleCommandNotFound()
			return
		}
	} else { // the request doesn't match a known stream and/or track at all!
		s.rtspClientConn.handleCommandNotFound()
		return
	}

	switch cmdName {
	case "TEARDOWN":
		s.handleCommandTearDown()
	case "PLAY":
		s.handleCommandPlay(subsession, fullRequestStr)
	case "PAUSE":
		s.handleCommandPause()
	case "GET_PARAMETER":
		s.handleCommandGetParameter()
	case "SET_PARAMETER":
		s.handleCommandSetParameter()
	}
}

func (s *RTSPClientSession) handleCommandPlay(subsession IServerMediaSubSession, fullRequestStr string) {
	rtspURL := s.rtspServer.RtspURL(s.serverMediaSession.StreamName())

	// Parse the client's "Scale:" header, if any:
	scale, sawScaleHeader := parseScaleHeader(fullRequestStr)

	// Try to set the stream's scale factor to this value:
	if subsession == nil {
		scale = s.serverMediaSession.testScaleFactor()
	} else {
		scale = subsession.testScaleFactor(scale)
	}

	var buf string
	if sawScaleHeader {
		buf = fmt.Sprintf("Scale: %f\r\n", scale)
	}
	scaleHeaderStr := buf

	buf = ""
	var rangeStart, rangeEnd, duration float32
	var absStartTime, absEndTime string

	rangeHeader, sawRangeHeader := parseRangeHeader(fullRequestStr)
	if sawRangeHeader && rangeHeader.absStartTime == "" {
		if subsession == nil {
			duration = s.serverMediaSession.Duration()
		} else {
			//duration = subsession.Duration()
		}
		if duration < 0 {
			duration = -duration
		}

		rangeStart = rangeHeader.rangeStart
		rangeEnd = rangeHeader.rangeEnd
		absStartTime = rangeHeader.absStartTime
		absEndTime = rangeHeader.absEndTime

		if rangeStart < 0 {
			rangeStart = 0
		} else if rangeStart > duration {
			rangeStart = duration
		}
		if rangeEnd < 0 {
			rangeEnd = 0
		} else if rangeEnd > duration {
			rangeEnd = duration
		}

		if (scale > 0.0 && rangeStart > rangeEnd && rangeEnd > 0.0) || (scale < 0.0 && rangeStart < rangeEnd) {
			// "rangeStart" and "rangeEnd" were the wrong way around; swap them:
			rangeStart, rangeEnd = rangeEnd, rangeStart
		}

		// We're seeking by 'absolute' time:
		if absEndTime == "" {
			buf = fmt.Sprintf("Range: clock=%s-\r\n", absStartTime)
		} else {
			buf = fmt.Sprintf("Range: clock=%s-%s\r\n", absStartTime, absEndTime)
		}
	} else {
		// We're seeking by relative (NPT) time:
		if rangeEnd == 0.0 && scale >= 0.0 {
			buf = fmt.Sprintf("Range: npt=%.3f-\r\n", rangeStart)
		} else {
			buf = fmt.Sprintf("Range: npt=%.3f-%.3f\r\n", rangeStart, rangeEnd)
		}
	}

	for i := 0; i < s.numStreamStates; i++ {
		if subsession == nil || s.numStreamStates == 1 {
			if sawScaleHeader {
				if s.streamStates.subsession != nil {
					//s.streamStates.subsession.setStreamScale(s.ourSessionID, s.streamStates.streamToken, scale)
				}
			}
			if sawRangeHeader {
				// Special case handling for seeking by 'absolute' time:
				if absStartTime != "" {
					if s.streamStates.subsession != nil {
						//s.streamStates.subsession.seekStream(s.ourSessionID, s.streamStates.streamToken, absStartTime, absEndTime)
					}
				} else { // Seeking by relative (NPT) time:
					var streamDuration float32 = 0.0                   // by default; means: stream until the end of the media
					if rangeEnd > 0.0 && (rangeEnd+0.001) < duration { // the 0.001 is because we limited the values to 3 decimal places
						// We want the stream to end early.  Set the duration we want:
						streamDuration = rangeEnd - rangeStart
						if streamDuration < 0.0 {
							streamDuration = -streamDuration // should happen only if scale < 0.0
						}
					}
					if s.streamStates.subsession != nil {
						//var numBytes int
						//s.streamStates.subsession.seekStream(s.ourSessionID, s.streamStates.streamToken, rangeStart, streamDuration, numBytes)
					}
				}
			}
		}
	}

	rangeHeaderStr := buf

	rtpSeqNum, rtpTimestamp := s.streamStates.subsession.startStream(s.ourSessionID, s.streamStates.streamToken)
	urlSuffix := s.streamStates.subsession.TrackID()

	// Create a "RTP-INFO" line. It will get filled in from each subsession's state:
	rtpInfoFmt := "RTP-INFO:" +
		"%s" +
		"url=%s/%s" +
		";seq=%d" +
		";rtptime=%d"

	rtpInfo := fmt.Sprintf(rtpInfoFmt, "0", rtspURL, urlSuffix, rtpSeqNum, rtpTimestamp)

	// Fill in the response:
	s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
		"CSeq: %s\r\n"+
		"%s"+
		"%s"+
		"%s"+
		"Session: %08X\r\n"+
		"%s\r\n", s.rtspClientConn.currentCSeq,
		DateHeader(),
		scaleHeaderStr,
		rangeHeaderStr,
		s.ourSessionID,
		rtpInfo)
}

func (s *RTSPClientSession) handleCommandPause() {
	s.streamStates.subsession.pauseStream(s.streamStates.streamToken)
	/*
		for i := 0; i < s.numStreamStates; i++ {
			s.streamStates[i].subsession.pauseStream()
		}*/

	s.rtspClientConn.setRTSPResponseWithSessionID("200 OK", s.ourSessionID)
}

func (s *RTSPClientSession) handleCommandGetParameter() {
	s.rtspClientConn.setRTSPResponseWithSessionID("200 OK", s.ourSessionID)
}

func (s *RTSPClientSession) handleCommandSetParameter() {
	s.rtspClientConn.setRTSPResponseWithSessionID("200 OK", s.ourSessionID)
}

func (s *RTSPClientSession) handleCommandTearDown() {
	s.streamStates.subsession.deleteStream(s.streamStates.streamToken)
	/*
		for i := 0; i < s.numStreamStates; i++ {
			s.streamStates[i].subsession.deleteStream()
		}*/
}

func (s *RTSPClientSession) noteLiveness() {
	if !s.isTimerRunning {
		go s.livenessTimeoutTask(time.Second * s.rtspServer.reclamationTestSeconds)
		s.isTimerRunning = true
	} else {
		//fmt.Println("noteLiveness", s.livenessTimeoutTimer)
		s.livenessTimeoutTimer.Reset(time.Second * s.rtspServer.reclamationTestSeconds)
	}
}

func (s *RTSPClientSession) livenessTimeoutTask(d time.Duration) {
	s.livenessTimeoutTimer = time.NewTimer(d)

	for {
		select {
		case <-s.livenessTimeoutTimer.C:
			fmt.Println("livenessTimeoutTask")
		}
	}
}
