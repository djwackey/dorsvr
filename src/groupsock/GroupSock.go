package groupsock

type OutputSocket struct {
	sourcePort  uint
	lastSentTTL uint
}

func (this *OutputSocket) write(destAddr string, port int, buffer []byte, bufferSize uint) bool {
	if !writeSocket(destAddr, port, buffer, bufferSize) {
		return false
	}

	return true
}

func (this *OutputSocket) sourcePortNum() uint {
	return this.sourcePort
}

type GroupSock struct {
	OutputSocket
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
	this.write("192.168.1.224", 554, buffer, bufferSize)
	return true
}

func (this *GroupSock) handleRead() {
}

func (this *GroupSock) TTL() uint {
	return this.ttl
}
