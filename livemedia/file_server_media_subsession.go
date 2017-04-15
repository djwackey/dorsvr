package livemedia

type FileServerMediaSubsession struct {
	OnDemandServerMediaSubsession
	fileName string
	fileSize int64
}

func (s *FileServerMediaSubsession) initFileServerMediaSubsession(isubsession IServerMediaSubsession, fileName string) {
	s.fileName = fileName
	s.initOnDemandServerMediaSubsession(isubsession)
}
