package arena

// BufferWithError is an analog to bytes.Buffer, but it delegates all allocations to the specified allocator.
//
// Important!!! All methods of BufferWithError type will properly return errors instead of panic during allocations.
// This behavior is different from bytes.Buffer.
// If the buffer becomes too large, methods of this type will NOT panic with ErrTooLarge.
// Please refer to arena.Buffer for bytes.Buffer compatible behavior.
// Still, this type should be used if you want to handle errors properly.
//
// It can be used to construct big strings or as a target for encoders, etc.
type BufferWithError struct {
	alloc         *BytesView
	currentBuffer Bytes
}

// NewBuffer creates buffer on top of target allocator
func NewBufferWithError(target bufferAllocator) *BufferWithError {
	return &BufferWithError{alloc: NewBytesView(target)}
}

func (b *BufferWithError) init(initSize int) error {
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
// Important!!! Returned err can be not nil!!! This behavior is different from bytes.Buffer
// Please refer to arena.Buffer for bytes.Buffer compatible behavior.
//
// The return value n is the length of s;
// err can be not nil.
func (b *BufferWithError) WriteString(s string) (n int, err error) {
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
//
// Important!!! Returned err can be not nil!!! This behavior is different from bytes.Buffer
// Please refer to arena.Buffer for bytes.Buffer compatible behavior.
//
// error can be not nil.
func (b *BufferWithError) WriteByte(c byte) error {
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
//
// Important!!! Returned err can be not nil!!! This behavior is different from bytes.Buffer
// Please refer to arena.Buffer for bytes.Buffer compatible behavior.
//
// err can be not nil.
func (b *BufferWithError) Write(p []byte) (n int, err error) {
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
// arena.BufferWithError.CopyBytesToHeap method.
func (b *BufferWithError) Bytes() []byte {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return nil
	}
	return b.alloc.BytesToRef(b.currentBuffer)
}

// String returns a string holding the whole underlying buffer.
// The result string aliases the buffer content and target arena,
// so it is valid only until the next buffer modification or arena.Cleanup
// If you want to move result string out of the arena to the general heap, you can use
// arena.BufferWithError.CopyBytesToStringOnHeap method.
func (b *BufferWithError) String() string {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return "<nil>"
	}
	return b.alloc.BytesToStringRef(b.currentBuffer)
}

// CopyBytesToStringOnHeap returns a general heap copy of the whole underlying buffer as string.
// Can be used if you want to pass this result string to other goroutine
// or if you want to destroy/recycle underlying arena and left this string accessible.
func (b *BufferWithError) CopyBytesToStringOnHeap() string {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return "<nil>"
	}
	return b.alloc.CopyBytesToStringOnHeap(b.currentBuffer)
}

// CopyBytesToHeap returns a general heap copy of the whole underlying buffer.
// Can be used if you want to pass this result bytes to other goroutine
// or if you want to destroy/recycle underlying arena and left this bytes accessible.
func (b *BufferWithError) CopyBytesToHeap() []byte {
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
func (b *BufferWithError) ArenaBytes() Bytes {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return Bytes{}
	}
	return b.currentBuffer
}

// Cap returns the capacity of the buffer's underlying byte slice
func (b *BufferWithError) Cap() int {
	return b.currentBuffer.Cap()
}

// Len returns the number of bytes of the of the underlying buffer
func (b *BufferWithError) Len() int {
	return b.currentBuffer.Len()
}
