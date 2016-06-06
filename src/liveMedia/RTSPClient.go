package liveMedia

import (
	"fmt"
	. "groupsock"
	. "include"
	"net"
	"strconv"
	"strings"
)

// default value; you can reassign this in your application if you need to
var responseBufferSize = 20000

type RTSPClient struct {
	baseURL                       string
	userAgentHeaderStr            string
	responseBuffer                []byte
	cseq                          int
	tunnelOverHTTPPortNum         int
	responseBufferBytesLeft       uint
	responseBytesAlreadySeen      uint
	tcpConn                       *net.TCPConn
	scs                           *StreamClientState
	requestsAwaitingResponse      *RequestQueue
	requestsAwaitingConnection    *RequestQueue
	requestsAwaitingHTTPTunneling *RequestQueue
}

type RequestRecord struct {
	cseq        int
	commandName string
	contentStr  string
	handler     interface{}
	subsession  *MediaSubSession
	session     *MediaSession
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
	requestRecord.handler = responseHandler
	requestRecord.commandName = commandName
	return requestRecord
}

func (this *RequestRecord) CommandName() string {
	return this.commandName
}

func (this *RequestRecord) Session() *MediaSession {
	return this.session
}

func (this *RequestRecord) Subsession() *MediaSubSession {
	return this.subsession
}

func (this *RequestRecord) CSeq() int {
	return this.cseq
}

func (this *RequestRecord) ContentStr() string {
	return this.contentStr
}

func (this *RequestRecord) Handle(rtspClient *RTSPClient, resultCode int, resultStr string) {
	if this.handler != nil {
		this.handler.(func(rtspClient *RTSPClient, resultCode int, resultStr string))(rtspClient, resultCode, resultStr)
	}
}

func (this *RTSPClient) InitRTSPClient(rtspURL, appName string) {
	this.cseq = 1
	this.responseBuffer = make([]byte, responseBufferSize)
	this.SetBaseURL(rtspURL)

	this.requestsAwaitingResponse = NewRequestQueue()
	this.requestsAwaitingConnection = NewRequestQueue()

	this.scs = NewStreamClientState()

	// Set the "User-Agent:" header to use in each request:
	libName := "Dor Streaming Media v"
	libVersionStr := MEDIA_SERVER_VERSION
	libPrefix := ""
	libSuffix := ""
	if appName != "" {
		libPrefix = " ("
		libSuffix = ")"
	}

	userAgentName := fmt.Sprintf("%s%s%s%s%s", appName, libPrefix, libName, libVersionStr, libSuffix)
	this.SetUserAgentString(userAgentName)
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

func (this *RTSPClient) SendSetupCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "SETUP", responseHandler))
}

func (this *RTSPClient) SendPlayCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "PLAY", responseHandler))
}

func (this *RTSPClient) SendPauseCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "PAUSE", responseHandler))
}

func (this *RTSPClient) SendRecordCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "RECORD", responseHandler))
}

func (this *RTSPClient) SendTeardownCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "TEARDOWN", responseHandler))
}

func (this *RTSPClient) SendSetParameterCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "SET_PARAMETER", responseHandler))
}

func (this *RTSPClient) SendGetParameterCommand(responseHandler interface{}) int {
	this.cseq++
	return this.sendRequest(NewRequestRecord(this.cseq, "GET_PARAMETER", responseHandler))
}

func (this *RTSPClient) SetUserAgentString(userAgentName string) {
	formatStr := "User-Agent: %s\r\n"
	this.userAgentHeaderStr = fmt.Sprintf(formatStr, userAgentName)
}

func (this *RTSPClient) SetBaseURL(url string) {
	this.baseURL = url
}

func (this *RTSPClient) setupHTTPTunneling() {
}

func (this *RTSPClient) openConnection() bool {
	//SetupStreamSocket()
	rtspUrl, result := this.parseRTSPURL(this.baseURL)
	if !result {
		return false
	}

	result = this.connectToServer(rtspUrl.address, rtspUrl.port)
	if !result {
		return false
	}

	//defer this.tcpConn.Close()
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

	this.tcpConn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Println("Failed to connect to server.", err)
		return false
	}

	return true
}

