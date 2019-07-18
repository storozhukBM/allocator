package arena

import (
	"reflect"
	"unsafe"
)

type bufferAllocator interface {
	Alloc(size uintptr, alignment uintptr) (Ptr, error)
	ToRef(p Ptr) unsafe.Pointer
	Metrics() Metrics
}

type BytesView struct {
	alloc bufferAllocator
}

func NewBytesView(alloc bufferAllocator) *BytesView {
	if alloc == nil {
		return &BytesView{alloc: &GenericAllocator{}}
	}
	return &BytesView{alloc: alloc}
}

func (s *BytesView) MakeBytes(len int) (Bytes, error) {
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len), 1)
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	return Bytes{
		data: slicePtr,
		len:  uintptr(len),
		cap:  uintptr(len),
	}, nil
}

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

func (s *BytesView) Append(bytesSlice Bytes, bytesToAppend ...byte) (Bytes, error) {
	target, allocErr := s.growIfNecessary(bytesSlice, len(bytesToAppend))
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	target.len = bytesSlice.len + uintptr(len(bytesToAppend))
	copy(s.BytesToRef(target)[bytesSlice.len:], bytesToAppend)
	return target, nil
}

func (s *BytesView) AppendString(bytesSlice Bytes, str string) (Bytes, error) {
	target, allocErr := s.growIfNecessary(bytesSlice, len(str))
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	target.len = bytesSlice.len + uintptr(len(str))
	copy(s.BytesToRef(target)[bytesSlice.len:], str)
	return target, nil
}

func (s *BytesView) AppendByte(bytesSlice Bytes, byteToAppend byte) (Bytes, error) {
	target, allocErr := s.growIfNecessary(bytesSlice, 1)
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	target.len = bytesSlice.len + 1
	s.BytesToRef(target)[bytesSlice.len] = byteToAppend
	return target, nil
}

func (s *BytesView) Embed(src []byte) (Bytes, error) {
	result, allocErr := s.MakeBytes(len(src))
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	resultAsSlice := s.BytesToRef(result)
	copy(resultAsSlice, src)
	return result, nil
}

func (s *BytesView) EmbedAsBytes(src []byte) ([]byte, error) {
	bytes, allocErr := s.Embed(src)
	if allocErr != nil {
		return nil, allocErr
	}
	return s.BytesToRef(bytes), nil
}

func (s *BytesView) EmbedAsString(src []byte) (string, error) {
	bytes, allocErr := s.Embed(src)
	if allocErr != nil {
		return "", allocErr
	}
	return s.BytesToStringRef(bytes), nil
}

func (s *BytesView) BytesToRef(bytes Bytes) []byte {
	sliceHdr := s.bytesToSliceHeader(bytes)
	return *(*[]byte)(unsafe.Pointer(&sliceHdr))
}

func (s *BytesView) BytesToStringRef(bytes Bytes) string {
	sliceHdr := s.bytesToSliceHeader(bytes)
	return *(*string)(unsafe.Pointer(&sliceHdr))
}

func (s *BytesView) CopyBytesToHeap(bytes Bytes) []byte {
	sliceFromArena := s.BytesToRef(bytes)
	copyOnHeap := make([]byte, bytes.len)
	copy(copyOnHeap, sliceFromArena)
	return copyOnHeap
}

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

	nextPtr, probeAllocErr := s.alloc.Alloc(0, 1)
	if probeAllocErr != nil {
		return Bytes{}, probeAllocErr
	}
	// current allocation offset is the same as previous
	// we can try to just enhance current buffer
	nextAllocationIsRightAfterTargetSlice := nextPtr.offset == target.data.offset+uint32(target.cap)
	if nextAllocationIsRightAfterTargetSlice && s.alloc.Metrics().AvailableBytes >= requiredSize {
		_, enhancingErr := s.alloc.Alloc(uintptr(requiredSize), 1)
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
	copy(s.BytesToRef(newTarget), s.BytesToRef(target))
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
