package liveMedia

var MAX_LENGTH uint = 32
var singleBitMask = [8]byte{0x80, 0x40, 0x20, 0x10, 0x08, 0x04, 0x02, 0x01}

type BitVector struct {
	totNumBits    uint
	curBitIndex   uint
	baseBitOffset uint
	baseByte      []byte
}

func NewBitVector(baseByte []byte, baseBitOffset, totNumBits uint) *BitVector {
	bitVector := new(BitVector)
	bitVector.init(baseByte, baseBitOffset, totNumBits)
	return bitVector
}

func (this *BitVector) init(baseByte []byte, baseBitOffset, totNumBits uint) {
	this.baseByte = baseByte
	this.baseBitOffset = baseBitOffset
	this.totNumBits = totNumBits
}

func (this *BitVector) getBits(numBits uint) uint {
	if numBits == 0 {
		return 0
	}

	tmpBuf := []byte{0, 0, 0, 0}

	if numBits > MAX_LENGTH {
		numBits = MAX_LENGTH
	}

	var overflowingBits uint
	if numBits > this.totNumBits-this.curBitIndex {
		overflowingBits = numBits - (this.totNumBits - this.curBitIndex)
	}

	this.shiftBits(tmpBuf, this.baseByte, 0, this.baseBitOffset+this.curBitIndex, numBits-overflowingBits)
	this.curBitIndex += (numBits - overflowingBits)

	result := uint((tmpBuf[0] << 24) | (tmpBuf[1] << 16) | (tmpBuf[2] << 8) | tmpBuf[3])
	result >>= (MAX_LENGTH - numBits)         // move into low-order part of word
	result &= (0xFFFFFFFF << overflowingBits) // so any overflow bits are 0
	return result
}

func (this *BitVector) get1Bit() uint {
	// The following is equivalent to "getBits(1)", except faster:

	if this.curBitIndex >= this.totNumBits { /* overflow */
		return 0
	} else {
		this.curBitIndex++
		totBitOffset := this.baseBitOffset + this.curBitIndex
		curFromByte := this.baseByte[totBitOffset/8]
		result := (curFromByte >> (7 - (totBitOffset % 8))) & 0x01
		return uint(result)
	}
}

func (this *BitVector) get1BitBoolean() bool {
	return (this.get1Bit() != 0)
}

func (this *BitVector) shiftBits(toBaseByte, fromBaseByte []byte, toBitOffset, fromBitOffset, numBits uint) {
	if numBits == 0 {
		return
	}

	/* Note that from and to may overlap, if from>to */
	//fromBytePtr := fromBaseByte[fromBitOffset/8:]
	fromBitRem := fromBitOffset % 8
	//toByte := toBaseByte[toBitOffset/8:]
	toBitRem := toBitOffset % 8

	for numBits > 0 {
		fromBitMask := singleBitMask[fromBitRem]
		fromBit := fromBitMask
		toBitMask := singleBitMask[toBitRem]
		var toBytePtr byte
		if fromBit != 0 {
			//toByte |= toBitMask
		} else {
			toBytePtr &= ^toBitMask
		}

		if fromBitRem == 8 {
			//fromBytePtr++
			fromBitRem = 0
		}
		fromBitRem++
		if toBitRem == 8 {
			toBytePtr++
			toBitRem = 0
		}
		toBitRem++
		numBits--
	}
}

func (this *BitVector) skipBits(numBits uint) {
	if numBits > this.totNumBits-this.curBitIndex { /* overflow */
		this.curBitIndex = this.totNumBits
	} else {
		this.curBitIndex += numBits
	}
}

func (this *BitVector) get_expGolomb() uint {
	var numLeadingZeroBits uint
	var codeStart uint = 1

	for this.get1Bit() == 0 && this.curBitIndex < this.totNumBits {
		numLeadingZeroBits++
		codeStart *= 2
	}

	return codeStart - 1 + this.getBits(numLeadingZeroBits)
}
