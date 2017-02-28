package livemedia

import gs "github.com/djwackey/dorsvr/groupsock"

type H264FileMediaSubSession struct {
	FileServerMediaSubSession
}

func NewH264FileMediaSubSession(fileName string) *H264FileMediaSubSession {
	subsession := new(H264FileMediaSubSession)
	subsession.InitFileServerMediaSubSession(subsession, fileName)
	return subsession
}

func (s *H264FileMediaSubSession) createNewStreamSource() IFramedSource {
	//estBitrate = 500 // kbps, estimate

	// Create the video source:
	fileSource := newByteStreamFileSource(s.fileName)
	if fileSource == nil {
		return nil
	}
	s.fileSize = fileSource.FileSize()

	// Create a framer for the Video Elementary Stream:
	return newH264VideoStreamFramer(fileSource)
}

func (s *H264FileMediaSubSession) createNewRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint) IRTPSink {
	return NewH264VideoRTPSink(rtpGroupSock, rtpPayloadType)
}
