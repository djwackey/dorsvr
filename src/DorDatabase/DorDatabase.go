package DorDatabase

type ConfFileManager struct {
	host string
	user string
	pswd string
	name string
}

type DataBaseManager struct {
}

func NewConfFileManager() *ConfFileManager {
	return new(ConfFileManager)
}

func NewDataBaseManager() *DataBaseManager {
	return new(DataBaseManager)
}

func (confFileManager *ConfFileManager) ReadConfInfo(configFile string) bool {
	return true
}
