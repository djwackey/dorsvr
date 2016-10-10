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

func (server *RTSPServer) Start() {
	go server.IncomingConnectionHandler(server.rtspListen)
}

func (server *RTSPServer) StartMonitor() {
	go server.MonitorServe()
}

func (server *RTSPServer) MonitorServe() {
	log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
}

func (server *RTSPServer) SetupOurSocket(portNum int) (*net.TCPListener, error) {
	tcpAddr := fmt.Sprintf("0.0.0.0:%d", portNum)
	addr, _ := net.ResolveTCPAddr("tcp", tcpAddr)

	return net.ListenTCP("tcp", addr)
}

func (server *RTSPServer) SetUpTunnelingOverHTTP(httpPort int) bool {
	server.httpPort = httpPort

	var err error
	server.httpListen, err = server.SetupOurSocket(httpPort)
	if err != nil {
		return false
	}

	go server.IncomingConnectionHandler(server.httpListen)
	return true
}

func (server *RTSPServer) HttpServerPortNum() int {
	return server.httpPort
}

func (server *RTSPServer) RtspURL(streamName string) string {
	urlPrefix := server.RtspURLPrefix()
	return fmt.Sprintf("%s%s", urlPrefix, streamName)
}

func (server *RTSPServer) RtspURLPrefix() string {
	server.urlPrefix, _ = OurIPAddress()
	return fmt.Sprintf("rtsp://%s:%d/", server.urlPrefix, server.rtspPort)
}

func (server *RTSPServer) NumCPU() int {
	return runtime.NumCPU()
}

func (server *RTSPServer) IncomingConnectionHandler(serverListen *net.TCPListener) {
	for {
		tcpConn, err := serverListen.AcceptTCP()
		if err != nil {
			fmt.Println("failed to accept client.")
			continue
		}

		tcpConn.SetReadBuffer(50 * 1024)

		// Create a new object for handling server RTSP connection:
		go server.NewClientConnection(tcpConn)
	}
}

func (server *RTSPServer) NewClientConnection(conn net.Conn) {
	rtspClientConnection := NewRTSPClientConnection(server, conn)
	if rtspClientConnection != nil {
		rtspClientConnection.IncomingRequestHandler()
	}
}

func (server *RTSPServer) LookupServerMediaSession(streamName string) *ServerMediaSession {
	// Next, check whether we already have a "ServerMediaSession" for server file:
	sms, smsExists := server.serverMediaSessions[streamName]

	fid, err := os.Open(streamName)
	if err != nil {
		if smsExists {
			server.RemoveServerMediaSession(sms)
		}
		return nil
	}
	defer fid.Close()

	if !smsExists {
		sms = server.CreateNewSMS(streamName)
		server.AddServerMediaSession(sms)
	}

	return sms
}

func (server *RTSPServer) AddServerMediaSession(serverMediaSession *ServerMediaSession) {
	sessionName := serverMediaSession.StreamName()

	// in case an existing "ServerMediaSession" with server name already exists
	session, _ := server.serverMediaSessions[sessionName]
	server.RemoveServerMediaSession(session)

	server.serverMediaSessions[sessionName] = serverMediaSession
}

func (server *RTSPServer) RemoveServerMediaSession(serverMediaSession *ServerMediaSession) {
	if serverMediaSession != nil {
		sessionName := serverMediaSession.StreamName()
		delete(server.serverMediaSessions, sessionName)
	}
}

func (server *RTSPServer) CreateNewSMS(fileName string) *ServerMediaSession {
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
