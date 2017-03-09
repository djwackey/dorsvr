package rtspserver

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/djwackey/dorsvr/livemedia"
)

type RTSPClientSession struct {
	isMulticast          bool
	isTimerRunning       bool
	streamAfterSETUP     bool
	numStreamStates      int
	TCPStreamIDCount     uint
	ourSessionID         uint
	rtspServer           *RTSPServer
	streamStates         *StreamServerState
	rtspClientConn       *RTSPClientConnection
	serverMediaSession   *livemedia.ServerMediaSession
	livenessTimeoutTimer *time.Timer
}

func newRTSPClientSession(rtspClientConn *RTSPClientConnection, sessionID uint) *RTSPClientSession {
	session := new(RTSPClientSession)
	session.ourSessionID = sessionID
	session.rtspClientConn = rtspClientConn
	session.rtspServer = rtspClientConn.rtspServer
	session.noteLiveness()
	return session
}

func (s *RTSPClientSession) handleCommandSetup(urlPreSuffix, urlSuffix, reqStr string) {
	streamName, trackID := urlPreSuffix, urlSuffix

	sms := s.rtspServer.lookupServerMediaSession(streamName)
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
		s.numStreamStates = s.serverMediaSession.SubsessionCounter

		s.streamStates = new(StreamServerState)
		for i := 0; i < s.numStreamStates; i++ {
			s.streamStates.subsession = s.serverMediaSession.SubSessions[i]
		}
	}

	// Look up information for the specified subsession (track):
	//var streamNum int
	var subsession livemedia.IServerMediaSubSession
	if trackID != "" {
		for streamNum := 0; streamNum < s.numStreamStates; streamNum++ {
			subsession = s.streamStates.subsession
			if subsession != nil && strings.EqualFold(trackID, subsession.TrackID()) {
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
	transportHeader := livemedia.ParseTransportHeader(reqStr)
	rtpChannelID := transportHeader.RTPChannelID
	rtcpChannelID := transportHeader.RTCPChannelID
	streamingMode := transportHeader.StreamingMode
	clientRTPPort := transportHeader.ClientRTPPortNum
	clientRTCPPort := transportHeader.ClientRTCPPortNum
	streamingModeStr := transportHeader.StreamingModeStr

	if streamingMode == livemedia.RTP_TCP && rtpChannelID == 0xFF {
		rtpChannelID = s.TCPStreamIDCount
		rtcpChannelID = s.TCPStreamIDCount + 1
	}
	if streamingMode == livemedia.RTP_TCP {
		rtcpChannelID = s.TCPStreamIDCount + 2
	}

	_, sawRangeHeader := livemedia.ParseRangeHeader(reqStr)
	if sawRangeHeader {
		s.streamAfterSETUP = true
	} else if livemedia.ParsePlayNowHeader(reqStr) {
		s.streamAfterSETUP = true
	} else {
		s.streamAfterSETUP = false
	}

	sourceAddrStr := s.rtspClientConn.localAddr
	destAddrStr := s.rtspClientConn.remoteAddr

	var tcpSocketNum net.Conn
	if streamingMode == livemedia.RTP_TCP {
		tcpSocketNum = s.rtspClientConn.clientOutputSocket
	}

	streamParameter := subsession.GetStreamParameters(tcpSocketNum,
		destAddrStr,
		string(s.ourSessionID),
		clientRTPPort,
		clientRTCPPort,
		rtpChannelID,
		rtcpChannelID)
	serverRTPPort := streamParameter.ServerRTPPort
	serverRTCPPort := streamParameter.ServerRTCPPort

	s.streamStates.streamToken = streamParameter.StreamToken

	if s.isMulticast {
		switch streamingMode {
		case livemedia.RTP_UDP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP;multicast;destination=%s;source=%s;port=%d-%d;ttl=%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				livemedia.DateHeader(),
				destAddrStr,
				sourceAddrStr,
				serverRTPPort,
				serverRTCPPort,
				transportHeader.DestinationTTL,
				s.ourSessionID)
		case livemedia.RTP_TCP:
			// multicast streams can't be sent via TCP
			s.rtspClientConn.handleCommandUnsupportedTransport()
		case livemedia.RAW_UDP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: %s;multicast;destination=%s;source=%s;port=%d;ttl=%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				livemedia.DateHeader(),
				destAddrStr,
				sourceAddrStr,
				serverRTPPort,
				serverRTCPPort,
				transportHeader.DestinationTTL,
				s.ourSessionID)
		default:
		}
	} else {
		switch streamingMode {
		case livemedia.RTP_UDP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP;unicast;destination=%s;source=%s;client_port=%d-%d;server_port=%d-%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				livemedia.DateHeader(),
				destAddrStr,
				sourceAddrStr,
				clientRTPPort,
				clientRTCPPort,
				serverRTPPort,
				serverRTCPPort,
				s.ourSessionID)
		case livemedia.RTP_TCP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: RTP/AVP/TCP;unicast;destination=%s;source=%s;interleaved=%d-%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				livemedia.DateHeader(),
				destAddrStr,
				sourceAddrStr,
				rtpChannelID,
				rtcpChannelID,
				s.ourSessionID)
		case livemedia.RAW_UDP:
			s.rtspClientConn.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
				"CSeq: %s\r\n"+
				"%s"+
				"Transport: %s;unicast;destination=%s;source=%s;client_port=%d;server_port=%d\r\n"+
				"Session: %08X\r\n\r\n", s.rtspClientConn.currentCSeq,
				livemedia.DateHeader(),
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
	s.noteLiveness()

	var subsession livemedia.IServerMediaSubSession
	if s.serverMediaSession == nil { // There wasn't a previous SETUP!
		s.rtspClientConn.handleCommandNotSupported()
		return
	} else if urlSuffix != "" && strings.EqualFold(s.serverMediaSession.StreamName(), urlPreSuffix) {
		// Non-aggregated operation.
		// Look up the media subsession whose track id is "urlSuffix":
		for i := 0; i < s.serverMediaSession.SubsessionCounter; i++ {
			subsession = s.serverMediaSession.SubSessions[i]

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

func (s *RTSPClientSession) handleCommandPlay(subsession livemedia.IServerMediaSubSession, fullRequestStr string) {
	rtspURL := s.rtspServer.RtspURL(s.serverMediaSession.StreamName())

	// Parse the client's "Scale:" header, if any:
	scale, sawScaleHeader := livemedia.ParseScaleHeader(fullRequestStr)

	// Try to set the stream's scale factor to this value:
	if subsession == nil {
		scale = s.serverMediaSession.TestScaleFactor()
	} else {
		scale = subsession.TestScaleFactor(scale)
	}

	var buf string
	if sawScaleHeader {
		buf = fmt.Sprintf("Scale: %f\r\n", scale)
	}
	scaleHeaderStr := buf

	buf = ""
	var rangeStart, rangeEnd, duration float32
	var absStartTime, absEndTime string

	rangeHeader, sawRangeHeader := livemedia.ParseRangeHeader(fullRequestStr)
	if sawRangeHeader && rangeHeader.AbsStartTime == "" {
		if subsession == nil {
			duration = s.serverMediaSession.Duration()
		} else {
			//duration = subsession.Duration()
		}
		if duration < 0 {
			duration = -duration
		}

		rangeStart = rangeHeader.RangeStart
		rangeEnd = rangeHeader.RangeEnd
		absStartTime = rangeHeader.AbsStartTime
		absEndTime = rangeHeader.AbsEndTime

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

	rtpSeqNum, rtpTimestamp := s.streamStates.subsession.StartStream(s.ourSessionID, s.streamStates.streamToken,
		s.noteLiveness, s.rtspClientConn.handleAlternativeRequestByte)
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
		livemedia.DateHeader(),
		scaleHeaderStr,
		rangeHeaderStr,
		s.ourSessionID,
		rtpInfo)
}

func (s *RTSPClientSession) handleCommandPause() {
	s.streamStates.subsession.PauseStream(s.streamStates.streamToken)

	//for i := 0; i < s.numStreamStates; i++ {
	//	s.streamStates[i].subsession.PauseStream()
	//}

	s.rtspClientConn.setRTSPResponseWithSessionID("200 OK", s.ourSessionID)
}

func (s *RTSPClientSession) handleCommandGetParameter() {
	s.rtspClientConn.setRTSPResponseWithSessionID("200 OK", s.ourSessionID)
}

func (s *RTSPClientSession) handleCommandSetParameter() {
	s.rtspClientConn.setRTSPResponseWithSessionID("200 OK", s.ourSessionID)
}

func (s *RTSPClientSession) handleCommandTearDown() {
	s.streamStates.subsession.DeleteStream(s.streamStates.streamToken)

	//for i := 0; i < s.numStreamStates; i++ {
	//	s.streamStates[i].subsession.DeleteStream()
	//}
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
