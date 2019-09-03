package etalon

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"reflect"
	"unsafe"
)

type internalPointAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type PointView struct {
	alloc            internalPointAllocator
	lastAllocatedPtr arena.Ptr
}

func NewPointView(alloc internalPointAllocator) *PointView {
	if alloc == nil {
		return &PointView{alloc: &arena.GenericAllocator{}}
	}
	return &PointView{alloc: alloc}
}

func (s *PointView) MakeSlice(len int) ([]Point, error) {
	sliceHdr, allocErr := s.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]Point)(unsafe.Pointer(sliceHdr)), nil
}

func (s *PointView) MakeSliceWithCapacity(length int, capacity int) ([]Point, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeSlice(capacity)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]Point)(unsafe.Pointer(sliceHdr)), nil
}

func (s *PointView) Append(slice []Point, elemsToAppend ...Point) ([]Point, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]Point)(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *PointView) growIfNecessary(slice []Point, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar Point
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	availableSizeInBytes := int(sliceHdr.Cap-sliceHdr.Len) * int(tSize)
	if availableSizeInBytes >= requiredSizeInBytes {
		return sliceHdr, nil
	}

	emptyPtr := arena.Ptr{}
	if s.lastAllocatedPtr != emptyPtr && sliceHdr.Data == uintptr(s.alloc.ToRef(s.lastAllocatedPtr)) {
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
	dst := *(*[]Point)(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *PointView) makeSlice(len int) (*reflect.SliceHeader, error) {
	var tVar Point
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
