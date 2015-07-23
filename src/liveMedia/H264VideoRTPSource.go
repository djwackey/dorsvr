package liveMedia

type H264VideoRTPSource struct {
}

func NewH264VideoRTPSource() *H264VideoRTPSource {
	return new(H264VideoRTPSource)
}

type SPropRecord struct {
	sPropLength uint
	sPropBytes  []byte
}

func parseSPropParameterSets(sPropParameterSetsStr string) ([]*SPropRecord, uint) {
	return nil, 0
}
