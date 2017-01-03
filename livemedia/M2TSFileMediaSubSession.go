package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

var TRANSPORT_PACKET_SIZE uint = 188
var TRANSPORT_PACKETS_PER_NETWORK_PACKET uint = 7

type M2TSFileMediaSubSession struct {
	FileServerMediaSubSession
	duration float32
}

func NewM2TSFileMediaSubSession(fileName string) *M2TSFileMediaSubSession {
	subsession := new(M2TSFileMediaSubSession)
	subsession.InitFileServerMediaSubSession(subsession, fileName)
	return subsession
}

func (s *M2TSFileMediaSubSession) createNewStreamSource() IFramedSource {
	//inputDataChunkSize := TRANSPORT_PACKETS_PER_NETWORK_PACKET * TRANSPORT_PACKET_SIZE

	// Create the video source:
	fileSource := NewByteStreamFileSource(s.fileName)
	if fileSource == nil {
		return nil
	}
	s.fileSize = fileSource.FileSize()

	// Use the file size and the duration to estimate the stream's bitrate:
	//var estBitrate float32 = 5000   // kbps, estimate
	//if this.fileSize > 0 && this.duration > 0.0 {
	//    estBitrate = float32(this.fileSize) / (125 * this.duration) + 0.5  // kbps, rounded
	//}

	// Create a framer for the Transport Stream:
	framer := NewM2TSVideoStreamFramer(fileSource)
	return framer
}

func (s *M2TSFileMediaSubSession) createNewRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint) IRTPSink {
	return NewSimpleRTPSink(rtpGroupSock, 33, 90000, 1, "video", "MP2T", true, false)
}

func (s *M2TSFileMediaSubSession) Duration() float32 {
	return s.duration
}
