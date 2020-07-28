package arena

import (
	"fmt"
	"reflect"
	"unsafe"
)

type bufferAllocator interface {
	AllocUnaligned(size uintptr) (Ptr, error)
	ToRef(p Ptr) unsafe.Pointer
	Metrics() Metrics
}

// Bytes is an analog to []byte, but it represents a byte slice allocated inside one of the arenas.
// arena.Bytes is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// arena.Bytes can be converted to []byte by using arena.BytesView.BytesToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
// If you want to move a certain arena.Bytes out of arena to the general heap you can use
// arena.BytesView.CopyBytesToHeap method.
//
// arena.Bytes also can be used to represent strings allocated inside arena and converted
// to string using arena.BytesView.BytesToStringRef or arena.BytesView.CopyBytesToStringOnHeap.
type Bytes struct {
	data Ptr
	len  uintptr
	cap  uintptr
}

// String provides a string snapshot of the current arena.Bytes header.
func (b Bytes) String() string {
	return fmt.Sprintf("{data: %v len: %v cap: %v}", b.data, b.len, b.cap)
}

// Len returns the length of the arena.Bytes. Direct analog of len([]byte)
func (b Bytes) Len() int {
	return int(b.len)
}

// Cap returns the capacity of the arena.Bytes. Direct analog of cap([]byte)
func (b Bytes) Cap() int {
	return int(b.cap)
}

// SubSlice is an analog to []byte[low:high]
// Returns sub-slice of the arena.Bytes and panics in case of bounds out of range.
func (b Bytes) SubSlice(low int, high int) Bytes {
	inBounds := low >= 0 && low <= high && high <= int(b.len)
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: slice bounds out of range [%d:%d] with length %d",
			low, high, b.len,
		))
	}
	return Bytes{
		data: Ptr{
			offset:    b.data.offset + uintptr(low),
			bucketIdx: b.data.bucketIdx,
			arenaMask: b.data.arenaMask,
		},
		len: uintptr(high - low),
		cap: b.cap - uintptr(low),
	}
}

// BytesView is an allocation view that can be constructed on top of the target allocator
// and then used to allocate byte slices and strings inside this allocator.
//
// First of all, it operates with arena.Bytes type, which is is an analog to []byte,
// but it represents a byte slice allocated inside one of the arenas.
// For trivial cases, it can work directly with []byte or string.
type BytesView struct {
	alloc bufferAllocator
}

// NewBytesView creates an allocation view that can be constructed on top of the target allocator.
func NewBytesView(alloc bufferAllocator) *BytesView {
	if alloc == nil {
		return &BytesView{alloc: &GenericAllocator{}}
	}
	return &BytesView{alloc: alloc}
}

// MakeBytes is a direct analog of make([]byte, len)
// It allocates a slice with specified length inside your target allocator.
func (s *BytesView) MakeBytes(len int) (Bytes, error) {
	slicePtr, allocErr := s.alloc.AllocUnaligned(uintptr(len))
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	return Bytes{
		data: slicePtr,
		len:  uintptr(len),
		cap:  uintptr(len),
	}, nil
}

// MakeBytesWithCapacity is a direct analog of make([]byte, len, cap)
// It allocates a slice with specified length and capacity inside your target allocator.
func (s *BytesView) MakeBytesWithCapacity(length int, capacity int) (Bytes, error) {
	if capacity < length {
		return Bytes{}, AllocationInvalidArgumentError
	}
	bytes, allocErr := s.MakeBytes(capacity)
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	bytes.len = uintptr(length)
	return bytes, nil
}

// Append is a direct analog of append([]byte, ...byte).
// If necessary, it will allocate additional bytes from underlying allocator.
func (s *BytesView) Append(bytesSlice Bytes, bytesToAppend ...byte) (Bytes, error) {
	target, allocErr := s.growIfNecessary(bytesSlice, len(bytesToAppend))
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	target.len = bytesSlice.len + uintptr(len(bytesToAppend))
	copy(s.BytesToRef(target)[bytesSlice.len:], bytesToAppend)
	return target, nil
}

// AppendString appends bytes from target string to the end of target buffer.
// If necessary, it will allocate additional bytes from underlying allocator.
func (s *BytesView) AppendString(bytesSlice Bytes, str string) (Bytes, error) {
	target, allocErr := s.growIfNecessary(bytesSlice, len(str))
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	target.len = bytesSlice.len + uintptr(len(str))
	copy(s.BytesToRef(target)[bytesSlice.len:], str)
	return target, nil
}

// AppendByte appends one byte to the end of target buffer.
// If necessary, it will allocate additional bytes from underlying allocator.
func (s *BytesView) AppendByte(bytesSlice Bytes, byteToAppend byte) (Bytes, error) {
	target, allocErr := s.growIfNecessary(bytesSlice, 1)
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	target.len = bytesSlice.len + 1
	s.BytesToRef(target)[bytesSlice.len] = byteToAppend
	return target, nil
}

