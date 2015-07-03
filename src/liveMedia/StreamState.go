package liveMedia

import (
    . "groupsock"
)

type StreamState struct {
    rtpSink *RTPSink
    udpSink *BasicUDPSink
    rtpGS GroupSock
    rtcpGS GroupSock
    rtcpInstance RTCPInstance
    serverRTPPort int
    serverRTCPPort int
}

func NewStreamState() *StreamState {
    return &StreamState{}
}

func (this *StreamState) startPlaying() {
    if this.rtpSink != nil {
        this.rtpSink.startPlaying()
    } else if this.udpSink != nil {
        this.udpSink.startPlaying()
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
