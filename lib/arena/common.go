package arena

import (
	"fmt"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

const AllocationLimitError = Error("allocation limit")
const AllocationInvalidArgumentError = Error("allocation argument is invalid")

type Bytes struct {
	data Ptr
	len  uintptr
	cap  uintptr
}

func (b Bytes) String() string {
	return fmt.Sprintf("{data: %v len: %v cap: %v}", b.data, b.len, b.cap)
}

func (b Bytes) Len() int {
	return int(b.len)
}

func (b Bytes) Cap() int {
	return int(b.cap)
}

type Ptr struct {
	offset    uint32
	bucketIdx uint8

	arenaMask uint16
}

func (p Ptr) String() string {
	return fmt.Sprintf("{mask: %v bucketIdx: %v offset: %v}", p.arenaMask, p.bucketIdx, p.offset)
}

type Offset struct {
	p Ptr
}

func (o Offset) String() string {
	return o.p.String()
}

type Metrics struct {
	UsedBytes      int
	AvailableBytes int
	AllocatedBytes int
	MaxCapacity    int
}

func (p Metrics) String() string {
	return fmt.Sprintf(
		"{UsedBytes: %v AvailableBytes: %v AllocatedBytes %v MaxCapacity %v}",
		p.UsedBytes, p.AvailableBytes, p.AllocatedBytes, p.MaxCapacity,
	)
}

func calculateRequiredPadding(o Offset, targetAlignment int) int {
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (int(o.p.offset) % targetAlignment)) % targetAlignment
}

func clearBytes(buf []byte) {
	if len(buf) == 0 {
		return
	}
	buf[0] = 0
	for bufPart := 1; bufPart < len(buf); bufPart *= 2 {
		copy(buf[bufPart:], buf[:bufPart])
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
