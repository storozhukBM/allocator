package arena

type Buffer struct {
	alloc         *BytesView
	currentBuffer Bytes
}

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

func (b *Buffer) Bytes() []byte {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return nil
	}
	return b.alloc.BytesToRef(b.currentBuffer)
}

func (b *Buffer) String() string {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return ""
	}
	return b.alloc.BytesToStringRef(b.currentBuffer)
}

func (b *Buffer) CopyBytesToStringOnHeap() string {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return ""
	}
	return b.alloc.CopyBytesToStringOnHeap(b.currentBuffer)
}

func (b *Buffer) CopyBytesToHeap() []byte {
	if b.alloc == nil || b.currentBuffer.Len() == 0 {
		return nil
	}
	return b.alloc.CopyBytesToHeap(b.currentBuffer)
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
