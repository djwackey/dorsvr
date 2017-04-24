package livemedia

import (
	"fmt"
	"net"
	"os"

	gs "github.com/djwackey/dorsvr/groupsock"
)

type OnDemandServerMediaSubsession struct {
	ServerMediaSubsession
	cname            string
	sdpLines         string
	portNumForSDP    int
	initialPortNum   uint
	reuseFirstSource bool
	lastStreamToken  *StreamState
	destinations     map[string]*Destinations
}

type StreamParameter struct {
	IsMulticast     bool
	ClientRTPPort   uint
	ClientRTCPPort  uint
	ServerRTPPort   uint
	ServerRTCPPort  uint
	DestinationTTL  uint
	DestinationAddr string
	StreamToken     *StreamState
}

func (s *OnDemandServerMediaSubsession) initOnDemandServerMediaSubsession(isubsession IServerMediaSubsession) {
	s.initialPortNum = 6970
	s.cname, _ = os.Hostname()
	s.destinations = make(map[string]*Destinations)
	s.initBaseClass(isubsession)
}

func (s *OnDemandServerMediaSubsession) SDPLines() string {
	if s.sdpLines == "" {
		rtpPayloadType := 96 + s.TrackNumber() - 1

		var dummyAddr string
		dummyGroupSock := gs.NewGroupSock(dummyAddr, 0)
		dummyRTPSink := s.isubsession.createNewRTPSink(dummyGroupSock, rtpPayloadType)
		inputSource := s.isubsession.createNewStreamSource()

		s.setSDPLinesFromRTPSink(dummyRTPSink, inputSource, 500)
		dummyRTPSink.destroy()
		inputSource.destroy()
	}

	return s.sdpLines
}

func (s *OnDemandServerMediaSubsession) GetStreamParameters(tcpSocketNum net.Conn, destAddr,
	clientSessionID string, clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID uint) *StreamParameter {
	var streamBitrate uint = 500

	sp := new(StreamParameter)

	if s.lastStreamToken != nil {
		streamState := s.lastStreamToken
		sp.ServerRTPPort = streamState.ServerRTPPort()
		sp.ServerRTCPPort = streamState.ServerRTCPPort()

		sp.StreamToken = s.lastStreamToken
	} else {
		mediaSource := s.isubsession.createNewStreamSource()

		var dummyAddr string
		var rtpSink IMediaSink
		var udpSink *BasicUDPSink
		var rtpGroupSock, rtcpGroupSock *gs.GroupSock

		sp.ServerRTPPort = s.initialPortNum
		if clientRTCPPort == 0 {
			// We're streaming raw UDP (not RTP). Create a single groupsock:
			for {
				rtpGroupSock = gs.NewGroupSock(dummyAddr, sp.ServerRTPPort)
				if rtpGroupSock != nil {
					break
				}
				sp.ServerRTPPort++
			}
			udpSink = NewBasicUDPSink(rtpGroupSock)
		} else {
			// Normal case: We're streaming RTP (over UDP or TCP).  Create a pair of
			// groupsocks (RTP and RTCP), with adjacent port numbers (RTP port number even):
			for {
				rtpGroupSock = gs.NewGroupSock(dummyAddr, sp.ServerRTPPort)
				if rtpGroupSock == nil {
					sp.ServerRTPPort += 2
					continue
				}

				sp.ServerRTCPPort = sp.ServerRTPPort + 1
				rtcpGroupSock = gs.NewGroupSock(dummyAddr, sp.ServerRTCPPort)
				if rtcpGroupSock == nil {
					sp.ServerRTPPort += 2
					continue
				}
				break
			}
			rtpPayloadType := 96 + s.TrackNumber() - 1
			rtpSink = s.isubsession.createNewRTPSink(rtpGroupSock, rtpPayloadType)
		}

		// Set up the state of the stream.  The stream will get started later:
		s.lastStreamToken = newStreamState(s.isubsession,
			sp.ServerRTPPort,
			sp.ServerRTCPPort,
			rtpSink,
			udpSink,
			streamBitrate,
			mediaSource,
			rtpGroupSock,
			rtcpGroupSock)
		sp.StreamToken = s.lastStreamToken
	}

	// Record these destinations as being for this client session id:
	dests := newDestinations(tcpSocketNum, destAddr, clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID)
	s.destinations[clientSessionID] = dests

	return sp
}

