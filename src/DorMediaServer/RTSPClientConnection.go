package main

import (
	"fmt"
	"net"
    	//. "groupsock"
)

type RTSPClientConnection struct {
	mClientOutputSocket net.Conn
	mCurrentCSeq        string
	mResponseBuffer     string
}

func NewRTSPClientConnection(socket net.Conn) *RTSPClientConnection {
	return &RTSPClientConnection{mClientOutputSocket: socket}
}

func (this *RTSPClientConnection) IncomingRequestHandler() {
	buffer := make([]byte, 1024)
	isclose := false
	for {
		length, err := this.mClientOutputSocket.Read(buffer[0:1024])

		switch err {
		case nil:
			this.HandleRequestBytes(buffer, length)
		default:
			//err := conn.Close()
			fmt.Println(err.Error())
			if err.Error() == "EOF" {
				isclose = true
			}
		}

		sendBytes, err := this.mClientOutputSocket.Write([]byte(this.mResponseBuffer))
		if err != nil {
			fmt.Println("failed to send response buffer.", sendBytes)
		}

		fmt.Println(this.mResponseBuffer)

		if isclose {
			break
		}
	}

	fmt.Println("end connection.")
}

func (this *RTSPClientConnection) HandleRequestBytes(buf []byte, len int) {
	fmt.Println(string(buf[0:len]))
	requestString, parseSucceeded := ParseRTSPRequestString(buf, len)
	if parseSucceeded {
		this.mCurrentCSeq = "2"
		switch requestString.cmdName {
		case "OPTIONS":
			this.handleCommandOptions()
		case "DESCRIBE":
			this.handleCommandDescribe()
		case "SETUP":
			{
				//sessionId := OurRandom32()
				//clientSession := this.NewClientSession(sessionId)
				//clientSession.HandleCommandSetup()
			}
		case "PLAY", "PAUSE", "TEARDOWN", "GET_PARAMETER", "SET_PARAMETER":
			{
			}
		case "RECORD":
		default:
			this.handleCommandNotSupported()
		}
	}
}

func (this *RTSPClientConnection) handleCommandOptions() {
	this.mResponseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\nCSeq: %s\r\n%sPublic: %s\r\n\r\n", this.mCurrentCSeq, DateHeader(), allowedCommandNames)
}

func (this *RTSPClientConnection) handleCommandDescribe() {
	var rtspURL, sdpDescription string
	var sdpDescriptionSize int
	this.mResponseBuffer = fmt.Sprintf("RTSP/1.0 200 OK\r\nCSeq: %s\r\n"+
		"%s"+
		"Content-Base: %s/\r\n"+
		"Content-Type: application/sdp\r\n"+
		"Content-Length: %d\r\n\r\n"+
		"%s",
		this.mCurrentCSeq, DateHeader(), rtspURL, sdpDescriptionSize, sdpDescription)
}

func (this *RTSPClientConnection) handleCommandNotSupported() {
	this.mResponseBuffer = fmt.Sprintf("RTSP/1.0 405 Method Not Allowed\r\nCSeq: %s\r\n%sAllow: %s\r\n\r\n", this.mCurrentCSeq, DateHeader(), allowedCommandNames)
}

func (this *RTSPClientConnection) NewClientSession(sessionId int32) *RTSPClientSession {
    	return NewRTSPClientSession()
}
