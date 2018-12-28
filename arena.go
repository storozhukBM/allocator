package arena

import (
	"reflect"
	"unsafe"
)

type bucket struct {
	buffer []byte
	offset uintptr
}

func (b bucket) availableSize() uintptr {
	return uintptr(len(b.buffer)) - b.offset
}

type Arena struct {
	buckets []*bucket
}

func NewArena() *Arena {
	return &Arena{buckets: []*bucket{{buffer: make([]byte, 16*1024)}}}
}

func (a *Arena) Alloc(size uintptr) unsafe.Pointer {
	b := a.buckets[len(a.buckets)-1]
	if size > b.availableSize() {
		newSize := max(len(b.buffer)*2, int(size)*2)
		newBucket := &bucket{buffer: make([]byte, newSize)}
		a.buckets = append(a.buckets, newBucket)
		b = newBucket
	}
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&b.buffer))
	result := unsafe.Pointer(header.Data + uintptr(b.offset))
	b.offset += size
	return result
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
