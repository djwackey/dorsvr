package liveMedia

import (
  "fmt"
  "testing"
)

var SDP_DESC = `v=0\r\n
                o=- 1464450493310666 1 IN IP4 192.168.1.105\r\n
                s=H.264 Video, streamed by the Dor Media Server\r\n
                i=test.264\r\n
                t=0 0\r\n
                a=tool:Dor Streaming Media v2012.10.01\r\n
                a=type:broadcast\r\n
                a=control:*\r\n
                a=range:npt=0-\r\n
                a=x-qt-text-nam:H.264 Video, streamed by the Dor Media Server\r\n
                a=x-qt-text-inf:test.264\r\n
                m=video 0 RTP/AVP 96\r\n
                c=IN IP4 0.0.0.0\r\n
                b=AS:500\r\n
                a=rtpmap:96 H264/90000\r\n
                a=control:track1\r\n\r\n`

func Test_InitWithSDP(t *testing.T) {
  mediaSession := new(MediaSession)
  ok := mediaSession.InitWithSDP(SDP_DESC)
  if !ok {
    t.Log("success")
  } else {
    t.Log("failed")
  }
}
