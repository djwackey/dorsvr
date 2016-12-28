## Golang Streaming Server ##

[![Build Status](https://travis-ci.org/djwackey/dorsvr.svg?branch=master)](https://travis-ci.org/djwackey/dorsvr) [![GitHub issues](https://img.shields.io/github/issues/djwackey/dorsvr.svg)](https://github.com/djwackey/dorsvr/issues)
### Modules ###
* rtspserver - rtsp server
* rtspclient - rtsp client
* groupsock  - group socket
* scheduler  - task scheduler
* livemedia  - live media handler

### Install ###
	go get github.com/djwackey/dorsvr

### Format ###
	$ make fmt

### Testing ###
	$ make test

### Real-Time Streaming Protocol ###
The Real-Time Streaming Protocol allows to control multimedia streams delivered, for example, via RTP. Control includes absolute positioning within the media stream, recording and possibly device control.

#### OPTIONS ####
**Client -> Server**

	OPTIONS rtsp://192.168.1.105/stream.ts RTSP/1.0
	CSeq: 1
	User-Agent: LibVLC/2.1.2 (Dor Streaming Media v2016.06.05)
**Server -> Client**

	RTSP/1.0 200 OK
    CSeq: 1
    Public: DESCRIBE, SETUP, TEARDOWN, PLAY, PAUSE

#### DESCRIBE ####
**Client -> Server**

	DESCRIBE rtsp://192.168.1.105:8554/test.264 RTSP/1.0
	CSeq: 2
    Accept: application/sdp
    User-Agent: LibVLC/2.1.2 (Dor Streaming Media v2016.06.05)
**Server -> Client**

	RTSP/1.0 200 OK
	CSeq: 2
	Date: Sat, May 28 2016 15:48:13 GMT
	Content-Base: rtsp://192.168.1.105:8554/test.264/
	Content-Type: application/sdp
	Content-Length: 398

	v=0
	o=- 1464450493310666 1 IN IP4 192.168.1.105
	s=H.264 Video, streamed by the Dor Media Server
	i=test.264
	t=0 0
	a=tool:Dor Streaming Media v2012.10.01
	a=type:broadcast
	a=control:*
	a=range:npt=0-
	a=x-qt-text-nam:H.264 Video, streamed by the Dor Media Server
	a=x-qt-text-inf:test.264
	m=video 0 RTP/AVP 96
	c=IN IP4 0.0.0.0
	b=AS:500
	a=rtpmap:96 H264/90000
	a=control:track1

### SETUP ###
**Client -> Server**

	SETUP rtsp://192.168.1.105:8554/test.264/track1 RTSP/1.0
	CSeq: 3
	User-Agent: dorsvr (Dor Streaming Media v1.0.0.3)
	Transport: RTP/AVP;unicast;client_port=37175-37176

**Server -> Client**

	RTSP/1.0 200 OK
	CSeq: 3
	Date: Tue, Oct 18 2016 06:43:05 GMT
	Transport: RTP/AVP;unicast;destination=192.168.1.105;source=192.168.1.105;client_port=37175-37176;server_port=6970-6971
	Session: E1155C20

### PLAY ###
**Client -> Server**

	PLAY rtsp://192.168.1.105:8554/test.264/ RTSP/1.0
	CSeq: 4
	User-Agent: dorsvr (Dor Streaming Media v1.0.0.3)
	Session: E1155C20
	Range: npt=0.000-

**Server -> Client**

	RTSP/1.0 200 OK
	CSeq: 4
	Date: Tue, Oct 18 2016 06:43:05 GMT
	Range: npt=0.000-
	Session: E1155C20
	RTP-Info: url=rtsp://192.168.1.105:8554/test.264/track1;seq=40260;rtptime=3619422277

### PAUSE ###
**Client -> Server**

	PAUSE rtsp://192.168.1.105:8554/test.264 RTSP/1.0
	CSeq: 5
    Session: E1155C20

**Server -> Client**

	RTSP/1.0 200 OK
    CSeq: 5
	Session: E1155C20

### TEARDOWN ###
**Client -> Server**

	TEARDOWN rtsp://192.168.1.105:8554/test.264 RTSP/1.0
	CSeq: 6
	Session: E1155C20
	User-Agent: VLC media player (LIVE555 Streaming Media v2005.11.10)

**Server -> Client**

	RTSP/1.0 200 OK
	CSeq: 6
	Session: E1155C20
	Connection: Close

### SET_PARAMETER ###
**Client -> Server**

**Server -> Client**

### GET_PARAMETER ###
**Client -> Server**

**Server -> Client**

### ANNOUNCD ###
**Client -> Server**

**Server -> Client**

### RECORD ###
**Client -> Server**

**Server -> Client**
