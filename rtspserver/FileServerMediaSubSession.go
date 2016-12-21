package rtspserver

type FileServerMediaSubSession struct {
	OnDemandServerMediaSubSession
	fileName string
	fileSize int64
}

func (this *FileServerMediaSubSession) InitFileServerMediaSubSession(isubsession IServerMediaSubSession, fileName string) {
	this.fileName = fileName
	this.InitOnDemandServerMediaSubSession(isubsession)
}
