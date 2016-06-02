### Golang Streaming Server [![Build Status](https://travis-ci.org/djwackey/dorsvr.svg?branch=master)](https://travis-ci.org/djwackey/dorsvr)
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


#### DESCRIBE ####
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
s=H.264 Video, streamed by the LIVE555 Media Server
i=test.264
t=0 0
a=tool:LIVE555 Streaming Media v2012.10.01
a=type:broadcast
a=control:*
a=range:npt=0-
a=x-qt-text-nam:H.264 Video, streamed by the LIVE555 Media Server
a=x-qt-text-inf:test.264
m=video 0 RTP/AVP 96
c=IN IP4 0.0.0.0
b=AS:500
a=rtpmap:96 H264/90000
a=control:track1
```
