package arena

import (
	"errors"
	"fmt"
	"unsafe"
)

var allocationLimit = errors.New("allocation limit")

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

type Raw struct {
	buffer        []byte
	offset        int
	availableSize int
}

func NewRawArena(size uint) *Raw {
	return &Raw{
		buffer:        make([]byte, int(size)),
		availableSize: int(size),
	}
}

func (a *Raw) Alloc(size uintptr, alignment uintptr) (Ptr, error) {
	targetSize := int(size)
	if targetSize > a.availableSize {
		return Ptr{}, allocationLimit
	}

	targetAlignment := int(alignment)
	paddingSize := calculateRequiredPadding(a.CurrentOffset(), targetAlignment)
	a.offset += paddingSize
	a.availableSize -= paddingSize

	allocationOffset := a.offset
	a.offset += targetSize
	a.availableSize -= targetSize
	return Ptr{offset: uint32(allocationOffset)}, nil
}

func (a *Raw) CurrentOffset() Offset {
	return Offset{p: Ptr{offset: uint32(a.offset)}}
}

func (a *Raw) ToRef(p Ptr) unsafe.Pointer {
	return unsafe.Pointer(&a.buffer[int(p.offset)])
}

func (a *Raw) String() string {
	return fmt.Sprintf("rowarena{%v}", a.CurrentOffset())
}

func (a *Raw) Metrics() Metrics {
	return Metrics{
		UsedBytes:      a.offset,
		AvailableBytes: a.availableSize,
		AllocatedBytes: len(a.buffer),
		MaxCapacity:    len(a.buffer),
	}
}
