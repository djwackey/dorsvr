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
	dests   []destRecord
	portNum uint
	ttl     uint
}

func NewGroupSock(addrStr string, portNum uint) *GroupSock {
	gs := new(GroupSock)
	gs.ttl = 255
	gs.portNum = portNum
    gs.addDestination(addrStr, portNum)
	return gs
}

func (this *GroupSock) Output(buffer []byte, bufferSize, ttlToSend uint) bool {
    var writeSuccess bool
    for i:=0; i<len(dests); i++ {
        dest := dests[i]
	    if this.write(dest.addr, dest.port, buffer, bufferSize) {
            writeSuccess = true
        }
    }
	return writeSuccess
}

func (this *GroupSock) handleRead() {
}

func (this *GroupSock) TTL() uint {
	return this.ttl
}

func (this *GroupSock) addDestination(addr string, port uint) {
    this.dests = append(this.dests, NewDestRecord(addr, port))
}

func (this *GroupSock) delDestination(addr string, port uint) {
}

type destRecord struct {
	addrStr string
	portNum uint
}

func NewDestRecord(addr string, port uint) *destRecord {
    dest := new(destRecord)
    dest.addrStr = addr
    dest.portNum = port
    return dest
}
