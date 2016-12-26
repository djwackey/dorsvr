package rtspclient

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/dorsvr/livemedia"
	"github.com/djwackey/dorsvr/scheduler"
)

// default value; you can reassign this in your application if you need to
var responseBufferSize = 20000

type RTSPClient struct {
	baseURL                       string
	lastSessionID                 string
	sessionCookie                 string
	serverAddress                 string
	userAgentHeaderStr            string
	responseBuffer                []byte
	cseq                          int
	tcpStreamIDCount              uint
	tunnelOverHTTPPortNum         uint
	responseBufferBytesLeft       uint
	responseBytesAlreadySeen      uint
	tcpConn                       *net.TCPConn
	scs                           *StreamClientState
	requestsAwaitingResponse      *RequestQueue
	requestsAwaitingHTTPTunneling *RequestQueue
}

type ResponseHandler interface {
	Handle(rtspClient *RTSPClient, resultCode int, resultStr string)
}

func New() *RTSPClient {
	return new(RTSPClient)
}

func (c *RTSPClient) DialRTSP(rtspURL string) bool {
	appName := "dorcli"
	c.init(rtspURL, appName)

	// has already connected
	if c.tcpConn != nil {
		return false
	}

	if !c.openConnection() {
		return false
	}

	return true
}

func (c *RTSPClient) SendRequest() bool {
	sendBytes := c.sendDescribeCommand(continueAfterDESCRIBE)
	if sendBytes == 0 {
		fmt.Println("Failed to send describe command.")
		return false
	}

	return true
}

func (c *RTSPClient) Waiting() {
	scheduler.DoEventLoop()
}

func (c *RTSPClient) Close() {
	scs := c.SCS()

	//if scs.Subsession.RtcpInstance() != nil {
	//	scs.Subsession.RtcpInstance().SetByeHandler(nil, nil)
	//}

	c.SendTeardownCommand(scs.Session, nil)
}

func (c *RTSPClient) init(rtspURL, appName string) {
	c.cseq = 1
	c.responseBuffer = make([]byte, responseBufferSize)
	c.setBaseURL(rtspURL)

	c.requestsAwaitingResponse = newRequestQueue()

	c.scs = NewStreamClientState()

	// Set the "User-Agent:" header to use in each request:
	libName := "Dor Streaming Media v"
	libVersionStr := MEDIA_CLIENT_VERSION
	var libPrefix, libSuffix string
	if appName != "" {
		libPrefix = " ("
		libSuffix = ")"
	}

	userAgentName := fmt.Sprintf("%s%s%s%s%s", appName, libPrefix, libName, libVersionStr, libSuffix)
	c.setUserAgentString(userAgentName)
}

func (c *RTSPClient) URL() string {
	return c.baseURL
}

func (c *RTSPClient) SCS() *StreamClientState {
	return c.scs
}

func (c *RTSPClient) SendOptionsCommand(responseHandler interface{}) int {
	c.cseq++
	return c.sendRequest(newRequestRecord(c.cseq, "OPTIONS", responseHandler))
}

func (c *RTSPClient) SendAnnounceCommand(responseHandler interface{}) int {
	c.cseq++
	return c.sendRequest(newRequestRecord(c.cseq, "ANNOUNCE", responseHandler))
}

func (c *RTSPClient) sendDescribeCommand(responseHandler interface{}) int {
	c.cseq++
	return c.sendRequest(newRequestRecord(c.cseq, "DESCRIBE", responseHandler))
}

func (c *RTSPClient) SendSetupCommand(subsession *livemedia.MediaSubSession, responseHandler interface{}) int {
	c.cseq++
	record := newRequestRecord(c.cseq, "SETUP", responseHandler)
	record.setSubSession(subsession)
	return c.sendRequest(record)
}

func (c *RTSPClient) SendPlayCommand(session *livemedia.MediaSession, responseHandler interface{}) int {
	c.cseq++
	record := newRequestRecord(c.cseq, "PLAY", responseHandler)
	record.setSession(session)
	return c.sendRequest(record)
}

