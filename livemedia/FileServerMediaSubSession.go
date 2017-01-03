package livemedia

type FileServerMediaSubSession struct {
	OnDemandServerMediaSubSession
	fileName string
	fileSize int64
}

func (s *FileServerMediaSubSession) InitFileServerMediaSubSession(isubsession IServerMediaSubSession, fileName string) {
	s.fileName = fileName
	s.InitOnDemandServerMediaSubSession(isubsession)
}
