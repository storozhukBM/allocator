package testdata

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"reflect"
	"unsafe"
)

type internalStablePointsVectorAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type StablePointsVectorView struct {
	alloc            internalStablePointsVectorAllocator
	lastAllocatedPtr arena.Ptr
}

func NewStablePointsVectorView(alloc internalStablePointsVectorAllocator) *StablePointsVectorView {
	if alloc == nil {
		return &StablePointsVectorView{alloc: &arena.GenericAllocator{}}
	}
	return &StablePointsVectorView{alloc: alloc}
}

func (s *StablePointsVectorView) MakeSlice(len int) ([]StablePointsVector, error) {
	sliceHdr, allocErr := s.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]StablePointsVector)(unsafe.Pointer(sliceHdr)), nil
}

func (s *StablePointsVectorView) MakeSliceWithCapacity(length int, capacity int) ([]StablePointsVector, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeSlice(length)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]StablePointsVector)(unsafe.Pointer(sliceHdr)), nil
}

func (s *StablePointsVectorView) Append(slice []StablePointsVector, elemsToAppend ...StablePointsVector) ([]StablePointsVector, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]StablePointsVector)(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *StablePointsVectorView) growIfNecessary(slice []StablePointsVector, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar StablePointsVector
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
	dst := *(*[]StablePointsVector)(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *StablePointsVectorView) makeSlice(len int) (*reflect.SliceHeader, error) {
	var tVar StablePointsVector
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