func (c *RTSPClient) SendPauseCommand(responseHandler interface{}) int {
	c.cseq++
	return c.sendRequest(newRequestRecord(c.cseq, "PAUSE", responseHandler))
}

func (c *RTSPClient) SendRecordCommand(responseHandler interface{}) int {
	c.cseq++
	return c.sendRequest(newRequestRecord(c.cseq, "RECORD", responseHandler))
}

func (c *RTSPClient) SendTeardownCommand(session *livemedia.MediaSession, responseHandler interface{}) int {
	c.cseq++
	record := newRequestRecord(c.cseq, "TEARDOWN", responseHandler)
	record.setSession(session)
	return c.sendRequest(record)
}

func (c *RTSPClient) SendSetParameterCommand(responseHandler interface{}) int {
	c.cseq++
	return c.sendRequest(newRequestRecord(c.cseq, "SET_PARAMETER", responseHandler))
}

func (c *RTSPClient) SendGetParameterCommand(responseHandler interface{}) int {
	c.cseq++
	return c.sendRequest(newRequestRecord(c.cseq, "GET_PARAMETER", responseHandler))
}

func (c *RTSPClient) setUserAgentString(userAgentName string) {
	c.userAgentHeaderStr = fmt.Sprintf("User-Agent: %s\r\n", userAgentName)
}

func (c *RTSPClient) setBaseURL(url string) {
	c.baseURL = url
}

func (c *RTSPClient) setupHTTPTunneling() {
}

func (c *RTSPClient) openConnection() bool {
	rtspUrl, result := c.parseRTSPURL(c.baseURL)
	if !result {
		return false
	}

	c.serverAddress = rtspUrl.address

	result = c.connectToServer(rtspUrl.address, rtspUrl.port)
	if !result {
		return false
	}

	go c.incomingDataHandler()
	return true
}

func (c *RTSPClient) connectToServer(host string, port int) bool {
	tcpAddr := fmt.Sprintf("%s:%d", host, port)
	addr, err := net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Failed to resolve TCP address.", err)
		return false
	}

	fmt.Printf("Opening connection to %s, port %d...\n", host, port)

	c.tcpConn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Println("Failed to connect to server.", err)
		return false
	}

	fmt.Println("...remote connection opened\n")
	return true
}

func (c *RTSPClient) resetTCPSockets() {
	c.tcpConn.Close()
}

func (c *RTSPClient) createAuthenticatorStr(cmd, url string) string {
	//authFmt := "Authorization: Digest username=\"%s\", realm=\"%s\", " +
	//	"nonce=\"%s\", uri=\"%s\", response=\"%s\"\r\n"
	return ""
}

type RTSPURL struct {
	streamName string
	username   string
	password   string
	address    string
	port       int
}

func (c *RTSPClient) parseRTSPURL(url string) (*RTSPURL, bool) {
	rtspUrl := new(RTSPURL)
	var result bool
	for {
		// Parse the URL as "rtsp://[<username>[:<password>]@]<server-address-or-name>[:<port>][/<stream-name>]"
		prefix := "rtsp://"
		ret := strings.HasPrefix(url, prefix)
		if !ret {
			fmt.Println("URL is not of the form \"" + prefix + "\"")
			break
		}

		// Check whether "<username>[:<password>]@" occurs next.
		index := strings.Index(url, "@")
		if index != -1 {
			// found "@"
			s := strings.Split(url[7:index], ":")
			if len(s) <= 1 {
				fmt.Println("URL is not of the form \"" + url + "\"")
				break
			}

			rtspUrl.username = s[0]
			rtspUrl.password = s[1]
		} else {
			index = 7
		}

		parseBufferSize := 100
		if len(url) > parseBufferSize {
			fmt.Println("URL is too long")
			break
		}

		// Next, parse <server-address-or-name>, port, stream-name
		substrings := strings.Split(url[index:], "/")
		if len(substrings) <= 1 {
			fmt.Println("URL is not of the form \"" + url + "\"")
			break
		}
		rtspUrl.streamName = substrings[1]

		substrings = strings.Split(substrings[0], ":")
		if len(substrings) > 1 {
			rtspUrl.port, _ = strconv.Atoi(substrings[1])
			if rtspUrl.port < 1 || rtspUrl.port > 65535 {
				fmt.Println("Bad Port Number")
				break
			}
		} else {
			rtspUrl.port = 554 // default
		}

		rtspUrl.address = substrings[0]
		result = true
		break
	}
	return rtspUrl, result
}

