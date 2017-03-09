package rtspserver

import (
	"fmt"
	"net"
	"strings"

	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/dorsvr/livemedia"
)

const RTSP_BUFFER_SIZE = 10000

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

func newRTSPClientConnection(s *RTSPServer, socket net.Conn) *RTSPClientConnection {
	connection := new(RTSPClientConnection)
	connection.rtspServer = s
	connection.clientOutputSocket = socket
	localAddr := strings.Split(socket.LocalAddr().String(), ":")
	remoteAddr := strings.Split(socket.RemoteAddr().String(), ":")
	connection.localAddr = localAddr[0]
	connection.localPort = localAddr[1]
	connection.remoteAddr = remoteAddr[0]
	connection.remotePort = remoteAddr[1]
	return connection
}

func (c *RTSPClientConnection) incomingRequestHandler() {
	defer c.clientOutputSocket.Close()

	var isclose bool
	buffer := make([]byte, RTSP_BUFFER_SIZE)
	for {
		length, err := c.clientOutputSocket.Read(buffer)

		switch err {
		case nil:
			ok := c.handleRequestBytes(buffer, length)
			if !ok {
				isclose = true
			}
		default:
			if err.Error() == "EOF" {
				isclose = true
			}
		}

		if isclose {
			break
		}
	}

	//delete(c.rtspServer.clientSessions, c.sessionIDStr)
	fmt.Println("end connection.")
}

func (c *RTSPClientConnection) handleRequestBytes(buffer []byte, length int) bool {
	reqStr := string(buffer[:length])

	fmt.Printf("Received %d new bytes of request data.\n", length)

	var existed bool
	var clientSession *RTSPClientSession
	requestString, parseSucceeded := livemedia.ParseRTSPRequestString(reqStr, length)
	if parseSucceeded {
		fmt.Printf("Received a complete %s request:\n%s\n", requestString.CmdName, reqStr)

		c.currentCSeq = requestString.Cseq
		sessionIDStr := requestString.SessionIDStr
		switch requestString.CmdName {
		case "OPTIONS":
			c.handleCommandOptions()
		case "DESCRIBE":
			c.handleCommandDescribe(requestString.UrlPreSuffix, requestString.UrlSuffix, reqStr)
		case "SETUP":
			{
				if sessionIDStr == "" {
					var sessionID uint
					for {
						sessionID = uint(gs.OurRandom32())
						sessionIDStr = fmt.Sprintf("%08X", sessionID)

						if _, existed = c.rtspServer.clientSessions[sessionIDStr]; !existed {
							break
						}
					}
					clientSession = c.newClientSession(sessionID)
					c.rtspServer.clientSessions[sessionIDStr] = clientSession
				} else {
					if clientSession, existed = c.rtspServer.clientSessions[sessionIDStr]; !existed {
						c.handleCommandSessionNotFound()
					}
				}

				if clientSession != nil {
					clientSession.handleCommandSetup(requestString.UrlPreSuffix, requestString.UrlSuffix, reqStr)
				}
			}
		case "PLAY", "PAUSE", "TEARDOWN", "GET_PARAMETER", "SET_PARAMETER":
			{
				if clientSession, existed = c.rtspServer.clientSessions[sessionIDStr]; existed {
					clientSession.handleCommandWithinSession(requestString.CmdName,
						requestString.UrlPreSuffix, requestString.UrlSuffix, reqStr)
				} else {
					c.handleCommandSessionNotFound()
				}
			}
		case "RECORD":
		default:
			c.handleCommandNotSupported()
		}
	} else {
		requestString, parseSucceeded := livemedia.ParseHTTPRequestString(reqStr, length)
		if parseSucceeded {
			switch requestString.CmdName {
			case "GET":
				c.handleHTTPCommandTunnelingGET(requestString.SessionCookie)
			case "POST":
				extraData := ""
				extraDataSize := uint(0)
				c.handleHTTPCommandTunnelingPOST(requestString.SessionCookie, extraData, extraDataSize)
			default:
				c.handleHTTPCommandNotSupported()
			}
		} else {
			c.handleCommandBad()
		}
	}

	sendBytes, err := c.clientOutputSocket.Write([]byte(c.responseBuffer))
	if err != nil {
		fmt.Println("failed to send response buffer.", sendBytes)
		return false
	}
	return true
}

func (c *RTSPClientConnection) handleCommandOptions() {
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
		"CSeq: %s\r\n"+
		"%sPublic: %s\r\n\r\n",
		c.currentCSeq, livemedia.DateHeader(), livemedia.AllowedCommandNames)
}

func (c *RTSPClientConnection) handleCommandGetParameter() {
	c.setRTSPResponse("200 OK")
}

func (c *RTSPClientConnection) handleCommandSetParameter() {
	c.setRTSPResponse("200 OK")
}

