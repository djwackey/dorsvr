package liveMedia

import (
	"fmt"
	. "groupsock"
	. "include"
	"net"
	//"time"
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
	handler     interface{} //ResponseHandler
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

/*
func (this *RequestRecord) URL() string {
	return this.baseURL
}*/

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
	return this.connectToServer()
}

func (this *RTSPClient) connectToServer() bool {
	rtspUrl, result := this.parseRTSPURL(this.baseURL)
	if !result {
		return false
	}

	tcpAddr := fmt.Sprintf("%s:%d", rtspUrl.address, rtspUrl.port)
	addr, err := net.ResolveTCPAddr("tcp", tcpAddr)
	if err != nil {
		fmt.Println(err)
		return false
	}

	this.tcpConn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Println("Failed to connect to server.", this.baseURL, err)
		return false
	}

	//fmt.Println(rtspUrl)
	//defer this.tcpConn.Close()

	go this.incomingDataHandler()
	return true
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
	/*readBytes :=*/ ReadSocket(this.tcpConn, this.responseBuffer)
	this.handleResponseBytes()
}

func (this *RTSPClient) handleResponseBytes() {
	defer this.tcpConn.Close()

	for {
		select {}
	}
}

func (this *RTSPClient) handleRequestError(request *RequestRecord) {
	request.Handle(this, 0, "OK")
}

func (this *RTSPClient) sendRequest(request *RequestRecord) int {
	var connectionIsPending bool
	if !this.requestsAwaitingConnection.isEmpty() {
		connectionIsPending = true
	} else if this.tcpConn == nil {
		if !this.openConnection() {
			return 0
		}
		connectionIsPending = true
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

	contentStr := request.ContentStr()
	contentStrLen := len(contentStr)
	if contentStrLen > 0 {
		contentLengthHeaderFmt := "Content-Length: %s\r\n"
		contentLengthHeader := fmt.Sprintf(contentLengthHeaderFmt, contentStrLen)
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

	cmdFmt := "%s %s %s\r\n" +
		"CSeq: %d\r\n" +
		"%s" +
		"%s" +
		"%s"

	cmd := fmt.Sprintf(cmdFmt, request.CommandName(),
		this.baseURL,
		protocalStr,
		request.CSeq(),
		this.userAgentHeaderStr,
		extraHeaders,
		contentStr)

	writeBytes, err := this.tcpConn.Write([]byte(cmd))
	if err != nil {
		fmt.Println("RTSPClient::sendRequst", err, writeBytes)
	}
	//fmt.Println(cmd, writeBytes)

	this.handleRequestError(request)
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

func (this *RTSPClient) parseResponseCode(line []byte) (bool, int, []byte) {
	var result bool
	var responseCode int
	responseString := line

	for {
		num1, _ := fmt.Sscanf(string(line), "RTSP/%u", &responseCode)
		num2, _ := fmt.Sscanf(string(line), "HTTP/%u", &responseCode)
		if num1 != 1 && num2 != 1 {
			result = true
			break
		}

		// Use everything after the RTSP/* (or HTTP/*) as the response string:
		i := 0
		for string(responseString) != "" && responseString[i] != ' ' && responseString[i] != '\t' {
			i++
		}
		i = 0
		for string(responseString) != "" && (responseString[i] == ' ' || responseString[i] == '\t') {
			i++ // skip whitespace
		}
		break
	}
	return result, responseCode, responseString
}

type RequestQueue struct {
	requestRecords []*RequestRecord
}

func NewRequestQueue() *RequestQueue {
	requestQueue := new(RequestQueue)
	requestQueue.requestRecords = make([]*RequestRecord, 1024)
	return requestQueue
}

func (this *RequestQueue) enqueue(request *RequestRecord) {
	this.requestRecords = append(this.requestRecords, request)
}

func (this *RequestQueue) dequeue() *RequestRecord {
	return nil
}

func (this *RequestQueue) putAtHead(request *RequestRecord) {
}

func (this *RequestQueue) findByCSeq(cseq uint) {
}

func (this *RequestQueue) isEmpty() bool {
	return len(this.requestRecords) < 1
}