func (c *RTSPClient) incomingDataHandler() {
	defer c.tcpConn.Close()
	for {
		readBytes, err := gs.ReadSocket(c.tcpConn, c.responseBuffer)
		if err != nil {
			fmt.Println("Failed to read bytes.", err.Error())
			break
		}

		c.handleResponseBytes(c.responseBuffer, readBytes)
	}
}

func getLine(startOfLine string) (thisLineStart, nextLineStart string) {
	var index int
	for i, c := range startOfLine {
		// Check for the end of line: \r\n (but also accept \r or \n by itself):
		if c == '\r' || c == '\n' {
			if c == '\r' {
				if startOfLine[i+1] == '\n' {
					index = i + 2 // skip "\r\n"
				}
			} else {
				index = i + 1
			}

			thisLineStart = startOfLine[:i]
			nextLineStart = startOfLine[index:]
			break
		}
	}
	return nextLineStart, thisLineStart
}

func (c *RTSPClient) handleResponseBytes(buffer []byte, length int) {
	reqStr := string(buffer)[:length]

	fmt.Printf("Received %d new bytes of response data.\n", length)

	nextLineStart, thisLineStart := getLine(reqStr)
	responseCode, responseString, result := c.parseResponseCode(thisLineStart)
	if !result {
		// This does not appear to be a RTSP response; is's a RTSP request instead?
		c.handleIncomingRequest(reqStr, length)
		return
	}

	var n, cseq, contentLength int
	var rangeParamsStr, rtpInfoParamsStr string
	var headerParamsStr, sessionParamsStr string
	var transportParamsStr, scaleParamsStr string
	var wwwAuthenticateParamsStr, publicParamsStr string
	var foundRequest *RequestRecord
	var responseSuccess bool

	for {
		nextLineStart, thisLineStart = getLine(nextLineStart)
		if thisLineStart == "" {
			break
		}

		if headerParamsStr, result = c.checkForHeader(thisLineStart, "CSeq:", 5); result {
			n, _ = fmt.Sscanf(headerParamsStr, "%d", &cseq)
			if n != 1 || cseq <= 0 {
				fmt.Println("Bad \"CSeq\" header: \"", thisLineStart, "\"")
				break
			}

			for {
				request := c.requestsAwaitingResponse.dequeue()
				if request == nil {
					break
				}

				if request.CSeq() < cseq {
					//fmt.Println("WARNING: The server did not respond to our \"", request.CommandName(), "\"")
				} else if request.CSeq() == cseq {
					// This is the handler that we want. Remove its record, but remember it,
					// so that we can later call its handler:
					foundRequest = request
				} else {
					break
				}
			}
		} else if headerParamsStr, result = c.checkForHeader(thisLineStart, "Content-Length:", 15); result {
			n, _ = fmt.Sscanf(headerParamsStr, "%d", &contentLength)
			if n != 1 {
				fmt.Println("Bad \"Content-Length\" header: \"", thisLineStart, "\"")
				break
			}
		} else if headerParamsStr, result = c.checkForHeader(thisLineStart, "Content-Base:", 13); result {
			c.setBaseURL(headerParamsStr)
		} else if sessionParamsStr, result = c.checkForHeader(thisLineStart, "Session:", 8); result {
		} else if transportParamsStr, result = c.checkForHeader(thisLineStart, "Transport:", 10); result {
		} else if scaleParamsStr, result = c.checkForHeader(thisLineStart, "Scale:", 6); result {
		} else if rangeParamsStr, result = c.checkForHeader(thisLineStart, "Range:", 6); result {
		} else if rtpInfoParamsStr, result = c.checkForHeader(thisLineStart, "RTP-Info:", 9); result {
		} else if headerParamsStr, result = c.checkForHeader(thisLineStart, "WWW-Authenticate:", 17); result {
		} else if publicParamsStr, result = c.checkForHeader(thisLineStart, "Public:", 7); result {
		} else if publicParamsStr, result = c.checkForHeader(thisLineStart, "Allow:", 6); result {
		} else if headerParamsStr, result = c.checkForHeader(thisLineStart, "Location:", 9); result {
			c.setBaseURL(headerParamsStr)
		}
	}

	if foundRequest == nil {
		foundRequest = c.requestsAwaitingResponse.dequeue()
	}

	bodyStart := nextLineStart
	numBodyBytes := len(bodyStart)

	var commandName string
	if foundRequest != nil {
		commandName = foundRequest.CommandName
	} else {
		commandName = "(unknown)"
	}

	fmt.Printf("Received a complete %s response:\n%s\n", commandName, reqStr)

	var needToResendCommand bool
	if foundRequest != nil {
		if responseCode == 200 {
			switch foundRequest.CommandName {
			case "SETUP":
				if !c.handleSetupResponse(foundRequest.Subsession(),
					sessionParamsStr, transportParamsStr, false) {
					break
				}
			case "PLAY":
				if !c.handlePlayResponse(scaleParamsStr, rangeParamsStr, rtpInfoParamsStr) {
					break
				}
			case "TEARDOWN":
				if !c.handleTeardownResponse() {
					break
				}
			case "GET_PARAMETER":
				if !c.handleGetParameterResponse(foundRequest.ContentStr()) {
					break
				}
			default:
			}
		} else if responseCode == 401 && c.handleAuthenticationFailure(wwwAuthenticateParamsStr) {
			// We need to resend the command, with an "Authorization:" header:
			needToResendCommand = true

			if foundRequest.CommandName == "GET" {
				c.resetTCPSockets()
			}
		} else if responseCode == 301 || responseCode == 302 { // redirect
			// because we need to connect somewhere else next
			c.resetTCPSockets()
			needToResendCommand = true
		}
	}

	if needToResendCommand {
		return
	}

	responseSuccess = true

	if foundRequest != nil {
		if responseSuccess {
			var resultCode int
			var resultString string
			if responseCode == 200 {
				resultCode = 0

				if numBodyBytes > 0 {
					resultString = bodyStart
				} else {
					resultString = publicParamsStr
				}
			} else {
				resultCode = responseCode
				resultString = responseString
			}

			foundRequest.Handle(c, resultCode, resultString)
		} else {
			c.handleRequestError(foundRequest)
		}
	}
}

