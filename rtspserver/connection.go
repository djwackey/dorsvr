package rtspserver

import (
	"fmt"
	"net"
	"strings"

	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/dorsvr/livemedia"
	"github.com/djwackey/dorsvr/log"
)

const rtspBufferSize = 10000

type RTSPClientConnection struct {
	clientSocket   net.Conn
	localPort      string
	remotePort     string
	localAddr      string
	remoteAddr     string
	currentCSeq    string
	responseBuffer string
	rtspServer     *RTSPServer
}

func newRTSPClientConnection(s *RTSPServer, socket net.Conn) *RTSPClientConnection {
	localAddr := strings.Split(socket.LocalAddr().String(), ":")
	remoteAddr := strings.Split(socket.RemoteAddr().String(), ":")
	return &RTSPClientConnection{
		rtspServer:   s,
		localAddr:    localAddr[0],
		localPort:    localAddr[1],
		remoteAddr:   remoteAddr[0],
		remotePort:   remoteAddr[1],
		clientSocket: socket,
	}
}

func (c *RTSPClientConnection) incomingRequestHandler() {
	defer c.clientSocket.Close()

	var isclose bool
	buffer := make([]byte, rtspBufferSize)
	for {
		length, err := c.clientSocket.Read(buffer)

		switch err {
		case nil:
			err = c.handleRequestBytes(buffer, length)
			if err != nil {
				isclose = true
			}
		default:
			log.Info("default: %s", err.Error())
			if err.Error() == "EOF" {
				isclose = true
			}
		}

		if isclose {
			break
		}
	}

	log.Info("end connection.")
}

func (c *RTSPClientConnection) handleRequestBytes(buffer []byte, length int) error {
	reqStr := string(buffer[:length])

	log.Info("Received %d new bytes of request data.", length)

	var existed bool
	var clientSession *RTSPClientSession
	requestString, parseSucceeded := livemedia.ParseRTSPRequestString(reqStr, length)
	if parseSucceeded {
		log.Info("Received a complete %s request:\n%s", requestString.CmdName, reqStr)

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
					for {
						sessionIDStr = fmt.Sprintf("%08X", gs.OurRandom32())
						if _, existed = c.rtspServer.clientSessions[sessionIDStr]; !existed {
							break
						}
					}
					clientSession = c.newClientSession(sessionIDStr)
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

	sendBytes, err := c.clientSocket.Write([]byte(c.responseBuffer))
	if err != nil {
		log.Error(0, "failed to send response buffer.", sendBytes)
		return err
	}
	log.Info("send response:\n%s", c.responseBuffer)
	return nil
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

	if ok := c.authenticationOK("DESCRIPE", urlTotalSuffix, fullRequestStr); !ok {
		return
	}

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

// Don't do anything with "currentCSeq", because it might be nonsense
func (c *RTSPClientConnection) handleCommandBad() {
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
	if _, existed := c.rtspServer.clientHttpConnections[sessionCookie]; !existed {
		c.rtspServer.clientHttpConnections[sessionCookie] = c
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

// By default, we don't support requests to access streams via HTTP:
func (c *RTSPClientConnection) handleHTTPCommandStreamingGET(urlSuffix, fullRequestStr string) {
	c.handleHTTPCommandNotSupported()
}

func (c *RTSPClientConnection) setRTSPResponse(responseStr string) {
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%s\r\n",
		responseStr, c.currentCSeq, livemedia.DateHeader())
}

func (c *RTSPClientConnection) setRTSPResponseWithSessionID(responseStr string, sessionID string) {
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 %s\r\n"+
		"CSeq: %s\r\n"+
		"%sSession: %s\r\n\r\n",
		responseStr, c.currentCSeq, livemedia.DateHeader(), sessionID)
}

func (c *RTSPClientConnection) authenticationOK(cmdName, urlSuffix, fullRequestStr string) bool {
	if !c.rtspServer.specialClientAccessCheck(c.clientSocket, c.remoteAddr, urlSuffix) {
		c.setRTSPResponse("401 Unauthorized")
		return false
	}

	authentication := c.rtspServer.authentication
	// dont enable authentication control, pass it
	if authentication == nil {
		return true
	}

	for {
		// To authenticate, we first need to have a nonce set up
		// from a previous attempt:
		if authentication.nonce == "" {
			break
		}

		// Next, the request needs to contain an "Authorization:" header,
		// containing a username, (our) realm, (our) nonce, uri,
		// and response string:
		header := parseAuthorizationHeader(fullRequestStr)
		if header == nil {
			break
		}

		// Next, the username has to be known to us:
		authentication.password = authentication.lookupPassword(header.username)
		if authentication.password == "" {
			break
		}
		authentication.username = header.username

		// Finally, compute a digest response from the information that we have,
		// and compare it to the one that we were given:
		response := authentication.computeDigestResponse(cmdName, header.uri)
		if response == header.response {
			return true
		}
		break
	}

	authentication.randomNonce()
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 401 Unauthorized\r\n"+
		"CSeq: %s\r\n"+
		"%s"+
		"WWW-Authenticate: Digest realm=\"%s\", nonce=\"%s\"\r\n\r\n",
		c.currentCSeq,
		livemedia.DateHeader(),
		authentication.realm, authentication.nonce)
	return false
}

func (c *RTSPClientConnection) newClientSession(sessionID string) *RTSPClientSession {
	return newRTSPClientSession(c, sessionID)
}
