package liveMedia

import (
	"fmt"
	. "groupsock"
	"net"
)

type RTSPClientConnection struct {
	clientOutputSocket net.Conn
	currentCSeq        string
	responseBuffer     string
	rtspServer         *RTSPServer
}

func NewRTSPClientConnection(socket net.Conn) *RTSPClientConnection {
	return &RTSPClientConnection{clientOutputSocket: socket}
}

func (this *RTSPClientConnection) IncomingRequestHandler() {
	buffer := make([]byte, 1024)
	isclose := false
	for {
		length, err := this.clientOutputSocket.Read(buffer[:1024])

		switch err {
		case nil:
			this.HandleRequestBytes(buffer, length)
		default:
			//err := conn.Close()
			fmt.Println(err.Error())
			if err.Error() == "EOF" {
				isclose = true
			}
		}

		fmt.Println(this.responseBuffer)

		sendBytes, err := this.clientOutputSocket.Write([]byte(this.responseBuffer))
		if err != nil {
			fmt.Println("failed to send response buffer.", sendBytes)
		}

		if isclose {
			break
		}
	}

	fmt.Println("end connection.")
	this.clientOutputSocket.Close()
}

func (this *RTSPClientConnection) HandleRequestBytes(buf []byte, len int) {
	fmt.Println("HandleRequestBytes", string(buf[:len]))
	requestString, parseSucceeded := ParseRTSPRequestString(buf, len)
	if parseSucceeded {
		this.currentCSeq = "2"
		switch requestString.cmdName {
		case "OPTIONS":
			this.handleCommandOptions()
		case "DESCRIBE":
			this.handleCommandDescribe(string(buf))
		case "SETUP":
			{
				sessionId := OurRandom32()
				clientSession := this.NewClientSession(sessionId)
				clientSession.HandleCommandSetup()
			}
		case "PLAY", "PAUSE", "TEARDOWN", "GET_PARAMETER", "SET_PARAMETER":
			{
			}
		case "RECORD":
		default:
			this.handleCommandNotSupported()
		}
	}
}

func (this *RTSPClientConnection) handleCommandOptions() {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\nCSeq: %s\r\n%sPublic: %s\r\n\r\n", this.currentCSeq, DateHeader(), allowedCommandNames)
}

func (this *RTSPClientConnection) handleCommandNotFound() {
	this.setRTSPResponse("404 Stream Not Found")
}

func (this *RTSPClientConnection) handleCommandSessionNotFound() {
	this.setRTSPResponse("454 Session Not Found")
}

func (this *RTSPClientConnection) handleCommandUnsupportedTransport() {
	this.setRTSPResponse("461 Unsupported Transport")
}

func (this *RTSPClientConnection) handleCommandDescribe(fullRequestStr string) {
	var urlTotalSuffix string

	this.AuthenticationOK("DESCRIPE", urlTotalSuffix, fullRequestStr)

	var session *ServerMediaSession
	session = this.rtspServer.LookupServerMediaSession(urlTotalSuffix)
	if session == nil {
		this.handleCommandNotFound()
		return
	}
	sdpDescription := session.GenerateSDPDescription()
	if len(sdpDescription) <= 0 {
		this.setRTSPResponse("404 File Not Found, Or In Incorrect Format")
		return
	}

	streamName := session.StreamName()
	rtspURL := this.rtspServer.RtspURL(streamName)
	var sdpDescriptionSize int
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\nCSeq: %s\r\n"+
		"%s"+
		"Content-Base: %s/\r\n"+
		"Content-Type: application/sdp\r\n"+
		"Content-Length: %d\r\n\r\n"+
		"%s",
		this.currentCSeq, DateHeader(), rtspURL, sdpDescriptionSize, sdpDescription)
}

func (this *RTSPClientConnection) handleCommandNotSupported() {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 405 Method Not Allowed\r\nCSeq: %s\r\n%sAllow: %s\r\n\r\n", this.currentCSeq, DateHeader(), allowedCommandNames)
}

func (this *RTSPClientConnection) setRTSPResponse(responseStr string) {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n",
		responseStr, this.currentCSeq, DateHeader())
}

func (this *RTSPClientConnection) AuthenticationOK(cmdName string, urlSuffix string, fullRequestStr string) bool {
	return true
}

func (this *RTSPClientConnection) NewClientSession(sessionId uint32) *RTSPClientSession {
	return NewRTSPClientSession()
}
