package rtspserver

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/djwackey/dorsvr/auth"
	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/dorsvr/livemedia"
	"github.com/djwackey/gitea/log"
)

const rtspBufferSize = 10000

type RTSPClientConnection struct {
	socket         net.Conn
	localPort      string
	remotePort     string
	localAddr      string
	remoteAddr     string
	currentCSeq    string
	responseBuffer string
	server         *RTSPServer
	digest         *auth.Digest
}

func newRTSPClientConnection(server *RTSPServer, socket net.Conn) *RTSPClientConnection {
	localAddr := strings.Split(socket.LocalAddr().String(), ":")
	remoteAddr := strings.Split(socket.RemoteAddr().String(), ":")
	return &RTSPClientConnection{
		server:     server,
		socket:     socket,
		localAddr:  localAddr[0],
		localPort:  localAddr[1],
		remoteAddr: remoteAddr[0],
		remotePort: remoteAddr[1],
		digest:     auth.NewDigest(),
	}
}

func (c *RTSPClientConnection) destroy() error {
	return c.socket.Close()
}

func (c *RTSPClientConnection) incomingRequestHandler() {
	defer c.socket.Close()

	var isclose bool
	buffer := make([]byte, rtspBufferSize)
	for {
		length, err := c.socket.Read(buffer)

		switch err {
		case nil:
			err = c.handleRequestBytes(buffer, length)
			if err != nil {
				log.Error(4, "Failed to handle Request Bytes: %v", err)
				isclose = true
			}
		default:
			log.Info("default: %v", err)
			if err.Error() == "EOF" {
				isclose = true
			}
		}

		if isclose {
			break
		}
	}

	log.Info("end connection[%s:%s].", c.remoteAddr, c.remotePort)
}

func (c *RTSPClientConnection) handleRequestBytes(buffer []byte, length int) error {
	if length < 0 {
		return errors.New("EOF")
	}

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
						if _, existed = c.server.getClientSession(sessionIDStr); !existed {
							break
						}
					}
					clientSession = c.newClientSession(sessionIDStr)
					c.server.addClientSession(sessionIDStr, clientSession)
				} else {
					if clientSession, existed = c.server.getClientSession(sessionIDStr); !existed {
						c.handleCommandSessionNotFound()
					}
				}

				if clientSession != nil {
					clientSession.handleCommandSetup(requestString.UrlPreSuffix, requestString.UrlSuffix, reqStr)
				}
			}
		case "PLAY", "PAUSE", "TEARDOWN", "GET_PARAMETER", "SET_PARAMETER":
			{
				if clientSession, existed = c.server.getClientSession(sessionIDStr); existed {
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

	sendBytes, err := c.socket.Write([]byte(c.responseBuffer))
	if err != nil {
		log.Error(4, "failed to send response buffer.%d", sendBytes)
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

	var sms *livemedia.ServerMediaSession
	sms = c.server.lookupServerMediaSession(urlTotalSuffix)
	if sms == nil {
		c.handleCommandNotFound()
		return
	}

	sdpDescription := sms.GenerateSDPDescription()
	sdpDescriptionSize := len(sdpDescription)
	if sdpDescriptionSize <= 0 {
		c.setRTSPResponse("404 File Not Found, Or In Incorrect Format")
		return
	}

	streamName := sms.StreamName()
	rtspURL := c.server.RtspURL(streamName)
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
	if _, existed := c.server.clientHTTPConnections[sessionCookie]; !existed {
		c.server.clientHTTPConnections[sessionCookie] = c
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
	if !c.server.specialClientAccessCheck(c.socket, c.remoteAddr, urlSuffix) {
		c.setRTSPResponse("401 Unauthorized")
		return false
	}

	authDatabase := c.server.authDatabase
	// dont enable authentication control, pass it
	if authDatabase == nil {
		return true
	}

	for {
		// To authenticate, we first need to have a nonce set up
		// from a previous attempt:
		if c.digest.Nonce == "" {
			break
		}

		// Next, the request needs to contain an "Authorization:" header,
		// containing a username, (our) realm, (our) nonce, uri,
		// and response string:
		header := auth.ParseAuthorizationHeader(fullRequestStr)
		if header == nil {
			break
		}

		// Next, the username has to be known to us:
		c.digest.Password = authDatabase.LookupPassword(header.Username)
		if c.digest.Password == "" {
			break
		}
		c.digest.Username = header.Username

		// Finally, compute a digest response from the information that we have,
		// and compare it to the one that we were given:
		response := c.digest.ComputeResponse(cmdName, header.URI)
		if response == header.Response {
			return true
		}
		break
	}

	c.digest.Realm = authDatabase.Realm
	c.digest.RandomNonce()
	c.responseBuffer = fmt.Sprintf("RTSP/1.0 401 Unauthorized\r\n"+
		"CSeq: %s\r\n"+
		"%s"+
		"WWW-Authenticate: Digest realm=\"%s\", nonce=\"%s\"\r\n\r\n",
		c.currentCSeq,
		livemedia.DateHeader(),
		c.digest.Realm, c.digest.Nonce)
	return false
}

func (c *RTSPClientConnection) newClientSession(sessionID string) *RTSPClientSession {
	return newRTSPClientSession(c, sessionID)
}
