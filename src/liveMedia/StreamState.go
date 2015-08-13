package liveMedia

import (
	//"fmt"
	. "groupsock"
)

//////// StreamState ////////
type StreamState struct {
	master              IServerMediaSubSession
	rtpSink             IRTPSink
	udpSink             *BasicUDPSink
	rtpGS               *GroupSock
	rtcpGS              *GroupSock
	rtcpInstance        *RTCPInstance
	mediaSource         IFramedSource
	serverRTPPort       uint
	serverRTCPPort      uint
	totalBW             uint
	areCurrentlyPlaying bool
}

func NewStreamState(master IServerMediaSubSession, serverRTPPort, serverRTCPPort uint, rtpSink IRTPSink, udpSink *BasicUDPSink, totalBW uint, mediaSource IFramedSource, rtpGS, rtcpGS *GroupSock) *StreamState {
	streamState := new(StreamState)
	streamState.rtpGS = rtpGS
	streamState.rtcpGS = rtcpGS
	streamState.master = master
	streamState.rtpSink = rtpSink
	streamState.udpSink = udpSink
	streamState.totalBW = totalBW
	streamState.mediaSource = mediaSource
	streamState.serverRTPPort = serverRTPPort
	streamState.serverRTCPPort = serverRTCPPort
	return streamState
}

func (this *StreamState) startPlaying(dests *Destinations) {
	if dests == nil {
		return
	}

	if this.rtcpInstance == nil && this.rtpSink != nil {
		this.rtcpInstance = NewRTCPInstance(this.rtcpGS, this.totalBW, this.master.CNAME())
	}

	if dests.isTCP {
		if this.rtpSink != nil {
			//this.rtpSink.addStreamSocket(dests.tcpSocketNum, dests.rtpChannelId)
			//this.rtpSink.setServerRequestAlternativeByteHandler(dests.tcpSocketNum, serverRequestAlternativeByteHandler, serverRequestAlternativeByteHandlerClientData)
		}
		if this.rtcpInstance != nil {
			this.rtcpInstance.setSpecificRRHandler()
		}
	} else {
		// Tell the RTP and RTCP 'groupsocks' about this destination
		// (in case they don't already have it):
		if this.rtpGS != nil {
			this.rtpGS.AddDestination(dests.addr, dests.rtpPort)
		}
		if this.rtcpGS != nil {
			this.rtcpGS.AddDestination(dests.addr, dests.rtcpPort)
		}
		if this.rtcpInstance != nil {
			//rtcpRRHandler := ""
			//rtcpRRHandlerClientData := ""
			//this.rtcpInstance.setSpecificRRHandler(dests.addr, dests.rtcpPort, rtcpRRHandler, rtcpRRHandlerClientData)
		}
	}

	if !this.areCurrentlyPlaying && this.mediaSource != nil {
		if this.rtpSink != nil {
			this.rtpSink.startPlaying(this.mediaSource)
			this.areCurrentlyPlaying = true
		} else if this.udpSink != nil {
			this.areCurrentlyPlaying = true
			this.udpSink.startPlaying(this.mediaSource)
		}
	}
}

func (this *StreamState) pause() {
	if this.rtpSink != nil {
		this.rtpSink.stopPlaying()
	}
	if this.udpSink != nil {
		this.udpSink.stopPlaying()
	}
	this.areCurrentlyPlaying = false
}

func (this *StreamState) endPlaying(dests *Destinations) {
	if this.rtpSink != nil {
		//this.rtpSink.removeStreamSocket()
	}
	if this.rtcpInstance != nil {
		this.rtcpInstance.unsetSpecificRRHandler()
	}
}

func (this *StreamState) ServerRTPPort() uint {
	return this.serverRTPPort
}

func (this *StreamState) ServerRTCPPort() uint {
	return this.serverRTCPPort
}

func (this *StreamState) afterPlayingStreamState() {
	this.reclaim()
}

func (this *StreamState) reclaim() {
}

func (this *StreamState) RtpSink() IRTPSink {
	return this.rtpSink
}
