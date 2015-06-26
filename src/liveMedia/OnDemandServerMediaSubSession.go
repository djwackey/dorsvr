package liveMedia

import (
    "fmt"
    . "groupsock"
}

type OnDemandServerMediaSubSession struct {
    SDPLines string
    trackId string
    trackNumber int
    portNumForSDP int
}

func (this *OnDemandServerMediaSubSession) sdpLines() {
    if !this.SDPLines {
        setSDPLinesFromRTPSink()
    }
}

func (this *OnDemandServerMediaSubSession) getStreamParameters(rtpChannelId, rtcpChannelId uint) {
}

func (this *OnDemandServerMediaSubSession) trackId() string {
    if this.trackId == "" {
        this.trackId = fmt.Sprintf("track%d", this.trackNumber)
    }
    return this.trackId
}

func (this *OnDemandServerMediaSubSession) setSDPLinesFromRTPSink(rtpSink *RTPSink, estBitrate uint) {
    if rtpSink == nil {
        return
    }

    rtpPayloadType := rtpSink->rtpPayloadType()
    mediaType := rtpSink.sdpMediaType()
    rangeLine := rangeSDPLine()

    auxSDPLine := getAuxSDPLine()
    if auxSDPLine == nil {
        auxSDPLine = ""
    }

    sdpFmt := "m=%s %u RTP/AVP %d\r\n" +
              "c=IN IP4 %s\r\n" +
              "b=AS:%u\r\n" +
              "%s" +
              "%s" +
              "%s" +
              "a=control:%s\r\n"

    this.SDPLines = fmt.Sprintf(sdpFmt, mediaType, this.portNumForSDP, rtpPayloadType, OurIPAddress(), estBitrate, rangeLine, auxSDPLine, trackId())
}
