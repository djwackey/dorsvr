package liveMedia

import (
	"fmt"
	"net"
	"time"
	. "include"
)

// default value; you can reassign this in your application if you need to
var responseBufferSize = 20000

type RTSPClient struct {
	cseq    uint
	baseURL string
    userAgentHeaderStr string
	tcpConn *net.TCPConn
}

type RequestRecord struct {
	cseq        uint
	commandName string
	handler     interface{}
}

func NewRTSPClient(rtspURL, appName string) *RTSPClient {
    rtspClient := new(RTSPClient)
    rtspClient.InitRTSPClient(rtspURL, appName)
	return rtspClient
}

func NewRequestRecord(cseq uint, commandName string, responseHandler interface{}) *RequestRecord {
	return &RequestRecord{cseq, commandName, responseHandler}
}

func (this *RequestRecord) commandName() string {
	return this.commandName
}

func (this *RequestRecord) subsession() string {
	return ""
}

func (this *RTSPClient) InitRTSPClient(rtspURL, appName string) {
    this.cseq = 1
    this.SetBaseURL(rtspURL)

    // Set the "User-Agent:" header to use in each request:
    libName := "Dor Streaming Media v"
    libVersionStr := MEDIA_SERVER_VERSION
    libPrefix := ""
    libSuffix := ""
    if appName != "" {
        libPrefix = " ("
        libSuffix = ")"
    }

    userAgentName = fmt.Sprintf("%s%s%s%s%s", appName, libPrefix, libName, libVersionStr, libSuffix)
    this.SetUserAgentString(userAgentName)
}

func (this *RTSPClient) SendOptionsCommand(responseHandler interface{}) uint {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "OPTIONS", responseHandler))
}

func (this *RTSPClient) SendAnnounceCommand(responseHandler interface{}) uint {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "ANNOUNCE", responseHandler))
}

func (this *RTSPClient) SendDescribeCommand(responseHandler interface{}) uint {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "DESCRIBE", responseHandler))
}

func (this *RTSPClient) SendSetupCommand() {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "SETUP", responseHandler))
}

func (this *RTSPClient) SendPlayCommand() {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "PLAY", responseHandler))
}

func (this *RTSPClient) SendPauseCommand() {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "PAUSE", responseHandler))
}

func (this *RTSPClient) SendRecordCommand() {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "RECORD", responseHandler))
}

func (this *RTSPClient) SendTeardownCommand() {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "TEARDOWN", responseHandler))
}

func (this *RTSPClient) SendSetParameterCommand() {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "SET_PARAMETER", responseHandler))
}

func (this *RTSPClient) SendGetParameterCommand() {
	this.cseq ++
	return this.sendRequest(NewRequestRecord(this.cseq, "GET_PARAMETER", responseHandler))
}

func (this *RTSPClient) SetUserAgentString(userAgentName string) {
    formatStr := "User-Agent: %s\r\n"
    this.userAgentHeaderStr = fmt.Sprintf(formatStr, userAgentName)
}

func (this *RTSPClient) SetBaseURL(url string) {
	this.baseURL = url
}

func (this *RTSPClient) openConnection() {
	//SetupStreamSocket()
	this.connectToServer()
}

func (this *RTSPClient) connectToServer() {
	addr, err := net.ResolveTCPAddr("tcp", this.baseURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	this.tcpConn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		fmt.Println("Failed to connect to server.", this.baseURL, err)
		return
	}
	defer this.tcpConn.Close()

	go this.incomingDataHandler()
}

func (this *RTSPClient) incomingDataHandler() {
	this.handleResponseBytes()
}

func (this *RTSPClient) handleResponseBytes() {
	for {
		fmt.Println("handleResponseBytes")
		time.Sleep(1000 * time.Millisecond)
	}
}

func (this *RTSPClient) sendRequest(request *RequestRecord) uint {
	if this.tcpConn == nil {
		this.connectToServer()
	}

	switch request.commandName() {
	case "OPTIONS", "ANNOUNCE":
		//extraHeaders := "Content-Type: application/sdp\r\n"
	case "DESCRIBE":
		//extraHeaders := "Accept: application/sdp\r\n"
	case "SETUP":
		//subsession := request.subsession()
	case "PLAY":
	case "GET", "POST":
	default:
	}

	cmdFmt := "%s %s %s\r\n"

	writeBytes, err := this.tcpConn.Write([]byte(cmdFmt))
	if err != nil {
		fmt.Println(err, writeBytes)
	}
	return 0
}