func (c *RTSPClient) handleRequestError(request *RequestRecord) {
	request.Handle(c, -1, "FAILED")
}

func (c *RTSPClient) sendRequest(request *RequestRecord) int {
	if c.tunnelOverHTTPPortNum != 0 {
		c.setupHTTPTunneling()
		c.requestsAwaitingHTTPTunneling.enqueue(request)
		return request.CSeq()
	}

	protocalStr := "RTSP/1.0"
	contentLengthHeader := ""

	contentStr := request.ContentStr()
	contentStrLen := len(contentStr)
	if contentStrLen > 0 {
		contentLengthHeader = fmt.Sprintf("Content-Length: %s\r\n", contentStrLen)
	}

	cmdURL := c.baseURL
	var extraHeaders string
	switch request.CommandName {
	case "OPTIONS", "ANNOUNCE":
		extraHeaders = "Content-Type: application/sdp\r\n"
	case "DESCRIBE":
		extraHeaders = "Accept: application/sdp\r\n"
	case "SETUP":
		subsession := request.Subsession()
		streamUsingTCP := (request.BoolFlags() & 0x1) != 0
		streamOutgoing := (request.BoolFlags() & 0x2) != 0

		prefix, separator, suffix := c.constructSubSessionURL(subsession)

		var transportFmt string
		if subsession.ProtocolName() == "UDP" {
			transportFmt = "Transport: RAW/RAW/UDP%s%s%s=%d-%d\r\n"
		} else {
			transportFmt = "Transport: RTP/AVP%s%s%s=%d-%d\r\n"
		}

		cmdURL = fmt.Sprintf("%s%s%s", prefix, separator, suffix)

		var modeStr string
		if streamOutgoing {
			modeStr = ";mode=receive"
		}

		var rtpNumber, rtcpNumber uint
		var transportTypeStr, portTypeStr string
		if streamUsingTCP {
			transportTypeStr = "/TCP;unicast"
			portTypeStr = ";interleaved"
			rtpNumber = c.tcpStreamIDCount
			c.tcpStreamIDCount += 1
			rtcpNumber = c.tcpStreamIDCount
			c.tcpStreamIDCount += 1
		} else {
			transportTypeStr = ";unicast"
			portTypeStr = ";c_port"
			rtpNumber = subsession.ClientPortNum()
			rtcpNumber = rtpNumber + 1
		}

		transportStr := fmt.Sprintf(transportFmt, transportTypeStr, modeStr,
			portTypeStr, rtpNumber, rtcpNumber)

		sessionStr := c.createSessionString(c.lastSessionID)

		extraHeaders = fmt.Sprintf("%s%s", transportStr, sessionStr)
	case "PLAY", "PAUSE", "TEARDOWN", "RECORD", "SET_PARAMETER", "GET_PARAMETER":
		if c.lastSessionID == "" {
			fmt.Println("No RTSP session is currently in progress")
			c.handleRequestError(request)
			return request.CSeq()
		}

		var sessionID string
		var originalScale float32
		if request.Session() != nil {
			sessionID = c.lastSessionID
			originalScale = request.Session().Scale()
		} else {
			subsession := request.Subsession()
			prefix, separator, suffix := c.constructSubSessionURL(subsession)
			cmdURL = fmt.Sprintf("%s%s%s", prefix, separator, suffix)

			sessionID = subsession.SessionID()
			originalScale = subsession.Scale()
		}

		if request.CommandName == "PLAY" {
			sessionStr := c.createSessionString(sessionID)
			scaleStr := c.createScaleString(request.Scale(), originalScale)
			rangeStr := c.createRangeString(request.Start, request.End,
				request.AbsStartTime, request.AbsEndTime)

			extraHeaders = fmt.Sprintf("%s%s%s", sessionStr, scaleStr, rangeStr)
		} else {
			extraHeaders = c.createSessionString(sessionID)
		}
	case "GET", "POST":
		var extraHeadersFmt string
		if request.CommandName == "GET" {
			extraHeadersFmt = "x-sessioncookie: %s\r\n" +
				"Accept: application/x-rtsp-tunnelled\r\n" +
				"Pragma: no-cache\r\n" +
				"Cache-Control: no-cache\r\n"
		} else {
			extraHeadersFmt = "x-sessioncookie: %s\r\n" +
				"Content-Type: application/x-rtsp-tunnelled\r\n" +
				"Pragma: no-cache\r\n" +
				"Cache-Control: no-cache\r\n" +
				"Content-Length: 32767\r\n" +
				"Expires: Sun, 9 Jan 1972 00:00:00 GMT\r\n"
		}
		extraHeaders = fmt.Sprintf(extraHeadersFmt, c.sessionCookie)
	default:
	}

	authenticatorStr := c.createAuthenticatorStr(request.CommandName, c.baseURL)

	cmdFmt := "%s %s %s\r\n" +
		"CSeq: %d\r\n" +
		"%s" +
		"%s" +
		"%s" +
		"%s" +
		"\r\n" +
		"%s"

	cmd := fmt.Sprintf(cmdFmt, request.CommandName,
		cmdURL,
		protocalStr,
		request.CSeq(),
		authenticatorStr,
		c.userAgentHeaderStr,
		extraHeaders,
		contentLengthHeader,
		contentStr)

	writeBytes, err := c.tcpConn.Write([]byte(cmd))
	if err != nil {
		fmt.Println("RTSPClient::sendRequst", err, writeBytes)
		c.handleRequestError(request)
	}

	if c.tunnelOverHTTPPortNum == 0 {
		c.requestsAwaitingResponse.enqueue(request)
	}

	fmt.Printf("Sending request:\n%s\n", cmd)
	return writeBytes
}

