package etalon

import (
	"reflect"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalCoordinateAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type coordinatePtr struct {
	ptr arena.Ptr
}

type coordinateBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

func (s coordinateBuffer) Len() int {
	return s.len
}

func (s coordinateBuffer) Cap() int {
	return s.cap
}

type coordinateView struct {
	Ptr    internalCoordinatePtrView
	Slice  internalCoordinateSliceView
	Buffer internalCoordinateBufferView
}

func newCoordinateView(alloc internalCoordinateAllocator) *coordinateView {
	if alloc == nil {
		state := internalCoordinateState{alloc: &arena.GenericAllocator{}}
		return &coordinateView{
			Ptr:    internalCoordinatePtrView{state: state},
			Slice:  internalCoordinateSliceView{state: state},
			Buffer: internalCoordinateBufferView{state: state},
		}
	}
	state := internalCoordinateState{alloc: alloc}
	return &coordinateView{
		Ptr:    internalCoordinatePtrView{state: state},
		Slice:  internalCoordinateSliceView{state: state},
		Buffer: internalCoordinateBufferView{state: state},
	}
}

type internalCoordinatePtrView struct {
	state internalCoordinateState
}

func (s *internalCoordinatePtrView) New() (coordinatePtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return coordinatePtr{}, allocErr
	}
	ptr := coordinatePtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalCoordinatePtrView) Embed(value coordinate) (coordinatePtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return coordinatePtr{}, allocErr
	}
	valueInPool := (*coordinate)(s.state.alloc.ToRef(slice.data))
	*valueInPool = value
	ptr := coordinatePtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalCoordinatePtrView) DeRef(allocPtr coordinatePtr) coordinate {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*coordinate)(ref)
	return *valuePtr
}

func (s *internalCoordinatePtrView) ToRef(allocPtr coordinatePtr) *coordinate {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*coordinate)(ref)
	return valuePtr
}

type internalCoordinateSliceView struct {
	state internalCoordinateState
}

func (s *internalCoordinateSliceView) Make(len int) ([]coordinate, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]coordinate)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalCoordinateSliceView) MakeWithCapacity(length int, capacity int) ([]coordinate, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeGoSlice(capacity)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]coordinate)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalCoordinateSliceView) Append(slice []coordinate, elemsToAppend ...coordinate) ([]coordinate, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]coordinate)(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *internalCoordinateSliceView) growIfNecessary(slice []coordinate, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar coordinate
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	availableSizeInBytes := int(sliceHdr.Cap-sliceHdr.Len) * int(tSize)
	if availableSizeInBytes >= requiredSizeInBytes {
		return sliceHdr, nil
	}

	emptyPtr := arena.Ptr{}
	if s.state.lastAllocatedPtr != emptyPtr && sliceHdr.Data == uintptr(s.state.alloc.ToRef(s.state.lastAllocatedPtr)) {
		nextPtr, probeAllocErr := s.state.alloc.Alloc(0, 1)
		if probeAllocErr != nil {
			return nil, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		nextPtrAddr := uintptr(s.state.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceHdr.Data+(uintptr(sliceHdr.Cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.state.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.state.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return nil, enhancingErr
			}
			sliceHdr.Cap += requiredLen
			return sliceHdr, nil
		}
	}
	newDstSlice, allocErr := s.makeGoSlice(2 * (int(sliceHdr.Cap) + requiredLen))
	if allocErr != nil {
		return nil, allocErr
	}
	dst := *(*[]coordinate)(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *internalCoordinateSliceView) makeGoSlice(len int) (*reflect.SliceHeader, error) {
	valueSlice, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceRef := s.state.alloc.ToRef(valueSlice.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(sliceRef),
		Len:  len,
		Cap:  len,
	}
	return &sliceHdr, nil
}

type internalCoordinateBufferView struct {
	state internalCoordinateState
}

func (s *internalCoordinateBufferView) Make(len int) (coordinateBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return coordinateBuffer{}, allocErr
	}
	return sliceHdr, nil
}

func (s *internalCoordinateBufferView) MakeWithCapacity(length int,
	capacity int) (coordinateBuffer, error) {
	if capacity < length {
		return coordinateBuffer{}, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.state.makeSlice(capacity)
	if allocErr != nil {
		return coordinateBuffer{}, allocErr
	}
	sliceHdr.len = length
	return sliceHdr, nil
}

func (s *internalCoordinateBufferView) Append(
	slice coordinateBuffer,
	elemsToAppend ...coordinate,
) (coordinateBuffer, error) {

	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return coordinateBuffer{}, allocErr
	}
	target.len = slice.len + len(elemsToAppend)
	result := s.ToRef(target)
	copy(result[slice.len:], elemsToAppend)
	return target, nil
}

func (s *internalCoordinateBufferView) ToRef(slice coordinateBuffer) []coordinate {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(dataRef),
		Len:  slice.len,
		Cap:  slice.cap,
	}
	return *(*[]coordinate)(unsafe.Pointer(&sliceHdr))
}

func (s *internalCoordinateBufferView) growIfNecessary(
	slice coordinateBuffer,
	requiredLen int,
) (coordinateBuffer, error) {
	var tVar coordinate
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	availableSizeInBytes := int(slice.cap-slice.len) * int(tSize)
	if availableSizeInBytes >= requiredSizeInBytes {
		return slice, nil
	}

	emptyPtr := arena.Ptr{}
	if s.state.lastAllocatedPtr != emptyPtr && slice.data == s.state.lastAllocatedPtr {
		nextPtr, probeAllocErr := s.state.alloc.Alloc(0, 1)
		if probeAllocErr != nil {
			return coordinateBuffer{}, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		sliceDataAddr := uintptr(s.state.alloc.ToRef(slice.data))
		nextPtrAddr := uintptr(s.state.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceDataAddr+(uintptr(slice.cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.state.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.state.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return coordinateBuffer{}, enhancingErr
			}
			slice.cap += requiredLen
			return slice, nil
		}
	}
	newDstSlice, allocErr := s.state.makeSlice(2 * (int(slice.cap) + requiredLen))
	if allocErr != nil {
		return coordinateBuffer{}, allocErr
	}
	dst := s.ToRef(newDstSlice)
	prev := s.ToRef(slice)
	copy(dst, prev)
	return newDstSlice, nil
}

type internalCoordinateState struct {
	alloc            internalCoordinateAllocator
	lastAllocatedPtr arena.Ptr
}

func (s *internalCoordinateState) makeSlice(len int) (coordinateBuffer, error) {
	var tVar coordinate
	tSize := unsafe.Sizeof(tVar)
	tAlignment := unsafe.Alignof(tVar)
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len)*tSize, tAlignment)
	if allocErr != nil {
		return coordinateBuffer{}, allocErr
	}
	s.lastAllocatedPtr = slicePtr
	sliceHdr := coordinateBuffer{
		data: slicePtr,
		len:  len,
		cap:  len,
	}
	return sliceHdr, nil
}
