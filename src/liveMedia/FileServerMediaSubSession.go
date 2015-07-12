package liveMedia

type FileServerMediaSubSession struct {
	OnDemandServerMediaSubSession
	fileName string
	fileSize int64
}

func (this *FileServerMediaSubSession) InitFileServerMediaSubSession(isubsession IServerMediaSubSession) {
	this.InitOnDemandServerMediaSubSession(isubsession)
}
