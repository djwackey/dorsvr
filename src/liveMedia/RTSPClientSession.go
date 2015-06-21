package liveMedia

type RTSPClientSession struct {
	numStreamStates int
}

func NewRTSPClientSession() *RTSPClientSession {
	return new(RTSPClientSession)
}

func (this *RTSPClientSession) HandleCommandSetup() {
}

func (this *RTSPClientSession) HandleCommandWithinSession() {
}

func (this *RTSPClientSession) HandleCommandPlay() {
}

func (this *RTSPClientSession) HandleCommandPause() {
}

func (this *RTSPClientSession) HandleCommandGetParameter() {
}

func (this *RTSPClientSession) HandleCommandSetParameter() {
}

func (this *RTSPClientSession) HandleCommandTearDown() {
}
