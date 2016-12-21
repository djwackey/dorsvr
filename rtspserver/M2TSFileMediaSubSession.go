package rtspserver

var TRANSPORT_PACKET_SIZE uint = 188
var TRANSPORT_PACKETS_PER_NETWORK_PACKET uint = 7

type M2TSFileMediaSubSession struct {
	FileServerMediaSubSession
	duration float32
}

func NewM2TSFileMediaSubSession(fileName string) *M2TSFileMediaSubSession {
	m2tsFileMediaSubSession := new(M2TSFileMediaSubSession)
	m2tsFileMediaSubSession.InitFileServerMediaSubSession(m2tsFileMediaSubSession, fileName)
	return m2tsFileMediaSubSession
}

func (this *M2TSFileMediaSubSession) createNewStreamSource() IFramedSource {
	//inputDataChunkSize := TRANSPORT_PACKETS_PER_NETWORK_PACKET * TRANSPORT_PACKET_SIZE

	// Create the video source:
	fileSource := NewByteStreamFileSource(this.fileName)
	if fileSource == nil {
		return nil
	}
	this.fileSize = fileSource.FileSize()

	// Use the file size and the duration to estimate the stream's bitrate:
	//var estBitrate float32 = 5000   // kbps, estimate
	//if this.fileSize > 0 && this.duration > 0.0 {
	//    estBitrate = float32(this.fileSize) / (125 * this.duration) + 0.5  // kbps, rounded
	//}

	// Create a framer for the Transport Stream:
	framer := NewM2TSVideoStreamFramer(fileSource)
	return framer
}

func (this *M2TSFileMediaSubSession) createNewRTPSink(rtpGroupSock *GroupSock, rtpPayloadType uint) IRTPSink {
	return NewSimpleRTPSink(rtpGroupSock, 33, 90000, 1, "video", "MP2T", true, false)
}

func (this *M2TSFileMediaSubSession) Duration() float32 {
	return this.duration
}
