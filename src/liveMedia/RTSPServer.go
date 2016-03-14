package liveMedia

import (
	"fmt"
	. "groupsock"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"time"
)

type RTSPServer struct {
	urlPrefix                         string
	rtspPort                          int
	httpPort                          int
	rtspListen                        *net.TCPListener
	httpListen                        *net.TCPListener
	clientSessions                    map[string]*RTSPClientSession
	serverMediaSessions               map[string]*ServerMediaSession
	clientConnectionsForHTTPTunneling map[string]*RTSPClientConnection
	reclamationTestSeconds            time.Duration
}

func NewRTSPServer(portNum int) *RTSPServer {
	rtspServer := new(RTSPServer)
	rtspServer.rtspPort = portNum

	var err error
	rtspServer.rtspListen, err = rtspServer.SetupOurSocket(portNum)
	if err != nil {
		return nil
	}

	runtime.GOMAXPROCS(rtspServer.NumCPU())

	rtspServer.StartMonitor()
	rtspServer.clientSessions = make(map[string]*RTSPClientSession)
	rtspServer.serverMediaSessions = make(map[string]*ServerMediaSession)
	rtspServer.clientConnectionsForHTTPTunneling = make(map[string]*RTSPClientConnection)
	rtspServer.reclamationTestSeconds = 65
	return rtspServer
}

func (this *RTSPServer) Start() {
	go this.IncomingConnectionHandler(this.rtspListen)
}

func (this *RTSPServer) StartMonitor() {
	go this.MonitorServe()
}

func (this *RTSPServer) MonitorServe() {
	log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
}

func (this *RTSPServer) SetupOurSocket(portNum int) (*net.TCPListener, error) {
	tcpAddr := fmt.Sprintf("0.0.0.0:%d", portNum)
	addr, _ := net.ResolveTCPAddr("tcp", tcpAddr)

	return net.ListenTCP("tcp", addr)
}

func (this *RTSPServer) SetUpTunnelingOverHTTP(httpPort int) bool {
	this.httpPort = httpPort

	var err error
	this.httpListen, err = this.SetupOurSocket(httpPort)
	if err != nil {
		return false
	}

	go this.IncomingConnectionHandler(this.httpListen)
	return true
}

func (this *RTSPServer) HttpServerPortNum() int {
	return this.httpPort
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

func (this *RTSPServer) IncomingConnectionHandler(serverListen *net.TCPListener) {
	for {
		tcpConn, err := serverListen.AcceptTCP()
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
	// Next, check whether we already have a "ServerMediaSession" for this file:
	sms, smsExists := this.serverMediaSessions[streamName]

	fid, err := os.Open(streamName)
	if err != nil {
		if smsExists {
			this.RemoveServerMediaSession(sms)
		}
		return nil
	}
	defer fid.Close()

	if !smsExists {
		sms = this.CreateNewSMS(streamName)
		this.AddServerMediaSession(sms)
	}

	return sms
}

/*
func (this *RTSPServer) LookupServerMediaSession(streamName string) *ServerMediaSession {
	serverMediaSession, _ := this.serverMediaSessions[streamName]
	return serverMediaSession
}*/

func (this *RTSPServer) AddServerMediaSession(serverMediaSession *ServerMediaSession) {
	sessionName := serverMediaSession.StreamName()

	// in case an existing "ServerMediaSession" with this name already exists
	session, _ := this.serverMediaSessions[sessionName]
	this.RemoveServerMediaSession(session)

	this.serverMediaSessions[sessionName] = serverMediaSession
}

func (this *RTSPServer) RemoveServerMediaSession(serverMediaSession *ServerMediaSession) {
	if serverMediaSession != nil {
		sessionName := serverMediaSession.StreamName()
		delete(this.serverMediaSessions, sessionName)
	}
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
		serverMediaSession.AddSubSession(NewH264FileMediaSubSession(fileName))
	case "ts":
		//indexFileName := fmt.Sprintf("%sx", fileName)
		serverMediaSession = NewServerMediaSession("MPEG Transport Stream", fileName)
		serverMediaSession.AddSubSession(NewM2TSFileMediaSubSession(fileName))
	default:
	}
	return serverMediaSession
}
