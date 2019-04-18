package allocator

import "unsafe"

type arena interface {
	Alloc(size uintptr, alignment uintptr) (APtr, error)
	CurrentOffset() AOffset
	ToRef(p APtr) unsafe.Pointer
	Metrics() ArenaMetrics
}

func calculateRequiredPadding(o AOffset, targetAlignment int) int {
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (int(o.p.offset) % targetAlignment)) % targetAlignment
}