// Embed copies specified bytes to the underlying allocator arena.
//
// It can be used if you need a full copy for future use,
// but you want to eliminate excessive allocations or for future bytes manipulation
// or just to hide this byte slice from GC.
func (s *BytesView) Embed(src []byte) (Bytes, error) {
	result, allocErr := s.MakeBytes(len(src))
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	resultAsSlice := s.BytesToRef(result)
	copy(resultAsSlice, src)
	return result, nil
}

// EmbedAsBytes copies specified bytes to the underlying allocator arena.
//
// It can be used if you need a full copy for future use,
// but you want to eliminate excessive allocations or for future bytes manipulation.
func (s *BytesView) EmbedAsBytes(src []byte) ([]byte, error) {
	bytes, allocErr := s.Embed(src)
	if allocErr != nil {
		return nil, allocErr
	}
	return s.BytesToRef(bytes), nil
}

// EmbedAsString copies specified bytes to the underlying allocator arena and casts them to string.
//
// It can be used if you need a full copy for future use, but you want to eliminate excessive allocations.
func (s *BytesView) EmbedAsString(src []byte) (string, error) {
	bytes, allocErr := s.Embed(src)
	if allocErr != nil {
		return "", allocErr
	}
	return s.BytesToStringRef(bytes), nil
}

// BytesToRef converts arena.Bytes to []byte,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
// If you want to move a certain arena.Bytes out of arena to the general heap you can use
// arena.BytesView.CopyBytesToHeap method.
func (s *BytesView) BytesToRef(bytes Bytes) []byte {
	sliceHdr := s.bytesToSliceHeader(bytes)
	return *(*[]byte)(unsafe.Pointer(&sliceHdr))
}

// BytesToStringRef converts arena.Bytes to string,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
// If you want to move a certain arena.Bytes as a string out of arena to the general heap you can use
// arena.BytesView.CopyBytesToStringOnHeap method.
func (s *BytesView) BytesToStringRef(bytes Bytes) string {
	sliceHdr := s.bytesToSliceHeader(bytes)
	return *(*string)(unsafe.Pointer(&sliceHdr))
}

// CopyBytesToHeap copies Bytes to the general heap. Can be used if you want to pass this Bytes to other goroutine
// or if you want to destroy/recycle underlying arena and left this Bytes accessible.
func (s *BytesView) CopyBytesToHeap(bytes Bytes) []byte {
	sliceFromArena := s.BytesToRef(bytes)
	copyOnHeap := make([]byte, bytes.len)
	copy(copyOnHeap, sliceFromArena)
	return copyOnHeap
}

// CopyBytesToStringOnHeap copies Bytes to the general heap as string.
// Can be used if you want to pass this Bytes to other goroutine
// or if you want to destroy/recycle underlying arena and left this Bytes accessible.
func (s *BytesView) CopyBytesToStringOnHeap(bytes Bytes) string {
	sliceFromArena := s.BytesToRef(bytes)
	copyOnHeap := make([]byte, bytes.len)
	copy(copyOnHeap, sliceFromArena)
	return *(*string)(unsafe.Pointer(&copyOnHeap))
}

func (s *BytesView) growIfNecessary(bytesSlice Bytes, requiredSize int) (Bytes, error) {
	target := bytesSlice
	availableSize := int(target.cap - target.len)
	if availableSize >= requiredSize {
		return target, nil
	}

	nextPtr, probeAllocErr := s.alloc.AllocUnaligned(0)
	if probeAllocErr != nil {
		return Bytes{}, probeAllocErr
	}
	// current allocation offset is the same as previous
	// we can try to just enhance current buffer
	nextAllocationIsRightAfterTargetSlice := nextPtr.offset == target.data.offset+target.cap
	if nextAllocationIsRightAfterTargetSlice && s.alloc.Metrics().AvailableBytes >= requiredSize {
		_, enhancingErr := s.alloc.AllocUnaligned(uintptr(requiredSize))
		if enhancingErr != nil {
			return Bytes{}, enhancingErr
		}
		target.cap += uintptr(requiredSize)
		return target, nil
	}

	newSize := max(2*(int(target.cap)+requiredSize), 2*int(target.cap))
	newTarget, allocErr := s.MakeBytes(newSize)
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	if target.len > 0 {
		copy(s.BytesToRef(newTarget), s.BytesToRef(target))
	}
	target = newTarget

	return target, nil
}

func (s *BytesView) bytesToSliceHeader(bytes Bytes) reflect.SliceHeader {
	sliceBufferRef := s.alloc.ToRef(bytes.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(sliceBufferRef),
		Len:  int(bytes.len),
		Cap:  int(bytes.cap),
	}
	return sliceHdr
}
