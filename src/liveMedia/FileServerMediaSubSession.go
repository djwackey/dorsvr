package liveMedia

type FileServerMediaSubSession struct {
	OnDemandServerMediaSubSession
	fileName string
	fileSize int64
}

func (this *FileServerMediaSubSession) InitFileServerMediaSubSession() {
    this.InitOnDemandServerMediaSubSession()
}
