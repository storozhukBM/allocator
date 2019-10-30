package arena

// Buffer is an analog to bytes.Buffer, but it delegates all allocations to the specified allocator.
// It can be used to construct big strings or as a target for encoders, etc.
type Buffer struct {
	alloc         *BytesView
	currentBuffer Bytes
}

// NewBuffer creates buffer on top of target allocator
func NewBuffer(target bufferAllocator) *Buffer {
	return &Buffer{alloc: NewBytesView(target)}
}

func (b *Buffer) init(initSize int) error {
	if b.alloc == nil {
		b.alloc = NewBytesView(&GenericAllocator{})
	}
	if b.currentBuffer.Cap() == 0 {
		newBuffer, allocErr := b.alloc.MakeBytesWithCapacity(0, initSize)
		if allocErr != nil {
			return allocErr
		}
		b.currentBuffer = newBuffer
	}
	return nil
}

// WriteString appends the contents of s to the buffer, growing the buffer as
// needed inside target arena.
//
// The return value n is the length of s;
// err can be not nil. TODO: this should be fixed to comply with bytes.Buffer behaviour
func (b *Buffer) WriteString(s string) (n int, err error) {
	initErr := b.init(len(s))
	if initErr != nil {
		return 0, initErr
	}
	changedBuffer, allocErr := b.alloc.AppendString(b.currentBuffer, s)
	if allocErr != nil {
		return 0, allocErr
	}
	b.currentBuffer = changedBuffer
	return len(s), nil
}

// WriteByte appends the byte c to the buffer, growing the buffer as needed.
// The return value n is the length of s;
// err can be not nil. TODO: this should be fixed to comply with bytes.Buffer behaviour
func (b *Buffer) WriteByte(c byte) error {
	initErr := b.init(1)
	if initErr != nil {
		return initErr
	}
	changedBuffer, allocErr := b.alloc.AppendByte(b.currentBuffer, c)
	if allocErr != nil {
		return allocErr
	}
	b.currentBuffer = changedBuffer
	return nil
}

// Write appends the contents of p to the buffer, growing the buffer as
// needed. The return value n is the length of p;
// err can be not nil. TODO: this should be fixed to comply with bytes.Buffer behaviour
func (b *Buffer) Write(p []byte) (n int, err error) {
	initErr := b.init(len(p))
	if initErr != nil {
		return 0, initErr
	}
	changedBuffer, allocErr := b.alloc.Append(b.currentBuffer, p...)
	if allocErr != nil {
		return 0, allocErr
	}
	b.currentBuffer = changedBuffer
	return len(p), nil
}

// Bytes returns a slice holding the whole underlying buffer.
// The result slice aliases the buffer content and target arena,
// so it is valid only until the next buffer modification or arena.Cleanup
// If you want to move result bytes out of the arena to the general heap, you can use
// arena.Buffer.CopyBytesToHeap method.
func (b *Buffer) Bytes() []byte {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return nil
	}
	return b.alloc.BytesToRef(b.currentBuffer)
}

// String returns a string holding the whole underlying buffer.
// The result string aliases the buffer content and target arena,
// so it is valid only until the next buffer modification or arena.Cleanup
// If you want to move result string out of the arena to the general heap, you can use
// arena.Buffer.CopyBytesToStringOnHeap method.
func (b *Buffer) String() string {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return "<nil>"
	}
	return b.alloc.BytesToStringRef(b.currentBuffer)
}

// CopyBytesToStringOnHeap returns a general heap copy of the whole underlying buffer as string.
// Can be used if you want to pass this result string to other goroutine
// or if you want to destroy/recycle underlying arena and left this string accessible.
func (b *Buffer) CopyBytesToStringOnHeap() string {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return "<nil>"
	}
	return b.alloc.CopyBytesToStringOnHeap(b.currentBuffer)
}

// CopyBytesToHeap returns a general heap copy of the whole underlying buffer.
// Can be used if you want to pass this result bytes to other goroutine
// or if you want to destroy/recycle underlying arena and left this bytes accessible.
func (b *Buffer) CopyBytesToHeap() []byte {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return nil
	}
	return b.alloc.CopyBytesToHeap(b.currentBuffer)
}

// ArenaBytes returns the whole underlying buffer as arena.Bytes
//
// It can be used if you need a full copy for future use,
// but you want to eliminate excessive allocations or for future bytes manipulation
// or just to hide this byte slice from GC.
func (b *Buffer) ArenaBytes() Bytes {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return Bytes{}
	}
	return b.currentBuffer
}

// Cap returns the capacity of the buffer's underlying byte slice
func (b *Buffer) Cap() int {
	return b.currentBuffer.Cap()
}

// Len returns the number of bytes of the of the underlying buffer
func (b *Buffer) Len() int {
	return b.currentBuffer.Len()
}
