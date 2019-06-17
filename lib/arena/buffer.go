package arena

import "unsafe"

type bufferAllocator interface {
	Alloc(size uintptr, alignment uintptr) (Ptr, error)
	ToRef(p Ptr) unsafe.Pointer
	Metrics() Metrics
}

type Buffer struct {
	alloc         bufferAllocator
	currentBuffer Bytes
}

func NewBuffer(target bufferAllocator) *Buffer {
	return &Buffer{alloc: target}
}

func (b *Buffer) init(initSize int) error {
	if b.alloc == nil {
		b.alloc = &Simple{}
	}
	if b.currentBuffer.Cap() == 0 {
		newBuffer, allocErr := MakeBytesWithCapacity(b.alloc, 0, uintptr(initSize))
		if allocErr != nil {
			return allocErr
		}
		b.currentBuffer = newBuffer
	}
	return nil
}

func (b *Buffer) WriteString(s string) (n int, err error) {
	initErr := b.init(len(s))
	if initErr != nil {
		return 0, initErr
	}
	changedBuffer, allocErr := AppendString(b.alloc, b.currentBuffer, s)
	if allocErr != nil {
		return 0, allocErr
	}
	b.currentBuffer = changedBuffer
	return len(s), nil
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	initErr := b.init(len(p))
	if initErr != nil {
		return 0, initErr
	}
	changedBuffer, allocErr := Append(b.alloc, b.currentBuffer, p...)
	if allocErr != nil {
		return 0, allocErr
	}
	b.currentBuffer = changedBuffer
	return len(p), nil
}

func (b *Buffer) Bytes() []byte {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return nil
	}
	return BytesToRef(b.alloc, b.currentBuffer)
}

func (b *Buffer) String() string {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return ""
	}
	return BytesToStringRef(b.alloc, b.currentBuffer)
}

func (b *Buffer) CopyBytesToStringOnHeap() string {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return ""
	}
	return CopyBytesToStringOnHeap(b.alloc, b.currentBuffer)
}

func (b *Buffer) CopyBytesToHeap() []byte {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return nil
	}
	return CopyBytesToHeap(b.alloc, b.currentBuffer)
}

func (b *Buffer) ArenaBytes() Bytes {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return Bytes{}
	}
	return b.currentBuffer
}

func (b *Buffer) Cap() int {
	return b.Cap()
}
