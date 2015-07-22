package groupsock

type GroupSock struct {
	portNum uint
	ttl     uint
}

func NewGroupSock(portNum uint) *GroupSock {
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
