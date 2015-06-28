package liveMedia

import (
	"fmt"
	. "groupsock"
	"net"
	"runtime"
	"strings"
)

type RTSPServer struct {
	urlPrefix           string
	rtspPort            int
	listen              *net.TCPListener
	clientSessions      map[string]*RTSPClientSession
	serverMediaSessions map[string]*ServerMediaSession
}

func NewRTSPServer(portNum int) *RTSPServer {
	rtspServer := new(RTSPServer)
	rtspServer.rtspPort = portNum
	if rtspServer.SetupOurSocket() != nil {
		return nil
	}

	runtime.GOMAXPROCS(rtspServer.NumCPU())

	rtspServer.clientSessions = make(map[string]*RTSPClientSession)
	rtspServer.serverMediaSessions = make(map[string]*ServerMediaSession)
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

func (this *RTSPServer) NumCPU() int {
	return runtime.NumCPU()
}

func (this *RTSPServer) IncomingConnectionHandler() {
	for {
		tcpConn, err := this.listen.AcceptTCP()
		if err != nil {
			fmt.Println("failed to accept client.")
			continue
		}

		tcpConn.SetReadBuffer(50 * 1024)

		// Create a new object for handling this RTSP connection:
		go this.NewClientConnection(tcpConn)
	}
}

func (this *RTSPServer) NewClientConnection(conn net.Conn) {
	rtspClientConnection := NewRTSPClientConnection(this, conn)
	if rtspClientConnection != nil {
		rtspClientConnection.IncomingRequestHandler()
	}
}

func (this *RTSPServer) LookupServerMediaSession(streamName string) *ServerMediaSession {
	serverMediaSession, _ := this.serverMediaSessions[streamName]
	return serverMediaSession
}

func (this *RTSPServer) AddServerMediaSession(serverMediaSession *ServerMediaSession) {
	sessionName := serverMediaSession.StreamName()
	//this.RemoveServerMediaSession(sessionName); // in case an existing "ServerMediaSession" with this name already exists

	this.serverMediaSessions[sessionName] = serverMediaSession
}

func (this *RTSPServer) RemoveServerMediaSession(serverMediaSession *ServerMediaSession) {
	//delete(this.serverMediaSessions, serverMediaSession)
}

func (this *RTSPServer) CreateNewSMS(fileName string) *ServerMediaSession {
	var serverMediaSession *ServerMediaSession

	array := strings.Split(fileName, ".")
	if len(array) < 2 {
		return nil
	}

	extension := array[1]

	switch extension {
	case "264":
		// Assumed to be a H.264 Video Elementary Stream file:
		serverMediaSession = NewServerMediaSession("H.264 Video", fileName)
		//OutPacketBuffer::maxSize = 100000; // allow for some possibly large H.264 frames
		serverMediaSession.AddSubSession(NewH264FileMediaSubSession())
	default:
	}
	return serverMediaSession
}