func (c *RTSPClient) sessionURL(session *livemedia.MediaSession) string {
	url := session.ControlPath()
	if url == "" || url == "*" {
		url = c.baseURL
	}
	return url
}

func (c *RTSPClient) isAbsoluteURL(url string) bool {
	var isAbsolute bool
	for _, ch := range url {
		if ch == '/' {
			break
		}

		if ch == ':' {
			isAbsolute = true
			break
		}
	}
	return isAbsolute
}

func (c *RTSPClient) constructSubSessionURL(subsession *livemedia.MediaSubSession) (
	prefix string, separator string, suffix string) {

	prefix = c.sessionURL(subsession.ParentSession())
	suffix = subsession.ControlPath()

	if c.isAbsoluteURL(suffix) {
		prefix = ""
		separator = ""
	} else {
		separator = ""
	}
	return prefix, separator, suffix
}

func (c *RTSPClient) createSessionString(sessionID string) string {
	var sessionStr string
	if sessionID != "" {
		sessionStr = fmt.Sprintf("Session: %s\r\n", sessionID)
	}
	return sessionStr
}

func (c *RTSPClient) createScaleString(scale, currentScale float32) string {
	var buf string
	if scale != 1.0 || currentScale != 1.0 {
		buf = fmt.Sprintf("Scale: %f\r\n", scale)
	}
	return buf
}

