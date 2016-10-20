package liveMedia

import (
	"constant"
	"fmt"
	. "groupsock"
	"net"
	"strconv"
	"strings"
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
	requestsAwaitingConnection    *RequestQueue
	requestsAwaitingHTTPTunneling *RequestQueue
}

type RequestRecord struct {
	cseq         int
	boolFlags    int
	scale        float32
	start        float32
	end          float32
	commandName  string
	contentStr   string
	absStartTime string
	absEndTime   string
	handler      interface{}
	subsession   *MediaSubSession
	session      *MediaSession
}

type ResponseHandler interface {
	Handle(rtspClient *RTSPClient, resultCode int, resultStr string)
}

func NewRTSPClient(rtspURL, appName string) *RTSPClient {
	rtspClient := new(RTSPClient)
	rtspClient.InitRTSPClient(rtspURL, appName)
	return rtspClient
}

func NewRequestRecord(cseq int, commandName string, responseHandler interface{}) *RequestRecord {
	requestRecord := new(RequestRecord)
	requestRecord.cseq = cseq
	requestRecord.scale = 1.0
	requestRecord.start = 0.0
	requestRecord.end = -1.0
	requestRecord.handler = responseHandler
	requestRecord.commandName = commandName
	return requestRecord
}

func (record *RequestRecord) setSession(session *MediaSession) {
	record.session = session
}

func (record *RequestRecord) setSubSession(subsession *MediaSubSession) {
	record.subsession = subsession
}

func (record *RequestRecord) CommandName() string {
	return record.commandName
}

func (record *RequestRecord) Session() *MediaSession {
	return record.session
}

func (record *RequestRecord) Subsession() *MediaSubSession {
	return record.subsession
}

func (record *RequestRecord) BoolFlags() int {
	return record.boolFlags
}

func (record *RequestRecord) CSeq() int {
	return record.cseq
}

func (record *RequestRecord) Scale() float32 {
	return record.scale
}

func (record *RequestRecord) Start() float32 {
	return record.start
}

func (record *RequestRecord) End() float32 {
	return record.end
}

func (record *RequestRecord) AbsStartTime() string {
	return record.absStartTime
}

func (record *RequestRecord) AbsEndTime() string {
	return record.absEndTime
}

func (record *RequestRecord) ContentStr() string {
	return record.contentStr
}

func (record *RequestRecord) Handle(rtspClient *RTSPClient, resultCode int, resultStr string) {
	if record.handler != nil {
		record.handler.(func(rtspClient *RTSPClient, resultCode int, resultStr string))(rtspClient, resultCode, resultStr)
	}
}

func (this *RTSPClient) InitRTSPClient(rtspURL, appName string) {
	this.cseq = 1
	this.responseBuffer = make([]byte, responseBufferSize)
	this.setBaseURL(rtspURL)

	this.requestsAwaitingResponse = NewRequestQueue()
	this.requestsAwaitingConnection = NewRequestQueue()

	this.scs = NewStreamClientState()

	// Set the "User-Agent:" header to use in each request:
	libName := "Dor Streaming Media v"
	libVersionStr := constant.MEDIA_SERVER_VERSION
	libPrefix := ""
	libSuffix := ""
	if appName != "" {
		libPrefix = " ("
		libSuffix = ")"
	}

	userAgentName := fmt.Sprintf("%s%s%s%s%s", appName, libPrefix, libName, libVersionStr, libSuffix)
	this.setUserAgentString(userAgentName)
}

func (this *RTSPClient) URL() string {
	return this.baseURL
}

func (this *RTSPClient) SCS() *StreamClientState {
	return this.scs
}

func (this *RTSPClient) SendOptionsCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "OPTIONS", responseHandler))
}

func (this *RTSPClient) SendAnnounceCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "ANNOUNCE", responseHandler))
}

func (this *RTSPClient) SendDescribeCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "DESCRIBE", responseHandler))
}

func (this *RTSPClient) SendSetupCommand(subsession *MediaSubSession, responseHandler interface{}) int {
	this.cseq++
	record := NewRequestRecord(this.cseq, "SETUP", responseHandler)
	record.setSubSession(subsession)
	return this.sendRequest(record)
}

