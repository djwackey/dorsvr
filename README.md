# Golang Streaming Server

[![Build Status](https://travis-ci.org/djwackey/dorsvr.svg?branch=master)](https://travis-ci.org/djwackey/dorsvr) [![GitHub issues](https://img.shields.io/github/issues/djwackey/dorsvr.svg)](https://github.com/djwackey/dorsvr/issues)
### Modules ###
* DorDatabase      - database driver
* DorMediaPlayer   - media player
* DorMediaServer   - media server
* DorProxyServer   - proxy server
* GroupSock        - group socket
* LiveMedia        - live  media
* UsageEnvironment - usage environment

### Compile And Build ###
\> make
### Format ###
\> make fmt
### Testing ###
\> make test
### Inheritance ###
ServerMediaSubSession <- OnDemandServerMediaSubSession <- FileServerMediaSubSession <- H264FileMediaSubSession

FramedSource <- FramedFilter <- MPEGVideoStreamFramer <- H264VideoStreamFramer
             <- FramedFileSource <- ByteStreamFileSource

MediaSink <- RTPSink <- MultiFramedRTPSink <- VideoRTPSink <- H264VideoRTPSink

### Real-Time Streaming Protocol
The Real-Time Streaming Protocol allows to control multimedia streams delivered, for example, via RTP. Control includes absolute positioning within the media stream, recording and possibly device control.
#### OPTIONS
##### REQUEST #####
```
OPTIONS rtsp://192.168.1.1/stream.ts RTSP/1.0
CSeq: 1
User-Agent: LibVLC/2.1.2 (Dor Streaming Media v2016.06.05)
```

#### ANNOUNCD
#### DESCRIBE
##### REQUEST #####
##### RESPONSE #####
```
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
```
### SETUP
### PLAY
### PAUSE
### RECORD
### TEARDOWN
### SET_PARAMETER
### GET_PARAMETER
