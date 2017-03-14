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

func newStreamState(master IServerMediaSubSession, serverRTPPort, serverRTCPPort uint,
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

func (s *StreamState) startPlaying(dests *Destinations,
	rtcpRRHandler, serverRequestAlternativeByteHandler interface{}) {
	if dests == nil {
		return
	}

	if s.rtcpInstance == nil && s.rtpSink != nil {
		// Note: This starts RTCP running automatically
		// Create (and start) a 'RTCP instance' for this RTP sink:
		s.rtcpInstance = newRTCPInstance(s.rtcpGS, s.totalBW, s.master.CNAME())
	}

	if dests.isTCP {
		if s.rtpSink != nil {
			s.rtpSink.addStreamSocket(dests.tcpSocketNum, dests.rtpChannelID)
			s.rtpSink.setServerRequestAlternativeByteHandler(dests.tcpSocketNum, serverRequestAlternativeByteHandler)
		}
		if s.rtcpInstance != nil {
			s.rtcpInstance.setSpecificRRHandler(rtcpRRHandler)
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
			s.rtcpInstance.setSpecificRRHandler(rtcpRRHandler)
		}
	}

	if !s.areCurrentlyPlaying && s.mediaSource != nil {
		if s.rtpSink != nil {
			s.rtpSink.StartPlaying(s.mediaSource, s.afterPlayingStreamState)
			s.areCurrentlyPlaying = true
		} else if s.udpSink != nil {
			s.areCurrentlyPlaying = true
			s.udpSink.StartPlaying(s.mediaSource, s.afterPlayingStreamState)
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
	if dests.isTCP {
		if s.rtpSink != nil {
			s.rtpSink.delStreamSocket(dests.tcpSocketNum, dests.rtpChannelID)
		}
		if s.rtcpInstance != nil {
			s.rtcpInstance.unsetSpecificRRHandler()
		}
	} else {
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
	s.rtcpInstance.destroy()
}

func (s *StreamState) RtpSink() IRTPSink {
	return s.rtpSink
}
