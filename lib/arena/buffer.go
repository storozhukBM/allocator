package arena

import "bytes"

// Buffer is an analog to bytes.Buffer, but it delegates all allocations to the specified allocator.
//
// Important!!! Some methods of Buffer can panic with ErrTooLarge during allocations.
// This is required to be compatible with bytes.Buffer.
// If the buffer becomes too large, methods of this type will panic with ErrTooLarge.
// Please refer to arena.BufferWithError if you want to handle errors properly.
//
// It can be used to construct big strings or as a target for encoders, etc.
type Buffer struct {
	buf BufferWithError
}

// NewBuffer creates buffer on top of target allocator
func NewBuffer(target bufferAllocator) *Buffer {
	return &Buffer{buf: *NewBufferWithError(target)}
}

// WriteString appends the contents of s to the buffer, growing the buffer as
// needed inside target arena.
//
// Important!!! If the buffer becomes too large, methods of this type will panic with ErrTooLarge.
// This is required to be compatible with bytes.Buffer.
// Please refer to arena.BufferWithError if you want to handle errors properly.
//
// The return value n is the length of s;
func (b *Buffer) WriteString(s string) (n int, err error) {
	n, allocErr := b.buf.WriteString(s)
	if allocErr != nil {
		panic(bytes.ErrTooLarge)
	}
	return n, nil
}

// WriteByte appends the byte c to the buffer, growing the buffer as needed.
//
// Important!!! If the buffer becomes too large, methods of this type will panic with ErrTooLarge.
// This is required to be compatible with bytes.Buffer.
// Please refer to arena.BufferWithError if you want to handle errors properly.
func (b *Buffer) WriteByte(c byte) error {
	allocErr := b.buf.WriteByte(c)
	if allocErr != nil {
		panic(bytes.ErrTooLarge)
	}
	return nil
}

// Write appends the contents of p to the buffer, growing the buffer as
// needed. The return value n is the length of p;
//
// Important!!! If the buffer becomes too large, methods of this type will panic with ErrTooLarge.
// This is required to be compatible with bytes.Buffer.
// Please refer to arena.BufferWithError if you want to handle errors properly.
func (b *Buffer) Write(p []byte) (n int, err error) {
	n, allocErr := b.buf.Write(p)
	if allocErr != nil {
		panic(bytes.ErrTooLarge)
	}
	return n, nil
}

// Bytes returns a slice holding the whole underlying buffer.
// The result slice aliases the buffer content and target arena,
// so it is valid only until the next buffer modification or arena.Cleanup
// If you want to move result bytes out of the arena to the general heap, you can use
// arena.BufferWithError.CopyBytesToHeap method.
func (b *Buffer) Bytes() []byte {
	return b.buf.Bytes()
}

// String returns a string holding the whole underlying buffer.
// The result string aliases the buffer content and target arena,
// so it is valid only until the next buffer modification or arena.Cleanup
// If you want to move result string out of the arena to the general heap, you can use
// arena.BufferWithError.CopyBytesToStringOnHeap method.
func (b *Buffer) String() string {
	return b.buf.String()
}

// CopyBytesToStringOnHeap returns a general heap copy of the whole underlying buffer as string.
// Can be used if you want to pass this result string to other goroutine
// or if you want to destroy/recycle underlying arena and left this string accessible.
func (b *Buffer) CopyBytesToStringOnHeap() string {
	return b.buf.CopyBytesToStringOnHeap()
}

// CopyBytesToHeap returns a general heap copy of the whole underlying buffer.
// Can be used if you want to pass this result bytes to other goroutine
// or if you want to destroy/recycle underlying arena and left this bytes accessible.
func (b *Buffer) CopyBytesToHeap() []byte {
	return b.buf.CopyBytesToHeap()
}

// ArenaBytes returns the whole underlying buffer as arena.Bytes
//
// It can be used if you need a full copy for future use,
// but you want to eliminate excessive allocations or for future bytes manipulation
// or just to hide this byte slice from GC.
func (b *Buffer) ArenaBytes() Bytes {
	return b.buf.ArenaBytes()
}

// Cap returns the capacity of the buffer's underlying byte slice
func (b *Buffer) Cap() int {
	return b.buf.Cap()
}

// Len returns the number of bytes of the of the underlying buffer
func (b *Buffer) Len() int {
	return b.buf.Len()
}