func (c *RTSPClient) createRangeString(start, end float32, absStartTime, absEndTime string) string {
	var buf string
	if absStartTime != "" {
		// Create a "Range:" header that specifies 'absolute' time values:
		if absEndTime == "" {
			// There's no end time:
			buf = fmt.Sprintf("Range: clock=%s-\r\n", absStartTime)
		} else {
			// There's both a start and an end time; include them both in the "Range:" hdr
			buf = fmt.Sprintf("Range: clock=%s-%s\r\n", absStartTime, absEndTime)
		}
	} else {
		// Create a "Range:" header that specifies relative (i.e., NPT) time values:
		if start < 0 {
			// We're resuming from a PAUSE; there's no "Range:" header at all
		} else if end < 0 {
			// There's no end time:
			buf = fmt.Sprintf("Range: npt=%.3f-\r\n", start)
		} else {
			// There's both a start and an end time; include them both in the "Range:" hdr
			buf = fmt.Sprintf("Range: npt=%.3f-%.3f\r\n", start, end)
		}
	}
	return buf
}

func (c *RTSPClient) parseResponseCode(line string) (responseCode int, responseString string, result bool) {
	var version string
	responseString = line

	for {
		n, _ := fmt.Sscanf(line, "RTSP/%s %d", &version, &responseCode)
		if n == 2 {
			result = true
			break
		}

		n, _ = fmt.Sscanf(line, "HTTP/%s %d", &version, &responseCode)
		if n != 2 {
			result = true
			break
		}

		// Use everything after the RTSP/* (or HTTP/*) as the response string:
		i := 0
		for responseString != "" && responseString[i] != ' ' && responseString[i] != '\t' {
			i++
		}
		i = 0
		for responseString != "" && (responseString[i] == ' ' || responseString[i] == '\t') {
			i++ // skip whitespace
		}
		result = false
		break
	}
	return responseCode, responseString, result
}

