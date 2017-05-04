package rtspserver

import (
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/djwackey/dorsvr/auth"
	gs "github.com/djwackey/dorsvr/groupsock"
	"github.com/djwackey/dorsvr/livemedia"
	lg "github.com/djwackey/gitea/log"
)

type RTSPServer struct {
	urlPrefix              string
	rtspPort               int
	httpPort               int
	rtspListen             *net.TCPListener
	httpListen             *net.TCPListener
	clientSessions         map[string]*RTSPClientSession
	clientHTTPConnections  map[string]*RTSPClientConnection
	serverMediaSessions    map[string]*livemedia.ServerMediaSession
	reclamationTestSeconds time.Duration
	authDatabase           *auth.Database
	smsMutex               sync.Mutex
	sessionMutex           sync.Mutex
	httpConnectionMutex    sync.Mutex
	rtspConnectionMutex    sync.Mutex
}

func New(authDatabase *auth.Database) *RTSPServer {
	runtime.GOMAXPROCS(runtime.NumCPU())

	return &RTSPServer{
		authDatabase:           authDatabase,
		reclamationTestSeconds: 65,
		clientSessions:         make(map[string]*RTSPClientSession),
		clientHTTPConnections:  make(map[string]*RTSPClientConnection),
		serverMediaSessions:    make(map[string]*livemedia.ServerMediaSession),
	}
}

func (s *RTSPServer) Destroy() {
	s.rtspListen.Close()
	s.httpListen.Close()
}

func (s *RTSPServer) Listen(portNum int) error {
	s.rtspPort = portNum

	var err error
	s.rtspListen, err = s.setupOurSocket(portNum)
	if err == nil {
		s.startMonitor()
	}

	return err
}

func (s *RTSPServer) Start() {
	go s.incomingConnectionHandler(s.rtspListen)
}

func (s *RTSPServer) startMonitor() {
	go s.monitorServe()
}

func (s *RTSPServer) monitorServe() {
	log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
}

func (s *RTSPServer) setupOurSocket(portNum int) (*net.TCPListener, error) {
	tcpAddr := fmt.Sprintf("0.0.0.0:%d", portNum)
	addr, _ := net.ResolveTCPAddr("tcp", tcpAddr)

	return net.ListenTCP("tcp", addr)
}

func (s *RTSPServer) SetupTunnelingOverHTTP(httpPort int) bool {
	s.httpPort = httpPort

	var err error
	s.httpListen, err = s.setupOurSocket(httpPort)
	if err != nil {
		return false
	}

	go s.incomingConnectionHandler(s.httpListen)
	return true
}

func (s *RTSPServer) HTTPServerPortNum() int {
	return s.httpPort
}

func (s *RTSPServer) RtspURL(streamName string) string {
	urlPrefix := s.RtspURLPrefix()
	return fmt.Sprintf("%s%s", urlPrefix, streamName)
}

func (s *RTSPServer) RtspURLPrefix() string {
	s.urlPrefix, _ = gs.OurIPAddress()
	return fmt.Sprintf("rtsp://%s:%d/", s.urlPrefix, s.rtspPort)
}

func (s *RTSPServer) incomingConnectionHandler(l *net.TCPListener) {
	for {
		tcpConn, err := l.AcceptTCP()
		if err != nil {
			lg.Error(0, "failed to accept client.%s", err.Error())
			continue
		}

		tcpConn.SetReadBuffer(50 * 1024)

		// Create a new object for handling server RTSP connection:
		go s.newClientConnection(tcpConn)
	}
}

func (s *RTSPServer) newClientConnection(conn net.Conn) {
	c := newRTSPClientConnection(s, conn)
	if c != nil {
		c.incomingRequestHandler()
	}
}

func (s *RTSPServer) getServerMediaSession(streamName string) (sms *livemedia.ServerMediaSession, existed bool) {
	s.smsMutex.Lock()
	defer s.smsMutex.Unlock()
	sms, existed = s.serverMediaSessions[streamName]
	return
}

func (s *RTSPServer) lookupServerMediaSession(streamName string) *livemedia.ServerMediaSession {
	// Next, check whether we already have a "ServerMediaSession" for server file:
	sms, existed := s.getServerMediaSession(streamName)

	fid, err := os.Open(streamName)
	if err != nil {
		if existed {
			s.removeServerMediaSession(streamName)
		}
		return nil
	}
	defer fid.Close()

	if !existed {
		sms = s.createNewSMS(streamName)
		s.addServerMediaSession(sms)
	}

	return sms
}

func (s *RTSPServer) addServerMediaSession(sms *livemedia.ServerMediaSession) {
	sessionName := sms.StreamName()

	s.smsMutex.Lock()
	defer s.smsMutex.Unlock()
	s.serverMediaSessions[sessionName] = sms
}

func (s *RTSPServer) removeServerMediaSession(sessionName string) {
	s.smsMutex.Lock()
	defer s.smsMutex.Unlock()
	delete(s.serverMediaSessions, sessionName)
}

func (s *RTSPServer) getClientSession(sessionID string) (clientSession *RTSPClientSession, existed bool) {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()
	clientSession, existed = s.clientSessions[sessionID]
	return
}

func (s *RTSPServer) addClientSession(sessionID string, clientSession *RTSPClientSession) {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()
	s.clientSessions[sessionID] = clientSession
}

func (s *RTSPServer) removeClientSession(sessionID string) {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()
	delete(s.clientSessions, sessionID)
}

func (s *RTSPServer) createNewSMS(fileName string) (sms *livemedia.ServerMediaSession) {
	array := strings.Split(fileName, ".")
	if len(array) < 2 {
		return
	}

	extension := array[1]
	switch extension {
	case "264":
		// Assumed to be a H.264 Video Elementary Stream file:
		sms = livemedia.NewServerMediaSession("H.264 Video", fileName)
		// allow for some possibly large H.264 frames
		livemedia.OutPacketBufferMaxSize = 2000000
		sms.AddSubsession(livemedia.NewH264FileMediaSubsession(fileName))
	case "ts":
		//indexFileName := fmt.Sprintf("%sx", fileName)
		sms = livemedia.NewServerMediaSession("MPEG Transport Stream", fileName)
		sms.AddSubsession(livemedia.NewM2TSFileMediaSubsession(fileName))
	default:
	}
	return
}

func (s *RTSPServer) specialClientAccessCheck(clientSocket net.Conn, clientAddr, urlSuffix string) bool {
	return true
}