func (s *OnDemandServerMediaSubsession) getAuxSDPLine(rtpSink IMediaSink, inputSource IFramedSource) string {
	if rtpSink == nil {
		return ""
	}

	return rtpSink.AuxSDPLine()
}

func (s *OnDemandServerMediaSubsession) setSDPLinesFromRTPSink(rtpSink IMediaSink, inputSource IFramedSource, estBitrate uint) {
	if rtpSink == nil {
		return
	}

	mediaType := rtpSink.sdpMediaType()
	rtpmapLine := rtpSink.rtpmapLine()
	rtpPayloadType := rtpSink.rtpPayloadType()

	rangeLine := s.rangeSDPLine()
	auxSDPLine := s.isubsession.getAuxSDPLine(rtpSink, inputSource)
	if auxSDPLine == "" {
		auxSDPLine = ""
	}

	ipAddr := "0.0.0.0"
	sdpFmt := "m=%s %d RTP/AVP %d\r\n" +
		"c=IN IP4 %s\r\n" +
		"b=AS:%d\r\n" +
		"%s" +
		"%s" +
		"%s" +
		"a=control:%s\r\n"

	s.sdpLines = fmt.Sprintf(sdpFmt,
		mediaType,
		s.portNumForSDP,
		rtpPayloadType,
		ipAddr,
		estBitrate,
		rtpmapLine,
		rangeLine,
		auxSDPLine,
		s.TrackID())
}

func (s *OnDemandServerMediaSubsession) CNAME() string {
	return s.cname
}

func (s *OnDemandServerMediaSubsession) StartStream(clientSessionID string, streamState *StreamState,
	rtcpRRHandler, serverRequestAlternativeByteHandler interface{}) (rtpSeqNum, rtpTimestamp uint32) {
	destinations, _ := s.destinations[clientSessionID]
	go streamState.startPlaying(destinations, rtcpRRHandler, serverRequestAlternativeByteHandler)

	if streamState.RtpSink() != nil {
		rtpSeqNum = streamState.RtpSink().currentSeqNo()
		rtpTimestamp = streamState.RtpSink().presetNextTimestamp()
	}
	return
}

func (s *OnDemandServerMediaSubsession) SeekStream(sessionID string, streamState *StreamState, streamDuration float32) {
	if s.reuseFirstSource {
		return
	}
}

func (s *OnDemandServerMediaSubsession) PauseStream(streamState *StreamState) {
	streamState.pause()
}

func (s *OnDemandServerMediaSubsession) DeleteStream(sessionID string, streamState *StreamState) {
	if dest, existed := s.destinations[sessionID]; existed {
		streamState.endPlaying(dest)
		delete(s.destinations, sessionID)
	}

	if streamState != nil {
		streamState.reclaim()
	}
}

//////// Destinations ////////
type Destinations struct {
	isTCP         bool
	addrStr       string
	rtpPort       uint
	rtcpPort      uint
	rtpChannelID  uint
	rtcpChannelID uint
	tcpSocketNum  net.Conn
}

func newDestinations(tcpSocketNum net.Conn, destAddr string,
	clientRTPPort, clientRTCPPort, rtpChannelID, rtcpChannelID uint) *Destinations {
	var isTCP bool
	if tcpSocketNum != nil {
		isTCP = true
	}
	return &Destinations{
		isTCP:         isTCP,
		addrStr:       destAddr,
		rtpPort:       clientRTPPort,
		rtcpPort:      clientRTCPPort,
		rtpChannelID:  rtpChannelID,
		rtcpChannelID: rtcpChannelID,
		tcpSocketNum:  tcpSocketNum,
	}
}
