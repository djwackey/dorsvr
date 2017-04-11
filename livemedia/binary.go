package livemedia

import (
	"encoding/binary"
	"io"
)

var (
	DefaultEndianness = binary.BigEndian
)

func Uint16(b []byte) uint16 {
	return DefaultEndianness.Uint16(pad(b, 2))
}

func Uint32(b []byte) uint32 {
	return DefaultEndianness.Uint32(pad(b, 4))
}

func LittleEndianUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(padl(b, 4))
}

func Uint64(b []byte) uint64 {
	return DefaultEndianness.Uint64(pad(b, 8))
}

func PutUint8(n byte, w io.Writer) (int, error) {
	return w.Write([]byte{n})
}

func PutUint16(n uint16, w io.Writer) (int, error) {
	buffer := make([]byte, 2)
	DefaultEndianness.PutUint16(buffer, n)

	return w.Write(buffer)
}

func PutUint24(n uint32, w io.Writer) (int, error) {
	buf := []byte{
		byte((n >> 16) & 0xff),
		byte((n >> 8) & 0xff),
		byte((n >> 0) & 0xff),
	}

	return w.Write(buf)
}

func PutUint32(n uint32, w io.Writer) (int, error) {
	buf := make([]byte, 4)
	DefaultEndianness.PutUint32(buf, n)

	return w.Write(buf)
}

func LittleEndianPutUint32(n uint32, w io.Writer) (int, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, n)

	return w.Write(buf)
}

func pad(b []byte, n int) []byte {
	return append(make([]byte, n-len(b)), b...)
}

func padl(b []byte, n int) []byte {
	return append(b, make([]byte, n-len(b))...)
}
