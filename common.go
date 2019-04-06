package allocator

import "unsafe"

type arena interface {
	Alloc(size uintptr, alignment uintptr) (APtr, error)
	CurrentOffset() AOffset
	ToRef(p APtr) unsafe.Pointer
	AvailableSize() int
	Capacity() int
}

func calculateRequiredPadding(o AOffset, targetAlignment int) int {
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (int(o.p.offset) % targetAlignment)) % targetAlignment
}
