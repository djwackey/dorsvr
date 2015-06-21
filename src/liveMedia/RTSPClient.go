package liveMedia

import (
    "fmt"
    "net"
    "time"
)

// default value; you can reassign this in your application if you need to
var responseBufferSize = 20000

type RTSPClient struct {
    mCSeq uint
    mBaseURL string
    mTCPConn *net.TCPConn
}

type RequestRecord struct {
    mCSeq uint
    mCommandName string
    mHandler interface{}
}

func NewRTSPClient(rtspURL string) *RTSPClient {
    return &RTSPClient{}
}

func NewRequestRecord(cseq uint, commandName string, responseHandler interface{}) *RequestRecord {
    return &RequestRecord{cseq, commandName, responseHandler}
}

func (this *RequestRecord) commandName() string {
    return this.mCommandName
}

func (this *RequestRecord) subsession() string {
    return ""
}

func (this *RTSPClient) SendOptionsCommand() {
}

func (this *RTSPClient) SendAnnounceCommand(){
}

func (this *RTSPClient) SendDescribeCommand(responseHandler interface{}) uint {
    this.mCSeq += 1
    return this.sendRequest(NewRequestRecord(this.mCSeq, "DESCRIBE", responseHandler))
}

func (this *RTSPClient) SendSetupCommand() {
}

func (this *RTSPClient) SendPlayCommand() {
}

func (this *RTSPClient) SendPauseCommand() {
}

func (this *RTSPClient) SendRecordCommand() {
}

func (this *RTSPClient) SendTeardownCommand() {
}

func (this *RTSPClient) SendSetParameterCommand() {
}

func (this *RTSPClient) SendGetParameterCommand() {
}

func (this *RTSPClient) SetUserAgentString(userAgentName string) {
}

func (this *RTSPClient) SetBaseURL(url string) {
    this.mBaseURL = url
}

func (this *RTSPClient) openConnection() {
    //SetupStreamSocket()
    this.connectToServer()
}

func (this *RTSPClient) connectToServer() {
    addr, err := net.ResolveTCPAddr("tcp", this.mBaseURL)
    if err != nil {
        fmt.Println(err)
        return
    }

    this.mTCPConn, err = net.DialTCP("tcp", nil, addr)
    if err != nil {
        fmt.Println("Failed to connect to server.", this.mBaseURL, err)
        return
    }
    defer this.mTCPConn.Close()

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
    if this.mTCPConn == nil {
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

    writeBytes, err := this.mTCPConn.Write([]byte(cmdFmt))
    if err != nil {
        fmt.Println(err, writeBytes)
    }
    return 0
}
