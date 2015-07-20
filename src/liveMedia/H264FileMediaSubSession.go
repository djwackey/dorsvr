package liveMedia

import (
	//"fmt"
	. "groupsock"
)

type H264FileMediaSubSession struct {
	FileServerMediaSubSession
}

func NewH264FileMediaSubSession(fileName string) *H264FileMediaSubSession {
	h264FileMediaSubSession := new(H264FileMediaSubSession)
	h264FileMediaSubSession.InitFileServerMediaSubSession(h264FileMediaSubSession, fileName)
	return h264FileMediaSubSession
}

func (this *H264FileMediaSubSession) CreateNewStreamSource() IFramedSource {
	//estBitrate = 500 // kbps, estimate

	// Create the video source:
	fileSource := NewByteStreamFileSource(this.fileName)
	if fileSource == nil {
		return nil
	}
	this.fileSize = fileSource.FileSize()

	// Create a framer for the Video Elementary Stream:
	return NewH264VideoStreamFramer(fileSource)
}

func (this *H264FileMediaSubSession) CreateNewRTPSink(rtpGroupSock *GroupSock, rtpPayloadType int) IRTPSink {
	return NewH264VideoRTPSink(rtpGroupSock, rtpPayloadType)
}
