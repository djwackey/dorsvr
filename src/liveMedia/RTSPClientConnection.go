package liveMedia

import (
	"fmt"
	. "groupsock"
	"net"
	"strings"
)

type RTSPClientConnection struct {
	clientOutputSocket net.Conn
	localPort          string
	remotePort         string
	localAddr          string
	remoteAddr         string
	currentCSeq        string
	responseBuffer     string
	rtspServer         *RTSPServer
}

func NewRTSPClientConnection(rtspServer *RTSPServer, socket net.Conn) *RTSPClientConnection {
	rtspClientConn := new(RTSPClientConnection)
	rtspClientConn.rtspServer = rtspServer
	rtspClientConn.clientOutputSocket = socket
	localAddr := strings.Split(fmt.Sprintf("%s", socket.LocalAddr()), ":")
	remoteAddr := strings.Split(fmt.Sprintf("%s", socket.RemoteAddr()), ":")
	rtspClientConn.localAddr = localAddr[0]
	rtspClientConn.localPort = localAddr[1]
	rtspClientConn.remoteAddr = remoteAddr[0]
	rtspClientConn.remotePort = remoteAddr[1]
	return rtspClientConn
}

func (this *RTSPClientConnection) GetRTSPServer() *RTSPServer {
	return this.rtspServer
}

func (this *RTSPClientConnection) IncomingRequestHandler() {
	defer this.clientOutputSocket.Close()

	isclose := false
	buffer := make([]byte, 1024)
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

	//delete(this.rtspServer.clientSessions, this.sessionIdStr)
	fmt.Println("end connection.")
}

func (this *RTSPClientConnection) HandleRequestBytes(buf []byte, length int) {
	reqStr := string(buf)

	fmt.Println("HandleRequestBytes", reqStr[:length])

	var existed bool
	var clientSession *RTSPClientSession
	requestString, parseSucceeded := ParseRTSPRequestString(reqStr, length)
	if parseSucceeded {
		this.currentCSeq = requestString.cseq
		sessionIdStr := requestString.sessionIdStr
		switch requestString.cmdName {
		case "OPTIONS":
			this.handleCommandOptions()
		case "DESCRIBE":
			this.handleCommandDescribe(requestString.urlPreSuffix, requestString.urlSuffix, reqStr)
		case "SETUP":
			{
				if sessionIdStr == "" {
					var sessionId uint
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
					clientSession.HandleCommandSetup(requestString.urlPreSuffix, requestString.urlSuffix, reqStr)
				}
			}
		case "PLAY", "PAUSE", "TEARDOWN", "GET_PARAMETER", "SET_PARAMETER":
			{
				if clientSession, existed = this.rtspServer.clientSessions[sessionIdStr]; existed {
					clientSession.HandleCommandWithinSession(requestString.cmdName, requestString.urlPreSuffix, requestString.urlSuffix, reqStr)
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
				this.handleHTTPCommandTunnelingGET(requestString.sessionCookie)
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
	//fmt.Println("handleCommandDescribe", urlTotalSuffix)

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
		"Content-Base: %s/\r\n"+
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

func (this *RTSPClientConnection) handleHTTPCommandNotSupported() {
	this.responseBuffer = fmt.Sprintf("HTTP/1.0 405 Method Not Allowed\r\n%s\r\n\r\n", DateHeader())
}

func (this *RTSPClientConnection) handleHTTPCommandNotFound() {
	this.responseBuffer = fmt.Sprintf("HTTP/1.0 404 Not Found\r\n%s\r\n\r\n", DateHeader())
}

func (this *RTSPClientConnection) handleHTTPCommandTunnelingGET(sessionCookie string) {
	// Construct our response:
	this.responseBuffer = fmt.Sprintf("HTTP/1.0 200 OK\r\n" +
		"Date: Thu, 19 Aug 1982 18:30:00 GMT\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Pragma: no-cache\r\n" +
		"Content-Type: application/x-rtsp-tunnelled\r\n" +
		"\r\n")
}

func (this *RTSPClientConnection) handleHTTPCommandTunnelingPOST() {
}

func (this *RTSPClientConnection) handleHTTPCommandStreamingGET(urlSuffix, fullRequestStr string) {
	// By default, we don't support requests to access streams via HTTP:
	this.handleHTTPCommandNotSupported()
}

func (this *RTSPClientConnection) setRTSPResponse(responseStr string) {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n",
		responseStr, this.currentCSeq, DateHeader())
}

func (this *RTSPClientConnection) setRTSPResponseWithSessionId(responseStr string, sessionId uint) {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n"+
		"Session: %08X\r\n\r\n",
		responseStr, this.currentCSeq, DateHeader(), sessionId)
}

func (this *RTSPClientConnection) AuthenticationOK(cmdName, urlSuffix, fullRequestStr string) bool {
	return true
}

func (this *RTSPClientConnection) NewClientSession(sessionId uint) *RTSPClientSession {
	return NewRTSPClientSession(this, sessionId)
}
