package liveMedia

const (
    // RTCP packet types:
    RTCP_PT_SR = 200
    RTCP_PT_RR = 201
    RTCP_PT_SDES = 202
    RTCP_PT_BYE = 203
    RTCP_PT_APP = 204

    // SDES tags:
    RTCP_SDES_END = 0
    RTCP_SDES_CNAME = 1
    RTCP_SDES_NAME = 2
    RTCP_SDES_EMAIL = 3
    RTCP_SDES_PHONE = 4
    RTCP_SDES_LOC = 5
    RTCP_SDES_TOOL = 6
    RTCP_SDES_NOTE = 7
    RTCP_SDES_PRIV = 8
)

type RTCPInstance struct {
}

func NewRTCPInstance() *RTCPInstance {
    return &RTCPInstance{}
}

func (this *RTCPInstance) setByeHandler() {
}

func (this *RTCPInstance) setSRHandler() {
}

func (this *RTCPInstance) setRRHandler() {
}
