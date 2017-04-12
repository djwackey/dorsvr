package livemedia

import (
	"fmt"
	"testing"
)

var (
	optionsRequest = "OPTIONS rtsp://172.22.0.172/123.ts RTSP/1.0\r\n" +
		"CSeq: 1\r\n" +
		"User-Agent: LibVLC/2.1.2 (Dor Streaming Media v1.0.0.3))\r\n\r\n"

	descriptionRequest = "DESCRIBE rtsp://192.168.1.103/live1.264 RTSP/1.0\r\n" +
		"CSeq: 2\r\n" +
		"User-Agent: LibVLC/2.1.5 (Dor Streaming Media v1.0.0.3))\r\n" +
		"Accept: application/sdp\r\n\r\n"

	setupRequest = "SETUP rtsp://192.168.1.105:8554/test.264/track1 RTSP/1.0\r\n" +
		"CSeq: 3\r\n" +
		"User-Agent: dorsvr (Dor Streaming Media v1.0.0.3)\r\n" +
		"Transport: RTP/AVP;unicast;client_port=37175-37176\r\n\r\n"

	playRequest = "PLAY rtsp://192.168.1.105:8554/test.264/ RTSP/1.0\r\n" +
		"CSeq: 4\r\n" +
		"User-Agent: dorsvr (Dor Streaming Media v1.0.0.3)\r\n" +
		"Session: E1155C20\r\n" +
		"Range: npt=0.000-\r\n"

	teardownRequest = "TEARDOWN rtsp://192.168.1.105:8554/test.264 RTSP/1.0\r\n" +
		"CSeq: 5\r\n" +
		"Session: E1155C20\r\n" +
		"User-Agent: VLC media player (Dor Streaming Media v1.0.0.3))"
)

func TestParseRTSPRequestString(t *testing.T) {
	var verify bool = true

	cmdList := []string{"OPTIONS", "DESCRIBE", "SETUP", "PLAY", "TEARDOWN"}
	reqList := []string{optionsRequest, descriptionRequest, setupRequest, playRequest, teardownRequest}
	for i, req := range reqList {
		reqStr, ok := ParseRTSPRequestString(req, len(req))
		if !ok {
			break
		}

		// check request command
		if reqStr.CmdName != cmdList[i] {
			verify = false
			break
		}

		// check cseq
		if reqStr.Cseq != fmt.Sprintf("%d", i+1) {
			fmt.Println("parse cseq error", reqStr.Cseq, i+1)
			verify = false
			break
		}

		// check session id
		if reqStr.CmdName == "PLAY" || reqStr.CmdName == "TEARDOWN" {
			var sessionID string = "E1155C20"
			if reqStr.SessionIDStr != sessionID {
				fmt.Println("parse session id error", reqStr.Cseq, i+1)
				verify = false
				break
			}
		}

		// check content length
		fmt.Printf("CommandName: %s\n", reqStr.CmdName)
		fmt.Printf("ContentLength: %s\n", reqStr.ContentLength)
		// check url presuffix and suffix
		fmt.Printf("UrlPreSuffix: %s\n", reqStr.UrlPreSuffix)
		fmt.Printf("UrlSuffix: %s\n\n", reqStr.UrlSuffix)
	}
	if verify {
		t.Log("success")
	} else {
		t.Error("failed")
	}
}
