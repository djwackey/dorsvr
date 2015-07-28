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

func NewRTSPClientConnection(rtspServer *RTSPServer, socket net.Conn) *RTSPClientConnection {
	rtspClientConn := new(RTSPClientConnection)
	rtspClientConn.rtspServer = rtspServer
	rtspClientConn.clientOutputSocket = socket
	return rtspClientConn
}

func (this *RTSPClientConnection) GetRTSPServer() *RTSPServer {
	return this.rtspServer
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
			//fmt.Println(err.Error())
			if err.Error() == "EOF" {
				isclose = true
			}
		}

		if isclose {
			break
		}
	}

	fmt.Println("end connection.")
	this.clientOutputSocket.Close()
}

func (this *RTSPClientConnection) HandleRequestBytes(buf []byte, length int) {
	fmt.Println("HandleRequestBytes", string(buf[:length]))

	var existed bool
	var clientSession *RTSPClientSession
	requestString, parseSucceeded := ParseRTSPRequestString(string(buf), length)
	if parseSucceeded {
		this.currentCSeq = requestString.cseq
		sessionIdStr := requestString.sessionIdStr
		fmt.Println(requestString)
		switch requestString.cmdName {
		case "OPTIONS":
			this.handleCommandOptions()
			/*
			   case "GET_PARAMETER":
			       this.handleCommandGetParameter()
			   case "SET_PARAMETER":
			       this.handleCommandSetParameter()
			*/
		case "DESCRIBE":
			this.handleCommandDescribe(requestString.urlPreSuffix, requestString.urlSuffix, string(buf))
		case "SETUP":
			{
				if sessionIdStr == "" {
					var sessionId uint32
					for {
						sessionId = OurRandom32()
						sessionIdStr = fmt.Sprintf("%08X", sessionId)

						if _, existed = this.rtspServer.clientSessions[sessionIdStr]; !existed {
							break
						}
					}
					clientSession = this.NewClientSession(sessionId)
					this.rtspServer.clientSessions[sessionIdStr] = clientSession
				} else {
					if clientSession, existed = this.rtspServer.clientSessions[sessionIdStr]; !existed {
						this.handleCommandSessionNotFound()
					}
				}

				if clientSession != nil {
					clientSession.HandleCommandSetup(requestString.urlPreSuffix, requestString.urlSuffix, string(buf))
				}
			}
		case "PLAY", "PAUSE", "TEARDOWN", "GET_PARAMETER", "SET_PARAMETER":
			{
				if clientSession, existed = this.rtspServer.clientSessions[sessionIdStr]; existed {
					clientSession.HandleCommandWithinSession(requestString.cmdName, requestString.urlPreSuffix, requestString.urlSuffix, string(buf))
				} else {
					this.handleCommandSessionNotFound()
				}
			}
		case "RECORD":
		default:
			this.handleCommandBad()
		}
	} else {
		requestString, parseSucceeded := ParseHTTPRequestString()
		if parseSucceeded {
			switch requestString.cmdName {
			case "GET":
				this.handleHTTPCommandTunnelingGET()
			case "POST":
				this.handleHTTPCommandTunnelingPOST()
			default:
			}
		}
	}

	fmt.Println(this.responseBuffer)

	sendBytes, err := this.clientOutputSocket.Write([]byte(this.responseBuffer))
	if err != nil {
		fmt.Println("failed to send response buffer.", sendBytes)
	}
}

func (this *RTSPClientConnection) handleCommandOptions() {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
		"CSeq: %s\r\n"+
		"%sPublic: %s\r\n\r\n",
		this.currentCSeq, DateHeader(), allowedCommandNames)
}

func (this *RTSPClientConnection) handleCommandGetParameter() {
	this.setRTSPResponse("200 OK")
}

func (this *RTSPClientConnection) handleCommandSetParameter() {
	this.setRTSPResponse("200 OK")
}

func (this *RTSPClientConnection) handleCommandNotFound() {
	this.setRTSPResponse("404 Stream Not Found")
}

func (this *RTSPClientConnection) handleCommandSessionNotFound() {
	this.setRTSPResponse("454 Session Not Found")
}

func (this *RTSPClientConnection) HandleCommandUnsupportedTransport() {
	this.setRTSPResponse("461 Unsupported Transport")
}

func (this *RTSPClientConnection) handleCommandDescribe(urlPreSuffix, urlSuffix, fullRequestStr string) {
	urlTotalSuffix := urlSuffix
	fmt.Println("handleCommandDescribe", urlTotalSuffix)

	this.AuthenticationOK("DESCRIPE", urlTotalSuffix, fullRequestStr)

	var session *ServerMediaSession
	session = this.rtspServer.LookupServerMediaSession(urlTotalSuffix)
	if session == nil {
		this.handleCommandNotFound()
		return
	}

	sdpDescription := session.GenerateSDPDescription()
	sdpDescriptionSize := len(sdpDescription)
	if sdpDescriptionSize <= 0 {
		this.setRTSPResponse("404 File Not Found, Or In Incorrect Format")
		return
	}

	streamName := session.StreamName()
	rtspURL := this.rtspServer.RtspURL(streamName)
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
		"CSeq: %s\r\n"+
		"%s"+
		"Content-Base: %s\r\n"+
		"Content-Type: application/sdp\r\n"+
		"Content-Length: %d\r\n\r\n"+
		"%s",
		this.currentCSeq, DateHeader(), rtspURL, sdpDescriptionSize, sdpDescription)
}

func (this *RTSPClientConnection) handleCommandBad() {
	// Don't do anything with "fCurrentCSeq", because it might be nonsense
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 400 Bad Request\r\n"+
		"%sAllow: %s\r\n\r\n",
		DateHeader(), allowedCommandNames)
}

func (this *RTSPClientConnection) handleCommandNotSupported() {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 405 Method Not Allowed\r\n"+
		"CSeq: %s\r\n"+
		"%sAllow: %s\r\n\r\n",
		this.currentCSeq, DateHeader(), allowedCommandNames)
}

func (this *RTSPClientConnection) handleHTTPCommandTunnelingGET() {
}

func (this *RTSPClientConnection) handleHTTPCommandTunnelingPOST() {
}

func (this *RTSPClientConnection) setRTSPResponse(responseStr string) {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n",
		responseStr, this.currentCSeq, DateHeader())
}

func (this *RTSPClientConnection) setRTSPResponseWithSessionId(responseStr string, sessionId uint32) {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n"+
		"Session: %08X\r\n\r\n",
		responseStr, this.currentCSeq, DateHeader(), sessionId)
}

func (this *RTSPClientConnection) AuthenticationOK(cmdName, urlSuffix, fullRequestStr string) bool {
	return true
}

func (this *RTSPClientConnection) NewClientSession(sessionId uint32) *RTSPClientSession {
	return NewRTSPClientSession(this, sessionId)
}
