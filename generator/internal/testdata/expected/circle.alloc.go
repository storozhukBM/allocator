package testdata

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"reflect"
	"unsafe"
)

type internalCircleAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type CircleView struct {
	alloc            internalCircleAllocator
	lastAllocatedPtr arena.Ptr
}

func NewCircleView(alloc internalCircleAllocator) *CircleView {
	if alloc == nil {
		return &CircleView{alloc: &arena.GenericAllocator{}}
	}
	return &CircleView{alloc: alloc}
}

func (s *CircleView) MakeSlice(len int) ([]Circle, error) {
	sliceHdr, allocErr := s.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]Circle)(unsafe.Pointer(sliceHdr)), nil
}

func (s *CircleView) MakeSliceWithCapacity(length int, capacity int) ([]Circle, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeSlice(length)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]Circle)(unsafe.Pointer(sliceHdr)), nil
}

func (s *CircleView) Append(slice []Circle, elemsToAppend ...Circle) ([]Circle, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]Circle)(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *CircleView) growIfNecessary(slice []Circle, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar Circle
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	availableSize := int(sliceHdr.Cap - sliceHdr.Len)
	if availableSize >= requiredSizeInBytes {
		return sliceHdr, nil
	}

	lastAllocatedRef := uintptr(s.alloc.ToRef(s.lastAllocatedPtr))
	if sliceHdr.Data == lastAllocatedRef {
		nextPtr, probeAllocErr := s.alloc.Alloc(0, 1)
		if probeAllocErr != nil {
			return nil, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		nextPtrAddr := uintptr(s.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceHdr.Data+(uintptr(sliceHdr.Cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return nil, enhancingErr
			}
			sliceHdr.Cap += requiredLen
			return sliceHdr, nil
		}
	}
	newDstSlice, allocErr := s.makeSlice(2 * (int(sliceHdr.Cap) + requiredLen))
	if allocErr != nil {
		return nil, allocErr
	}
	dst := *(*[]Circle)(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *CircleView) makeSlice(len int) (*reflect.SliceHeader, error) {
	var tVar Circle
	tSize := unsafe.Sizeof(tVar)
	tAlignment := unsafe.Alignof(tVar)
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len)*tSize, tAlignment)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceRef := s.alloc.ToRef(slicePtr)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(sliceRef),
		Len:  len,
		Cap:  len,
	}
	return &sliceHdr, nil
}
