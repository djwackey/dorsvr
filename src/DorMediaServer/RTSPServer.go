package main

import (
	"fmt"
	. "groupsock"
	"net"
	//env "UsageEnvironment"
)

type RTSPServer struct {
	urlPrefix string
	rtspPort  int
	listen    net.Listener
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
	go this.IncomingConnectionHandler(0)
}

func (this *RTSPServer) SetupOurSocket() error {
	tcpAddr := fmt.Sprintf(":%d", this.rtspPort)
	var err error
	this.listen, err = net.Listen("tcp", tcpAddr)
	if err != nil {
		return err
	}

	return nil
}

func (this *RTSPServer) RtspURLPrefix() string {
	this.urlPrefix, _ = OurIPAddress()
	return fmt.Sprintf("rtsp://%s:%d/", this.urlPrefix, this.rtspPort)
}

func (this *RTSPServer) IncomingConnectionHandler(serverSocket int) {
	for {
		tcpConn, err := this.listen.Accept()
		if err != nil {
			fmt.Println("failed to accept client.")
			continue
		}

		//tcpConn.SetReadBuffer(50*1024)

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
