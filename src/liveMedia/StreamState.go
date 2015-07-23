package liveMedia

import (
	//"fmt"
	. "groupsock"
)

//////// StreamState ////////
type StreamState struct {
	master         IServerMediaSubSession
	rtpSink        IRTPSink
	udpSink        *BasicUDPSink
	rtpGS          *GroupSock
	rtcpGS         *GroupSock
	rtcpInstance   *RTCPInstance
	mediaSource    IFramedSource
	serverRTPPort  uint
	serverRTCPPort uint
	totalBW        uint
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

func (this *StreamState) startPlaying() {
	if this.rtcpInstance == nil && this.rtpSink != nil {
		this.rtcpInstance = NewRTCPInstance(this.rtcpGS, this.totalBW, this.master.CNAME())
	}

	//if dests.isTCP() {
	//    if this.rtcpInstance != nil {
	//    }
	//} else {
	//}

	if this.rtcpInstance != nil {
		this.rtcpInstance.setSpecificRRHandler()
	}

	if this.rtpSink != nil {
		this.rtpSink.startPlaying(this.mediaSource)
	} else if this.udpSink != nil {
		this.udpSink.startPlaying(this.mediaSource)
	}
}

func (this *StreamState) pause() {
	if this.rtpSink != nil {
		this.rtpSink.stopPlaying()
	}
	if this.udpSink != nil {
		this.udpSink.stopPlaying()
	}
}

func (this *StreamState) endPlaying() {
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

//////// Destinations ////////
type Destinations struct {
	isTCP         bool
	rtpPort       int
	rtcpPort      int
	rtpChannelId  uint
	rtcpChannelId uint
}

func NewDestinations(rtpDestPort, rtcpDestPort int) *Destinations {
	dests := new(Destinations)
	dests.isTCP = false
	dests.rtpPort = rtpDestPort
	dests.rtcpPort = rtcpDestPort
	return dests
}
