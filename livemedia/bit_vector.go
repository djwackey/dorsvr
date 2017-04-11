package livemedia

import (
	"encoding/binary"
)

var maxLength uint = 32
var singleBitMask = [8]byte{0x80, 0x40, 0x20, 0x10, 0x08, 0x04, 0x02, 0x01}

type BitVector struct {
	totNumBits    uint
	curBitIndex   uint
	baseBitOffset uint
	baseByte      []byte
}

func newBitVector(baseByte []byte, baseBitOffset, totNumBits uint) *BitVector {
	return &BitVector{
		baseByte:      baseByte,
		baseBitOffset: baseBitOffset,
		totNumBits:    totNumBits,
	}
}

func (v *BitVector) init(baseByte []byte, baseBitOffset, totNumBits uint) {
	v.baseByte = baseByte
	v.baseBitOffset = baseBitOffset
	v.totNumBits = totNumBits
}

func (v *BitVector) getBits(numBits uint) uint {
	if numBits == 0 {
		return 0
	}

	tmpBuf := make([]byte, 4)

	if numBits > maxLength {
		numBits = maxLength
	}

	var overflowingBits uint
	if numBits > v.totNumBits-v.curBitIndex {
		overflowingBits = numBits - (v.totNumBits - v.curBitIndex)
	}

	v.shiftBits(tmpBuf, v.baseByte, 0, v.baseBitOffset+v.curBitIndex, numBits-overflowingBits)
	v.curBitIndex += (numBits - overflowingBits)

	result := uint(binary.BigEndian.Uint32(tmpBuf))
	result >>= (maxLength - numBits)          // move into low-order part of word
	result &= (0xFFFFFFFF << overflowingBits) // so any overflow bits are 0
	return result
}

// The following is equivalent to "getBits(1)", except faster:
func (v *BitVector) get1Bit() uint {
	if v.curBitIndex >= v.totNumBits { /* overflow */
		return 0
	} else {
		totBitOffset := v.baseBitOffset + v.curBitIndex
		v.curBitIndex++
		curFromByte := v.baseByte[totBitOffset/8]
		result := (curFromByte >> (7 - (totBitOffset % 8))) & 0x01
		return uint(result)
	}
}

func (v *BitVector) get1BitBoolean() bool {
	return (v.get1Bit() != 0)
}

func (v *BitVector) shiftBits(toBaseByte, fromBaseByte []byte, toBitOffset, fromBitOffset, numBits uint) {
	if numBits == 0 {
		return
	}

	/* Note that from and to may overlap, if from>to */
	fromBytePtr := fromBaseByte[fromBitOffset/8:]
	fromBitRem := fromBitOffset % 8
	toBytePtr := toBaseByte[toBitOffset/8:]
	toBitRem := toBitOffset % 8

	for numBits > 0 {
		fromBitMask := singleBitMask[fromBitRem]
		fromBit := fromBytePtr[0] & fromBitMask
		toBitMask := singleBitMask[toBitRem]

		if fromBit != 0 {
			toBytePtr[0] |= toBitMask
		} else {
			toBytePtr[0] &= ^toBitMask
		}

		fromBitRem++
		if fromBitRem == 8 {
			fromBytePtr = fromBytePtr[1:]
			fromBitRem = 0
		}
		toBitRem++
		if toBitRem == 8 {
			toBytePtr = toBytePtr[1:]
			toBitRem = 0
		}
		numBits--
	}
}

func (v *BitVector) skipBits(numBits uint) {
	if numBits > v.totNumBits-v.curBitIndex { /* overflow */
		v.curBitIndex = v.totNumBits
	} else {
		v.curBitIndex += numBits
	}
}

func (v *BitVector) getExpGolomb() uint {
	var numLeadingZeroBits uint
	var codeStart uint = 1

	for v.get1Bit() == 0 && v.curBitIndex < v.totNumBits {
		numLeadingZeroBits++
		codeStart *= 2
	}

	return codeStart - 1 + v.getBits(numLeadingZeroBits)
}