func (c *RTSPClient) handleSetupResponse(subsession *livemedia.MediaSubSession,
	sessionParamsStr, transportParamsStr string, streamUsingTCP bool) bool {
	var success bool
	for {
		if sessionParamsStr == "" {
			fmt.Println("Missing or bad \"Session:\" header ")
			break
		}

		sessionID := sessionParamsStr

		subsession.SetSessionID(sessionID)

		c.lastSessionID = sessionID

		// Parse the "Transport:" header parameters:
		transportParams, ok := c.parseTransportParams(transportParamsStr)
		if !ok {
			fmt.Println("Missing or bad \"Transport:\" header ")
			break
		}

		subsession.SetRTPChannelID(transportParams.rtpChannelID)
		subsession.SetRTCPChannelID(transportParams.rtcpChannelID)
		subsession.SetServerPortNum(transportParams.serverPortNum)
		subsession.SetConnectionEndpointName(transportParams.serverAddressStr)

		if streamUsingTCP {
			if subsession.RTPSource != nil {
				subsession.RTPSource.SetStreamSocket()
			}
		} else {
			destAddress := c.serverAddress
			subsession.SetDestinations(destAddress)
		}

		success = true
		break
	}

	return success
}

type TransportParams struct {
	serverPortNum    uint
	rtpChannelID     uint
	rtcpChannelID    uint
	serverAddressStr string
}

func (c *RTSPClient) parseTransportParams(paramsStr string) (*TransportParams, bool) {
	var n int
	var serverPortNum, clientPortNum, multicastPortNumRTP, multicastPortNumRTCP uint
	var foundServerPortNum, foundClientPortNum, foundChannelIDs, foundMulticastPortNum bool
	var foundServerAddressStr, foundDestinationStr string
	var rtpChannelID, rtcpChannelID uint = 0xFF, 0xFF
	isMulticast := true

	params := strings.Split(paramsStr, ";")
	for _, param := range params {
		if param == "unicast" {
			isMulticast = false
		} else if n, _ = fmt.Sscanf(param, "server_port=%d", &serverPortNum); n == 1 {
			foundServerPortNum = true
		} else if n, _ = fmt.Sscanf(param, "c_port=%d", &clientPortNum); n == 1 {
			foundClientPortNum = true
		} else if n, _ = fmt.Sscanf(param, "destination=%s", &foundDestinationStr); n == 1 {
		} else if n, _ = fmt.Sscanf(param, "source=%s", &foundServerAddressStr); n == 1 {
		} else if n, _ = fmt.Sscanf(param, "interleaved=%d-%d", &rtpChannelID, &rtcpChannelID); n == 2 {
			foundChannelIDs = true
		} else {
			n1, _ := fmt.Sscanf(param, "port=%d-%d", &multicastPortNumRTP, &multicastPortNumRTCP)
			n2, _ := fmt.Sscanf(param, "port=%d", &multicastPortNumRTP)
			if n1 == 1 || n2 == 2 {
				foundMulticastPortNum = true
			}
		}
	}

	transportParams := new(TransportParams)
	transportParams.rtpChannelID = rtpChannelID
	transportParams.rtcpChannelID = rtcpChannelID

	if isMulticast && foundDestinationStr != "" && foundMulticastPortNum {
		transportParams.serverAddressStr = foundDestinationStr
		transportParams.serverPortNum = multicastPortNumRTP
		return transportParams, true
	}

	if foundChannelIDs || foundServerPortNum || foundClientPortNum {
		if foundClientPortNum && !foundServerPortNum {
			transportParams.serverPortNum = clientPortNum
		}
		transportParams.serverAddressStr = foundServerAddressStr
		return transportParams, true
	}

	return transportParams, false
}

func (c *RTSPClient) parseScaleParam(paramStr string) (scale float32, ok bool) {
	n, _ := fmt.Sscanf(paramStr, "%f", &scale)
	ok = (n == 1)
	return
}

func (c *RTSPClient) parseRTPInfoParams(paramsStr string) (seqNum, timestamp int, ok bool) {
	ok = true
	return
}

