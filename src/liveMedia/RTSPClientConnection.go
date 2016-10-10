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
	buffer := make([]byte, 4096)
	for {
		length, err := this.clientOutputSocket.Read(buffer)

		switch err {
		case nil:
			this.handleRequestBytes(buffer, length)
		default:
			if err.Error() == "EOF" {
				isclose = true
			}
		}

		if isclose {
			break
		}
	}

	//delete(this.rtspServer.clientSessions, this.sessionIDStr)
	fmt.Println("end connection.")
}

func (this *RTSPClientConnection) handleRequestBytes(buffer []byte, length int) {
	reqStr := string(buffer)

	fmt.Println("[---HandleRequestBytes---]")
	fmt.Println(reqStr[:length])

	var existed bool
	var clientSession *RTSPClientSession
	requestString, parseSucceeded := ParseRTSPRequestString(reqStr, length)
	if parseSucceeded {
		this.currentCSeq = requestString.cseq
		sessionIDStr := requestString.sessionIDStr
		switch requestString.cmdName {
		case "OPTIONS":
			this.handleCommandOptions()
		case "DESCRIBE":
			this.handleCommandDescribe(requestString.urlPreSuffix, requestString.urlSuffix, reqStr)
		case "SETUP":
			{
				if sessionIDStr == "" {
					var sessionID uint
					for {
						sessionID = OurRandom32()
						sessionIDStr = fmt.Sprintf("%08X", sessionID)

						if _, existed = this.rtspServer.clientSessions[sessionIDStr]; !existed {
							break
						}
					}
					clientSession = this.NewClientSession(sessionID)
					this.rtspServer.clientSessions[sessionIDStr] = clientSession
				} else {
					if clientSession, existed = this.rtspServer.clientSessions[sessionIDStr]; !existed {
						this.handleCommandSessionNotFound()
					}
				}

				if clientSession != nil {
					clientSession.HandleCommandSetup(requestString.urlPreSuffix, requestString.urlSuffix, reqStr)
				}
			}
		case "PLAY", "PAUSE", "TEARDOWN", "GET_PARAMETER", "SET_PARAMETER":
			{
				if clientSession, existed = this.rtspServer.clientSessions[sessionIDStr]; existed {
					clientSession.handleCommandWithinSession(requestString.cmdName,
						requestString.urlPreSuffix, requestString.urlSuffix, reqStr)
				} else {
					this.handleCommandSessionNotFound()
				}
			}
		case "RECORD":
		default:
			this.handleCommandNotSupported()
		}
	} else {
		requestString, parseSucceeded := ParseHTTPRequestString(reqStr, length)
		if parseSucceeded {
			switch requestString.cmdName {
			case "GET":
				this.handleHTTPCommandTunnelingGET(requestString.sessionCookie)
			case "POST":
				extraData := ""
				extraDataSize := uint(0)
				this.handleHTTPCommandTunnelingPOST(requestString.sessionCookie, extraData, extraDataSize)
			default:
				this.handleHTTPCommandNotSupported()
			}
		} else {
			this.handleCommandBad()
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
	if urlPreSuffix != "" {
		urlTotalSuffix = fmt.Sprintf("%s/%s", urlPreSuffix, urlSuffix)
	}

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
	// Don't do anything with "currentCSeq", because it might be nonsense
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 400 Bad Request\r\n"+
		"%sAllow: %s\r\n\r\n", DateHeader(), allowedCommandNames)
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
	if _, existed := this.rtspServer.clientConnectionsForHTTPTunneling[sessionCookie]; !existed {
		this.rtspServer.clientConnectionsForHTTPTunneling[sessionCookie] = this
	}

	// Construct our response:
	this.responseBuffer = fmt.Sprintf("HTTP/1.0 200 OK\r\n" +
		"Date: Thu, 19 Aug 1982 18:30:00 GMT\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Pragma: no-cache\r\n" +
		"Content-Type: application/x-rtsp-tunnelled\r\n\r\n")
}

func (this *RTSPClientConnection) handleHTTPCommandTunnelingPOST(sessionCookie, extraData string, extraDataSize uint) {
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

func (this *RTSPClientConnection) setRTSPResponseWithSessionID(responseStr string, sessionID uint) {
	this.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n"+
		"Session: %08X\r\n\r\n",
		responseStr, this.currentCSeq, DateHeader(), sessionID)
}

func (this *RTSPClientConnection) AuthenticationOK(cmdName, urlSuffix, fullRequestStr string) bool {
	return true
}

func (this *RTSPClientConnection) NewClientSession(sessionID uint) *RTSPClientSession {
	return NewRTSPClientSession(this, sessionID)
}
