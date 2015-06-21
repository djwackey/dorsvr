package main

import (
	"fmt"
	. "groupsock"
	"net"
)

type RTSPServer struct {
	urlPrefix string
	rtspPort  int
	listen    *net.TCPListener
}

func NewRTSPServer(portNum int) *RTSPServer {
	rtspServer := new(RTSPServer)
	rtspServer.rtspPort = portNum
	if rtspServer.SetupOurSocket() != nil {
		return nil
	}

	return rtspServer
}

func (this *RTSPServer) Start() {
	go this.IncomingConnectionHandler()
}

func (this *RTSPServer) SetupOurSocket() error {
	tcpAddr := fmt.Sprintf("0.0.0.0:%d", this.rtspPort)
    addr, _ := net.ResolveTCPAddr("tcp", tcpAddr)

	var err error
	this.listen, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	return nil
}

func (this *RTSPServer) RtspURL(streamName string) string {
	urlPrefix := this.RtspURLPrefix()
	return fmt.Sprintf("%s%s", urlPrefix, streamName)
}

func (this *RTSPServer) RtspURLPrefix() string {
	this.urlPrefix, _ = OurIPAddress()
	return fmt.Sprintf("rtsp://%s:%d/", this.urlPrefix, this.rtspPort)
}

func (this *RTSPServer) IncomingConnectionHandler() {
	for {
		tcpConn, err := this.listen.AcceptTCP()
		if err != nil {
			fmt.Println("failed to accept client.")
			continue
		}

		tcpConn.SetReadBuffer(50*1024)

		// Create a new object for handling this RTSP connection:
		go this.NewClientConnection(tcpConn)
	}
}

func (this *RTSPServer) NewClientConnection(conn net.Conn) {
	rtspClientConnection := NewRTSPClientConnection(conn)
	if rtspClientConnection != nil {
		rtspClientConnection.IncomingRequestHandler()
	}
}

func (this *RTSPServer) LookupServerMediaSession(streamName string) *ServerMediaSession {
	return nil
}

func (this *RTSPServer) AddServerMediaSession(mediaSession *ServerMediaSession) {
	return
}

func (this *RTSPServer) RemoveServerMediaSession(mediaSession *ServerMediaSession) {
	return
}

func (this *RTSPServer) CreateNewSMS(streamName string) *ServerMediaSession {
	var serverMediaSession *ServerMediaSession
	switch streamName {
	case ".264":
		serverMediaSession = NewServerMediaSession()
		serverMediaSession.AddSubSession(NewH264FileMediaSubSession())
	default:
	}
	return serverMediaSession
}