func (this *RTSPClient) createAuthenticatorStr() string {
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
			fmt.Println("[readSocket]", err.Error())
			break
		}

		fmt.Println("[readSocket] success for reading ", readBytes, string(this.responseBuffer))
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
					thisLineStart = startOfLine[:i]
					nextLineStart = startOfLine[index:]
					break
				}
			}

			index = i + 1
			thisLineStart = startOfLine[:i]
			nextLineStart = startOfLine[index:]
			break
		}
	}
	return nextLineStart, thisLineStart
}

func (this *RTSPClient) handleResponseBytes(buffer []byte, length int) {
	reqStr := string(buffer)[:length]

	nextLineStart, thisLineStart := getLine(reqStr)
	//fmt.Println("thisLineStart", thisLineStart)
	//fmt.Println("thisLineLength", len(thisLineStart))
	//fmt.Println("nextLineStart", nextLineStart)
	responseCode, responseString, result := this.parseResponseCode(thisLineStart)
	if !result {
		this.handleIncomingRequest()
		return
	}

	//fmt.Println("responseCode: ", responseCode)
	//fmt.Println("responseString: ", responseString)

	var ret, cseq, contentLength int
	var rangeParamsStr, rtpInfoParamsStr string
	var headerParamsStr, sessionParamsStr string
	var transportParamsStr, scaleParamsStr string
	var wwwAuthenticateParamsStr, publicParamsStr string
	var foundRequest *RequestRecord
	var responseSuccess bool

	for {
		//fmt.Println("thisLineStart:", thisLineStart)
		//fmt.Println("nextLineStart:", nextLineStart)

		nextLineStart, thisLineStart = getLine(nextLineStart)
		if thisLineStart == "" {
			break
		}

		//fmt.Println("yanfei:", thisLineStart)

		if headerParamsStr, result = this.checkForHeader(thisLineStart, "CSeq:", 5); result {
			ret, _ = fmt.Sscanf(headerParamsStr, "%d", &cseq)
			if ret != 1 || cseq <= 0 {
				fmt.Println("Bad \"CSeq\" header: \"", thisLineStart, "\"")
				break
			}

			for {
				request := this.requestsAwaitingResponse.dequeue()
				if request == nil {
					break
				}

				if request.CSeq() < cseq {
					fmt.Println("WARNING:")
				} else if request.CSeq() == cseq {
					// This is the handler that we want. Remove its record, but remember it,
					// so that we can later call its handler:
					foundRequest = request
				} else {
					break
				}
			}
		} else if headerParamsStr, result = this.checkForHeader(thisLineStart, "Content-Length:", 15); result {
			ret, _ = fmt.Sscanf(headerParamsStr, "%d", &contentLength)
			if ret != 1 {
				fmt.Println("Bad \"Content-Length\" header: \"", thisLineStart, "\"")
				break
			}
		} else if headerParamsStr, result = this.checkForHeader(thisLineStart, "Content-Base:", 13); result {
			this.SetBaseURL(headerParamsStr)
		} else if sessionParamsStr, result = this.checkForHeader(thisLineStart, "Session:", 8); result {
		} else if transportParamsStr, result = this.checkForHeader(thisLineStart, "Transport:", 10); result {
		} else if scaleParamsStr, result = this.checkForHeader(thisLineStart, "Scale:", 6); result {
		} else if rangeParamsStr, result = this.checkForHeader(thisLineStart, "Range:", 6); result {
		} else if rtpInfoParamsStr, result = this.checkForHeader(thisLineStart, "RTP-Info:", 9); result {
		} else if headerParamsStr, result = this.checkForHeader(thisLineStart, "WWW-Authenticate:", 17); result {
		} else if publicParamsStr, result = this.checkForHeader(thisLineStart, "Public:", 7); result {
		} else if publicParamsStr, result = this.checkForHeader(thisLineStart, "Allow:", 6); result {
		} else if headerParamsStr, result = this.checkForHeader(thisLineStart, "Location:", 9); result {
			this.SetBaseURL(headerParamsStr)
		}
	}

	bodyStart := nextLineStart

	fmt.Println(sessionParamsStr)
	fmt.Println(rangeParamsStr, rtpInfoParamsStr)
	fmt.Println(transportParamsStr, scaleParamsStr)
	fmt.Println(wwwAuthenticateParamsStr, publicParamsStr)

	if responseCode == 200 {
		switch foundRequest.CommandName() {
		case "SETUP":
		case "PLAY":
		case "TEARDOWN":
		case "GET_PARAMETER":
		default:
		}
	} else if responseCode == 401 {
	} else if responseCode == 301 || responseCode == 302 {
	}

	responseSuccess = true

	if foundRequest != nil {
		if responseSuccess {
			var resultCode int
			var resultString string
			if responseCode == 200 {
				resultCode = 0
				resultString = bodyStart
			} else {
				resultCode = responseCode
				resultString = responseString
			}

			fmt.Println("foundRequest:", foundRequest)
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
		fmt.Println("Success for opening Connection.")
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
		contentLengthHeaderFmt := "Content-Length: %s\r\n"
		contentLengthHeader = fmt.Sprintf(contentLengthHeaderFmt, contentStrLen)
		fmt.Println("contentLengthHeader", contentLengthHeader)
	}

	var extraHeaders string
	switch request.CommandName() {
	case "OPTIONS", "ANNOUNCE":
		extraHeaders = "Content-Type: application/sdp\r\n"
	case "DESCRIBE":
		extraHeaders = "Accept: application/sdp\r\n"
	case "SETUP":
		subsession := request.Subsession()
		this.constructSubSessionURL(subsession)
	case "PLAY":
		//sessionStr := this.createSessionString(sessionId)
	case "GET", "POST":
	default:
	}

	authenticatorStr := this.createAuthenticatorStr()

	cmdFmt := "%s %s %s\r\n" +
		"CSeq: %d\r\n" +
		"%s" +
		"%s" +
		"%s" +
		"%s" +
		"\r\n" +
		"%s"

	cmd := fmt.Sprintf(cmdFmt, request.CommandName(),
		this.baseURL,
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
	//fmt.Println(cmd, writeBytes)

	if this.tunnelOverHTTPPortNum == 0 {
		this.requestsAwaitingResponse.enqueue(request)
	}

	return writeBytes
}

func (this *RTSPClient) sessionURL(session *MediaSession) string {
	url := session.ControlPath()
	if url == "" || url == "*" {
		url = this.baseURL
	}
	return url
}

func (this *RTSPClient) constructSubSessionURL(subsession *MediaSubSession) (string, string, string) {
	prefix := "" //this.sessionURL(subsession.parentSession())
	suffix := subsession.ControlPath()
	separator := ""
	return prefix, separator, suffix
}

func (this *RTSPClient) createSessionString(sessionId string) string {
	var sessionStr string
	if sessionId != "" {
		sessionStr = fmt.Sprintf("Session: %s\r\n", sessionId)
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
		ret, _ := fmt.Sscanf(line, "RTSP/%s %d", &version, &responseCode)
		if ret == 2 {
			result = true
			break
		}

		ret, _ = fmt.Sscanf(line, "HTTP/%s %d", &version, &responseCode)
		if ret != 2 {
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

func (this *RTSPClient) handleIncomingRequest() {
}

func (this *RTSPClient) checkForHeader(line, headerName string, headerNameLength int) (headerParams string, result bool) {
	if !strings.EqualFold(headerName, line[:headerNameLength]) {
		return headerParams, false
	}

	var index int
	for i, c := range line[headerNameLength:] {
		if c != ' ' && c != '\t' {
			index = headerNameLength + i
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