func (c *RTSPClient) handlePlayResponse(scaleParamsStr, rangeParamsStr, rtpInfoParamsStr string) bool {
	return true
}

func (c *RTSPClient) handleTeardownResponse() bool {
	return true
}

func (c *RTSPClient) handleGetParameterResponse(parameterName string) bool {
	fmt.Println("handleGetParameterResponse", parameterName)
	return true
}

func (c *RTSPClient) handleAuthenticationFailure(paramsStr string) bool {
	return false
}

func (c *RTSPClient) handleIncomingRequest(reqStr string, length int) {
	requestString, parseSucceeded := livemedia.ParseRTSPRequestString(reqStr, length)
	if parseSucceeded {
		fmt.Printf("Received incoming RTSP request: %s\n", reqStr)

		buffer := fmt.Sprintf("RTSP/1.0 405 Method Not Allowed\r\nCSeq: %s\r\n\r\n", requestString.Cseq)
		c.tcpConn.Write([]byte(buffer))
	}
}

func (c *RTSPClient) checkForHeader(line, headerName string, headerNameLength int) (headerParams string, result bool) {
	if !strings.HasPrefix(line, headerName) {
		return headerParams, false
	}

	index := headerNameLength
	for _, c := range line[headerNameLength:] {
		if c == ' ' || c == '\t' {
			index += 1
		}
	}

	return line[index:], true
}

type RequestRecord struct {
	cseq         int
	boolFlags    int
	scale        float32
	Start        float32
	End          float32
	CommandName  string
	contentStr   string
	AbsStartTime string
	AbsEndTime   string
	handler      interface{}
	subsession   *livemedia.MediaSubSession
	session      *livemedia.MediaSession
}

func newRequestRecord(cseq int, commandName string, responseHandler interface{}) *RequestRecord {
	requestRecord := new(RequestRecord)
	requestRecord.cseq = cseq
	requestRecord.scale = 1.0
	requestRecord.Start = 0.0
	requestRecord.End = -1.0
	requestRecord.handler = responseHandler
	requestRecord.CommandName = commandName
	return requestRecord
}

func (record *RequestRecord) setSession(session *livemedia.MediaSession) {
	record.session = session
}

func (record *RequestRecord) setSubSession(subsession *livemedia.MediaSubSession) {
	record.subsession = subsession
}

func (record *RequestRecord) Session() *livemedia.MediaSession {
	return record.session
}

func (r *RequestRecord) Subsession() *livemedia.MediaSubSession {
	return r.subsession
}

func (r *RequestRecord) BoolFlags() int {
	return r.boolFlags
}

func (r *RequestRecord) CSeq() int {
	return r.cseq
}

func (r *RequestRecord) Scale() float32 {
	return r.scale
}

func (r *RequestRecord) ContentStr() string {
	return r.contentStr
}

func (r *RequestRecord) Handle(rtspClient *RTSPClient, resultCode int, resultStr string) {
	if r.handler != nil {
		r.handler.(func(rtspClient *RTSPClient,
			resultCode int, resultStr string))(rtspClient, resultCode, resultStr)
	}
}

type RequestQueue struct {
	index          int
	requestRecords []*RequestRecord
}

func newRequestQueue() *RequestQueue {
	requestQueue := new(RequestQueue)
	return requestQueue
}

func (q *RequestQueue) enqueue(request *RequestRecord) {
	q.requestRecords = append(q.requestRecords, request)
}

func (q *RequestQueue) dequeue() *RequestRecord {
	if len(q.requestRecords) <= q.index {
		q.index = 0
		return nil
	}

	requestRecord := q.requestRecords[q.index]
	q.index += 1
	return requestRecord
}

func (q *RequestQueue) putAtHead(request *RequestRecord) {
}

func (q *RequestQueue) findByCSeq(cseq uint) {
}

func (q *RequestQueue) isEmpty() bool {
	return len(q.requestRecords) < 1
}
