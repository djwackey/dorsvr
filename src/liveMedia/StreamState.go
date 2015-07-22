package liveMedia

import (
	//"fmt"
	. "groupsock"
)

type StreamState struct {
	rtpSink        IRTPSink
	udpSink        *BasicUDPSink
	rtpGS          *GroupSock
	rtcpGS         *GroupSock
	rtcpInstance   *RTCPInstance
	mediaSource    IFramedSource
	serverRTPPort  uint
	serverRTCPPort uint
	totalBW        int
}

func NewStreamState(serverRTPPort, serverRTCPPort uint, rtpSink IRTPSink, udpSink *BasicUDPSink, totalBW int, mediaSource IFramedSource, rtpGS, rtcpGS *GroupSock) *StreamState {
	streamState := new(StreamState)
	streamState.rtpGS = rtpGS
	streamState.rtcpGS = rtcpGS
	streamState.rtpSink = rtpSink
	streamState.udpSink = udpSink
	streamState.totalBW = totalBW
	streamState.mediaSource = mediaSource
	streamState.serverRTPPort = serverRTPPort
	streamState.serverRTCPPort = serverRTCPPort
	return streamState
}

func (this *StreamState) startPlaying() {
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