func (c *RTSPClientConnection) handleCommandNotFound() {
	c.setRTSPResponse("404 Stream Not Found")
}

func (c *RTSPClientConnection) handleCommandSessionNotFound() {
	c.setRTSPResponse("454 Session Not Found")
}

func (c *RTSPClientConnection) handleCommandUnsupportedTransport() {
	c.setRTSPResponse("461 Unsupported Transport")
}

func (c *RTSPClientConnection) handleAlternativeRequestByte(requestByte uint) {
	var buffer []byte
	if requestByte == 0xFF {
		c.handleRequestBytes(buffer, -1)
	} else if requestByte == 0xFE {
	} else {
	}
}

func (c *RTSPClientConnection) handleCommandDescribe(urlPreSuffix, urlSuffix, fullRequestStr string) {
	urlTotalSuffix := urlSuffix
	if urlPreSuffix != "" {
		urlTotalSuffix = fmt.Sprintf("%s/%s", urlPreSuffix, urlSuffix)
	}

	c.AuthenticationOK("DESCRIPE", urlTotalSuffix, fullRequestStr)

	var session *livemedia.ServerMediaSession
	session = c.rtspServer.lookupServerMediaSession(urlTotalSuffix)
	if session == nil {
		c.handleCommandNotFound()
		return
	}

	sdpDescription := session.GenerateSDPDescription()
	sdpDescriptionSize := len(sdpDescription)
	if sdpDescriptionSize <= 0 {
		c.setRTSPResponse("404 File Not Found, Or In Incorrect Format")
		return
	}

	streamName := session.StreamName()
	rtspURL := c.rtspServer.RtspURL(streamName)
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\n"+
		"CSeq: %s\r\n"+
		"%s"+
		"Content-Base: %s/\r\n"+
		"Content-Type: application/sdp\r\n"+
		"Content-Length: %d\r\n\r\n"+
		"%s",
		c.currentCSeq, livemedia.DateHeader(), rtspURL, sdpDescriptionSize, sdpDescription)
}

func (c *RTSPClientConnection) handleCommandBad() {
	// Don't do anything with "currentCSeq", because it might be nonsense
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 400 Bad Request\r\n"+
		"%sAllow: %s\r\n\r\n", livemedia.DateHeader(), livemedia.AllowedCommandNames)
}

func (c *RTSPClientConnection) handleCommandNotSupported() {
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 405 Method Not Allowed\r\n"+
		"CSeq: %s\r\n"+
		"%sAllow: %s\r\n\r\n",
		c.currentCSeq, livemedia.DateHeader(), livemedia.AllowedCommandNames)
}

func (c *RTSPClientConnection) handleHTTPCommandNotSupported() {
	c.responseBuffer = fmt.Sprintf("HTTP/1.0 405 Method Not Allowed\r\n%s\r\n\r\n", livemedia.DateHeader())
}

func (c *RTSPClientConnection) handleHTTPCommandNotFound() {
	c.responseBuffer = fmt.Sprintf("HTTP/1.0 404 Not Found\r\n%s\r\n\r\n", livemedia.DateHeader())
}

func (c *RTSPClientConnection) handleHTTPCommandTunnelingGET(sessionCookie string) {
	if _, existed := c.rtspServer.clientConnectionsForHTTPTunneling[sessionCookie]; !existed {
		c.rtspServer.clientConnectionsForHTTPTunneling[sessionCookie] = c
	}

	// Construct our response:
	c.responseBuffer = fmt.Sprintf("HTTP/1.0 200 OK\r\n" +
		"Date: Thu, 19 Aug 1982 18:30:00 GMT\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Pragma: no-cache\r\n" +
		"Content-Type: application/x-rtsp-tunnelled\r\n\r\n")
}

func (c *RTSPClientConnection) handleHTTPCommandTunnelingPOST(sessionCookie, extraData string, extraDataSize uint) {
}

func (c *RTSPClientConnection) handleHTTPCommandStreamingGET(urlSuffix, fullRequestStr string) {
	// By default, we don't support requests to access streams via HTTP:
	c.handleHTTPCommandNotSupported()
}

func (c *RTSPClientConnection) setRTSPResponse(responseStr string) {
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n",
		responseStr, c.currentCSeq, livemedia.DateHeader())
}

func (c *RTSPClientConnection) setRTSPResponseWithSessionID(responseStr string, sessionID uint) {
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n"+
		"Session: %08X\r\n\r\n",
		responseStr, c.currentCSeq, livemedia.DateHeader(), sessionID)
}

func (c *RTSPClientConnection) AuthenticationOK(cmdName, urlSuffix, fullRequestStr string) bool {
	return true
}

func (c *RTSPClientConnection) newClientSession(sessionID uint) *RTSPClientSession {
	return newRTSPClientSession(c, sessionID)
}
