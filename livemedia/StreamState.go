package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

//////// StreamState ////////
type StreamState struct {
	master              IServerMediaSubSession
	rtpSink             IRTPSink
	udpSink             *BasicUDPSink
	rtpGS               *gs.GroupSock
	rtcpGS              *gs.GroupSock
	rtcpInstance        *RTCPInstance
	mediaSource         IFramedSource
	serverRTPPort       uint
	serverRTCPPort      uint
	totalBW             uint
	areCurrentlyPlaying bool
}

func NewStreamState(master IServerMediaSubSession, serverRTPPort, serverRTCPPort uint,
	rtpSink IRTPSink, udpSink *BasicUDPSink, totalBW uint,
	mediaSource IFramedSource, rtpGS, rtcpGS *gs.GroupSock) *StreamState {
	state := new(StreamState)
	state.rtpGS = rtpGS
	state.rtcpGS = rtcpGS
	state.master = master
	state.rtpSink = rtpSink
	state.udpSink = udpSink
	state.totalBW = totalBW
	state.mediaSource = mediaSource
	state.serverRTPPort = serverRTPPort
	state.serverRTCPPort = serverRTCPPort
	return state
}

func (s *StreamState) startPlaying(dests *Destinations) {
	if dests == nil {
		return
	}

	if s.rtcpInstance == nil && s.rtpSink != nil {
		s.rtcpInstance = NewRTCPInstance(s.rtcpGS, s.totalBW, s.master.CNAME())
	}

	if dests.isTCP {
		if s.rtpSink != nil {
			s.rtpSink.addStreamSocket(dests.tcpSockNum, dests.rtpChannelID)
			//s.rtpSink.setServerRequestAlternativeByteHandler(dests.tcpSocketNum)
		}
		if s.rtcpInstance != nil {
			s.rtcpInstance.setSpecificRRHandler()
		}
	} else {
		// Tell the RTP and RTCP 'groupsocks' about this destination
		// (in case they don't already have it):
		if s.rtpGS != nil {
			s.rtpGS.AddDestination(dests.addrStr, dests.rtpPort)
		}
		if s.rtcpGS != nil {
			s.rtcpGS.AddDestination(dests.addrStr, dests.rtcpPort)
		}
		if s.rtcpInstance != nil {
			//rtcpRRHandler := ""
			//rtcpRRHandlerClientData := ""
			//this.rtcpInstance.setSpecificRRHandler(dests.addr, dests.rtcpPort, rtcpRRHandler, rtcpRRHandlerClientData)
		}
	}

	if !s.areCurrentlyPlaying && s.mediaSource != nil {
		if s.rtpSink != nil {
			s.rtpSink.StartPlaying(s.mediaSource)
			s.areCurrentlyPlaying = true
		} else if s.udpSink != nil {
			s.areCurrentlyPlaying = true
			s.udpSink.StartPlaying(s.mediaSource)
		}
	}
}

func (s *StreamState) pause() {
	if s.rtpSink != nil {
		s.rtpSink.StopPlaying()
	}
	if s.udpSink != nil {
		s.udpSink.StopPlaying()
	}
	s.areCurrentlyPlaying = false
}

func (s *StreamState) endPlaying(dests *Destinations) {
	if s.rtpSink != nil {
		s.rtpSink.delStreamSocket()
	}
	if s.rtcpInstance != nil {
		s.rtcpInstance.unsetSpecificRRHandler()
	}
}

func (s *StreamState) ServerRTPPort() uint {
	return s.serverRTPPort
}

func (s *StreamState) ServerRTCPPort() uint {
	return s.serverRTCPPort
}

func (s *StreamState) afterPlayingStreamState() {
	s.reclaim()
}

func (s *StreamState) reclaim() {
}

func (s *StreamState) RtpSink() IRTPSink {
	return s.rtpSink
}