func (this *RTSPClient) SendPlayCommand(session *MediaSession, responseHandler interface{}) int {
	this.cseq++
	record := NewRequestRecord(this.cseq, "PLAY", responseHandler)
	record.setSession(session)
	return this.sendRequest(record)
}

func (this *RTSPClient) SendPauseCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "PAUSE", responseHandler))
}

func (this *RTSPClient) SendRecordCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "RECORD", responseHandler))
}

func (this *RTSPClient) SendTeardownCommand(session *MediaSession, responseHandler interface{}) int {
	this.cseq++
	record := NewRequestRecord(this.cseq, "TEARDOWN", responseHandler)
	record.setSession(session)
	return this.sendRequest(record)
}

func (this *RTSPClient) SendSetParameterCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "SET_PARAMETER", responseHandler))
}

func (this *RTSPClient) SendGetParameterCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "GET_PARAMETER", responseHandler))
}

func (this *RTSPClient) setUserAgentString(userAgentName string) {
	this.userAgentHeaderStr = fmt.Sprintf("User-Agent: %s\r\n", userAgentName)
}

func (this *RTSPClient) setBaseURL(url string) {
	this.baseURL = url
}

func (this *RTSPClient) setupHTTPTunneling() {
}

func (this *RTSPClient) openConnection() bool {
	rtspUrl, result := this.parseRTSPURL(this.baseURL)
	if !result {
		return false
	}

	this.serverAddress = rtspUrl.address

	result = this.connectToServer(rtspUrl.address, rtspUrl.port)
	if !result {
		return false
	}

	go this.incomingDataHandler()
	return true
}

func (this *RTSPClient) connectToServer(host string, port int) bool {
	tcpAddr := fmt.Sprintf("%s:%d", host, port)
	addr, err := net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Failed to resolve TCP address.", err)
		return false
	}

	fmt.Printf("Opening connection to %s, port %d...\n", host, port)

	this.tcpConn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Println("Failed to connect to server.", err)
		return false
	}

	fmt.Println("...remote connection opened\n")
	return true
}

func (this *RTSPClient) resetTCPSockets() {
	this.tcpConn.Close()
}

