package main


type RTSPClientSession struct {
}

func NewRTSPClientSession() *RTSPClientSession {
	return new(RTSPClientSession)
}

func (this *RTSPClientSession) HandleCommandSetup() {
}