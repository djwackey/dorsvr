package livemedia

import (
	"time"

	gs "github.com/djwackey/dorsvr/groupsock"
	//"github.com/djwackey/dorsvr/log"
)

type H264FileMediaSubsession struct {
	FileServerMediaSubsession
	dummyRTPSink IMediaSink
	auxSDPLine   string
}

func NewH264FileMediaSubsession(fileName string) *H264FileMediaSubsession {
	subsession := new(H264FileMediaSubsession)
	subsession.initFileServerMediaSubsession(subsession, fileName)
	return subsession
}

func (s *H264FileMediaSubsession) createNewStreamSource() IFramedSource {
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

func (s *H264FileMediaSubsession) createNewRTPSink(rtpGroupSock *gs.GroupSock, rtpPayloadType uint) IMediaSink {
	return newH264VideoRTPSink(rtpGroupSock, uint32(rtpPayloadType))
}

func (s *H264FileMediaSubsession) getAuxSDPLine(rtpSink IMediaSink, inputSource IFramedSource) string {
	if s.auxSDPLine != "" {
		return s.auxSDPLine
	}

	if s.dummyRTPSink == nil {
		s.dummyRTPSink = rtpSink

		// start reading the file
		go s.dummyRTPSink.StartPlaying(inputSource, s.afterPlayingDummy)

		s.checkForAuxSDPLine()
	}
	return s.auxSDPLine
}

func (s *H264FileMediaSubsession) checkForAuxSDPLine() {
	var auxSDPLine string
	for s.auxSDPLine == "" {
		if s.dummyRTPSink == nil {
			break
		}

		auxSDPLine = s.dummyRTPSink.AuxSDPLine()
		if auxSDPLine != "" {
			s.auxSDPLine = auxSDPLine
			break
		}

		// delay 100ms
		uSecsToDelay := 100000
		time.Sleep(time.Duration(uSecsToDelay) * time.Microsecond)
	}
}

func (s *H264FileMediaSubsession) afterPlayingDummy() {
}