func (this *RTSPClient) createAuthenticatorStr(cmd, url string) string {
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

func (this *RTSPClient) parseRTSPURL(url string) (*RTSPURL, bool) {
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

func (this *RTSPClient) incomingDataHandler() {
	defer this.tcpConn.Close()
	for {
		readBytes, err := ReadSocket(this.tcpConn, this.responseBuffer)
		if err != nil {
			fmt.Println("Failed to read bytes.", err.Error())
			break
		}

		this.handleResponseBytes(this.responseBuffer, readBytes)
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

func (this *RTSPClient) handleResponseBytes(buffer []byte, length int) {
	reqStr := string(buffer)[:length]

	fmt.Printf("Received %d new bytes of response data.\n", length)

	nextLineStart, thisLineStart := getLine(reqStr)
	responseCode, responseString, result := this.parseResponseCode(thisLineStart)
	if !result {
		// This does not appear to be a RTSP response; is's a RTSP request instead?
		this.handleIncomingRequest(reqStr, length)
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

		if headerParamsStr, result = this.checkForHeader(thisLineStart, "CSeq:", 5); result {
			n, _ = fmt.Sscanf(headerParamsStr, "%d", &cseq)
			if n != 1 || cseq <= 0 {
				fmt.Println("Bad \"CSeq\" header: \"", thisLineStart, "\"")
				break
			}

			for {
				request := this.requestsAwaitingResponse.dequeue()
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
		} else if headerParamsStr, result = this.checkForHeader(thisLineStart, "Content-Length:", 15); result {
			n, _ = fmt.Sscanf(headerParamsStr, "%d", &contentLength)
			if n != 1 {
				fmt.Println("Bad \"Content-Length\" header: \"", thisLineStart, "\"")
				break
			}
		} else if headerParamsStr, result = this.checkForHeader(thisLineStart, "Content-Base:", 13); result {
			this.setBaseURL(headerParamsStr)
		} else if sessionParamsStr, result = this.checkForHeader(thisLineStart, "Session:", 8); result {
		} else if transportParamsStr, result = this.checkForHeader(thisLineStart, "Transport:", 10); result {
		} else if scaleParamsStr, result = this.checkForHeader(thisLineStart, "Scale:", 6); result {
		} else if rangeParamsStr, result = this.checkForHeader(thisLineStart, "Range:", 6); result {
		} else if rtpInfoParamsStr, result = this.checkForHeader(thisLineStart, "RTP-Info:", 9); result {
		} else if headerParamsStr, result = this.checkForHeader(thisLineStart, "WWW-Authenticate:", 17); result {
		} else if publicParamsStr, result = this.checkForHeader(thisLineStart, "Public:", 7); result {
		} else if publicParamsStr, result = this.checkForHeader(thisLineStart, "Allow:", 6); result {
		} else if headerParamsStr, result = this.checkForHeader(thisLineStart, "Location:", 9); result {
			this.setBaseURL(headerParamsStr)
		}
	}

	if foundRequest == nil {
		foundRequest = this.requestsAwaitingResponse.dequeue()
	}

	bodyStart := nextLineStart
	numBodyBytes := len(bodyStart)

	var commandName string
	if foundRequest != nil {
		commandName = foundRequest.CommandName()
	} else {
		commandName = "(unknown)"
	}

	fmt.Printf("Received a complete %s response:\n%s\n", commandName, reqStr)

	var needToResendCommand bool
	if foundRequest != nil {
		if responseCode == 200 {
			switch foundRequest.CommandName() {
			case "SETUP":
				if !this.handleSetupResponse(foundRequest.Subsession(),
					sessionParamsStr, transportParamsStr, false) {
					break
				}
			case "PLAY":
				if !this.handlePlayResponse(scaleParamsStr, rangeParamsStr, rtpInfoParamsStr) {
					break
				}
			case "TEARDOWN":
				if !this.handleTeardownResponse() {
					break
				}
			case "GET_PARAMETER":
				if !this.handleGetParameterResponse(foundRequest.ContentStr()) {
					break
				}
			default:
			}
		} else if responseCode == 401 && this.handleAuthenticationFailure(wwwAuthenticateParamsStr) {
			// We need to resend the command, with an "Authorization:" header:
			needToResendCommand = true

			if foundRequest.CommandName() == "GET" {
				this.resetTCPSockets()
			}
		} else if responseCode == 301 || responseCode == 302 { // redirect
			// because we need to connect somewhere else next
			this.resetTCPSockets()
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

			foundRequest.Handle(this, resultCode, resultString)
		} else {
			this.handleRequestError(foundRequest)
		}
	}
}

func (this *RTSPClient) handleRequestError(request *RequestRecord) {
	request.Handle(this, -1, "FAILED")
}

func (this *RTSPClient) sendRequest(request *RequestRecord) int {
	var connectionIsPending bool
	if !this.requestsAwaitingConnection.isEmpty() {
		connectionIsPending = true
	} else if this.tcpConn == nil {
		if !this.openConnection() {
			fmt.Println("Failed to open Connection.")
			return 0
		}
	}

	if connectionIsPending {
		this.requestsAwaitingConnection.enqueue(request)
		return request.CSeq()
	}

	if this.tunnelOverHTTPPortNum != 0 {
		this.setupHTTPTunneling()
		this.requestsAwaitingHTTPTunneling.enqueue(request)
		return request.CSeq()
	}

	protocalStr := "RTSP/1.0"
	contentLengthHeader := ""

	contentStr := request.ContentStr()
	contentStrLen := len(contentStr)
	if contentStrLen > 0 {
		contentLengthHeader = fmt.Sprintf("Content-Length: %s\r\n", contentStrLen)
	}

	cmdURL := this.baseURL
	var extraHeaders string
	switch request.CommandName() {
	case "OPTIONS", "ANNOUNCE":
		extraHeaders = "Content-Type: application/sdp\r\n"
	case "DESCRIBE":
		extraHeaders = "Accept: application/sdp\r\n"
	case "SETUP":
		subsession := request.Subsession()
		streamUsingTCP := (request.BoolFlags() & 0x1) != 0
		streamOutgoing := (request.BoolFlags() & 0x2) != 0

		prefix, separator, suffix := this.constructSubSessionURL(subsession)

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
			rtpNumber = this.tcpStreamIDCount
			this.tcpStreamIDCount += 1
			rtcpNumber = this.tcpStreamIDCount
			this.tcpStreamIDCount += 1
		} else {
			transportTypeStr = ";unicast"
			portTypeStr = ";client_port"
			rtpNumber = subsession.ClientPortNum()
			rtcpNumber = rtpNumber + 1
		}

		transportStr := fmt.Sprintf(transportFmt, transportTypeStr, modeStr,
			portTypeStr, rtpNumber, rtcpNumber)

		sessionStr := this.createSessionString(this.lastSessionID)

		extraHeaders = fmt.Sprintf("%s%s", transportStr, sessionStr)
	case "PLAY", "PAUSE", "TEARDOWN", "RECORD", "SET_PARAMETER", "GET_PARAMETER":
		if this.lastSessionID == "" {
			fmt.Println("No RTSP session is currently in progress")
			this.handleRequestError(request)
			return request.CSeq()
		}

		var sessionID string
		var originalScale float32
		if request.Session() != nil {
			sessionID = this.lastSessionID
			originalScale = request.Session().Scale()
		} else {
			subsession := request.Subsession()
			prefix, separator, suffix := this.constructSubSessionURL(subsession)
			cmdURL = fmt.Sprintf("%s%s%s", prefix, separator, suffix)

			sessionID = subsession.SessionID()
			originalScale = subsession.Scale()
		}

		if request.CommandName() == "PLAY" {
			sessionStr := this.createSessionString(sessionID)
			scaleStr := this.createScaleString(request.Scale(), originalScale)
			rangeStr := this.createRangeString(request.Start(), request.End(),
				request.AbsStartTime(), request.AbsEndTime())

			extraHeaders = fmt.Sprintf("%s%s%s", sessionStr, scaleStr, rangeStr)
		} else {
			extraHeaders = this.createSessionString(sessionID)
		}
	case "GET", "POST":
		var extraHeadersFmt string
		if request.CommandName() == "GET" {
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
		extraHeaders = fmt.Sprintf(extraHeadersFmt, this.sessionCookie)
	default:
	}

	authenticatorStr := this.createAuthenticatorStr(request.CommandName(), this.baseURL)

	cmdFmt := "%s %s %s\r\n" +
		"CSeq: %d\r\n" +
		"%s" +
		"%s" +
		"%s" +
		"%s" +
		"\r\n" +
		"%s"

	cmd := fmt.Sprintf(cmdFmt, request.CommandName(),
		cmdURL,
		protocalStr,
		request.CSeq(),
		authenticatorStr,
		this.userAgentHeaderStr,
		extraHeaders,
		contentLengthHeader,
		contentStr)

	writeBytes, err := this.tcpConn.Write([]byte(cmd))
	if err != nil {
		fmt.Println("RTSPClient::sendRequst", err, writeBytes)
		this.handleRequestError(request)
	}

	if this.tunnelOverHTTPPortNum == 0 {
		this.requestsAwaitingResponse.enqueue(request)
	}

	fmt.Printf("Sending request:\n%s\n", cmd)
	return writeBytes
}

func (this *RTSPClient) sessionURL(session *MediaSession) string {
	url := session.ControlPath()
	if url == "" || url == "*" {
		url = this.baseURL
	}
	return url
}

func (this *RTSPClient) isAbsoluteURL(url string) bool {
	var isAbsolute bool
	for _, c := range url {
		if c == '/' {
			break
		}

		if c == ':' {
			isAbsolute = true
			break
		}
	}
	return isAbsolute
}

func (this *RTSPClient) constructSubSessionURL(subsession *MediaSubSession) (
	prefix string, separator string, suffix string) {

	prefix = this.sessionURL(subsession.ParentSession())
	suffix = subsession.ControlPath()

	if this.isAbsoluteURL(suffix) {
		prefix = ""
		separator = ""
	} else {
		separator = ""
	}
	return prefix, separator, suffix
}

func (this *RTSPClient) createSessionString(sessionID string) string {
	var sessionStr string
	if sessionID != "" {
		sessionStr = fmt.Sprintf("Session: %s\r\n", sessionID)
	}
	return sessionStr
}

func (this *RTSPClient) createScaleString(scale, currentScale float32) string {
	var buf string
	if scale != 1.0 || currentScale != 1.0 {
		buf = fmt.Sprintf("Scale: %f\r\n", scale)
	}
	return buf
}

func (this *RTSPClient) createRangeString(start, end float32, absStartTime, absEndTime string) string {
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

func (this *RTSPClient) parseResponseCode(line string) (responseCode int, responseString string, result bool) {
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

func (this *RTSPClient) handleSetupResponse(subsession *MediaSubSession,
	sessionParamsStr, transportParamsStr string, streamUsingTCP bool) bool {
	var success bool
	for {
		if sessionParamsStr == "" {
			fmt.Println("Missing or bad \"Session:\" header ")
			break
		}

		sessionID := sessionParamsStr

		subsession.setSessionID(sessionID)

		this.lastSessionID = sessionID

		// Parse the "Transport:" header parameters:
		transportParams, ok := this.parseTransportParams(transportParamsStr)
		if !ok {
			fmt.Println("Missing or bad \"Transport:\" header ")
			break
		}

		subsession.rtpChannelID = transportParams.rtpChannelID
		subsession.rtcpChannelID = transportParams.rtcpChannelID
		subsession.serverPortNum = transportParams.serverPortNum
		subsession.connectionEndpointName = transportParams.serverAddressStr

		if streamUsingTCP {
			if subsession.rtpSource != nil {
				subsession.rtpSource.setStreamSocket()
			}
		} else {
			destAddress := this.serverAddress
			subsession.setDestinations(destAddress)
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

func (this *RTSPClient) parseTransportParams(paramsStr string) (*TransportParams, bool) {
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
		} else if n, _ = fmt.Sscanf(param, "client_port=%d", &clientPortNum); n == 1 {
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

func (this *RTSPClient) parseScaleParam(paramStr string) (scale float32, ok bool) {
	n, _ := fmt.Sscanf(paramStr, "%f", &scale)
	ok = (n == 1)
	return
}

func (this *RTSPClient) parseRTPInfoParams(paramsStr string) (seqNum, timestamp int, ok bool) {
	ok = true
	return
}

func (this *RTSPClient) handlePlayResponse(scaleParamsStr, rangeParamsStr, rtpInfoParamsStr string) bool {
	return true
}

func (this *RTSPClient) handleTeardownResponse() bool {
	return true
}

func (this *RTSPClient) handleGetParameterResponse(parameterName string) bool {
	fmt.Println("handleGetParameterResponse", parameterName)
	return true
}

func (this *RTSPClient) handleAuthenticationFailure(paramsStr string) bool {
	return false
}

func (this *RTSPClient) handleIncomingRequest(reqStr string, length int) {
	requestString, parseSucceeded := ParseRTSPRequestString(reqStr, length)
	if parseSucceeded {
		fmt.Printf("Received incoming RTSP request: %s\n", reqStr)

		buffer := fmt.Sprintf("RTSP/1.0 405 Method Not Allowed\r\nCSeq: %s\r\n\r\n", requestString.cseq)
		this.tcpConn.Write([]byte(buffer))
	}
}

func (this *RTSPClient) checkForHeader(line, headerName string, headerNameLength int) (headerParams string, result bool) {
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

type RequestQueue struct {
	index          int
	requestRecords []*RequestRecord
}

func NewRequestQueue() *RequestQueue {
	requestQueue := new(RequestQueue)
	return requestQueue
}

func (this *RequestQueue) enqueue(request *RequestRecord) {
	this.requestRecords = append(this.requestRecords, request)
}

func (this *RequestQueue) dequeue() *RequestRecord {
	if len(this.requestRecords) <= this.index {
		this.index = 0
		return nil
	}

	requestRecord := this.requestRecords[this.index]
	this.index += 1
	return requestRecord
}

func (this *RequestQueue) putAtHead(request *RequestRecord) {
}

func (this *RequestQueue) findByCSeq(cseq uint) {
}

func (this *RequestQueue) isEmpty() bool {
	return len(this.requestRecords) < 1
}
