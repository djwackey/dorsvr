package groupsock

type GroupSock struct {
	portNum int
	ttl     uint
}

func NewGroupSock(portNum int) *GroupSock {
	gs := new(GroupSock)
	gs.portNum = portNum
	gs.ttl = 255
	return gs
}

func (this *GroupSock) Output(buffer []byte, bufferSize, ttlToSend uint) bool {
	return true
}

func (this *GroupSock) handleRead() {
}

func (this *GroupSock) TTL() uint {
	return this.ttl
}
