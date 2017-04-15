package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

var TRANSPORT_PACKET_SIZE uint = 188
var TRANSPORT_PACKETS_PER_NETWORK_PACKET uint = 7

type M2TSFileMediaSubsession struct {
	FileServerMediaSubsession
	duration float32
}

func NewM2TSFileMediaSubsession(fileName string) *M2TSFileMediaSubsession {
	subsession := new(M2TSFileMediaSubsession)
	subsession.initFileServerMediaSubsession(subsession, fileName)
	return subsession
}

func (s *M2TSFileMediaSubsession) createNewStreamSource() IFramedSource {
	//inputDataChunkSize := TRANSPORT_PACKETS_PER_NETWORK_PACKET * TRANSPORT_PACKET_SIZE

	// Create the video source:
	fileSource := newByteStreamFileSource(s.fileName)
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

func (s *M2TSFileMediaSubsession) createNewRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint) IMediaSink {
	return newSimpleRTPSink(rtpGroupSock, 33, 90000, 1, "video", "MP2T", true, false)
}

func (s *M2TSFileMediaSubsession) Duration() float32 {
	return s.duration
}
