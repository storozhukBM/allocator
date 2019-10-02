package etalon

import (
	"reflect"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalStablePointsVectorAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type StablePointsVectorPtr struct {
	ptr arena.Ptr
}

type StablePointsVectorBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

func (s StablePointsVectorBuffer) Len() int {
	return s.len
}

func (s StablePointsVectorBuffer) Cap() int {
	return s.cap
}

type StablePointsVectorView struct {
	Ptr    internalStablePointsVectorPtrView
	Slice  internalStablePointsVectorSliceView
	Buffer internalStablePointsVectorBufferView
}

func NewStablePointsVectorView(alloc internalStablePointsVectorAllocator) *StablePointsVectorView {
	if alloc == nil {
		state := internalStablePointsVectorState{alloc: &arena.GenericAllocator{}}
		return &StablePointsVectorView{
			Ptr:    internalStablePointsVectorPtrView{state: state},
			Slice:  internalStablePointsVectorSliceView{state: state},
			Buffer: internalStablePointsVectorBufferView{state: state},
		}
	}
	state := internalStablePointsVectorState{alloc: alloc}
	return &StablePointsVectorView{
		Ptr:    internalStablePointsVectorPtrView{state: state},
		Slice:  internalStablePointsVectorSliceView{state: state},
		Buffer: internalStablePointsVectorBufferView{state: state},
	}
}

type internalStablePointsVectorPtrView struct {
	state internalStablePointsVectorState
}

func (s *internalStablePointsVectorPtrView) New() (StablePointsVectorPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return StablePointsVectorPtr{}, allocErr
	}
	ptr := StablePointsVectorPtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalStablePointsVectorPtrView) Embed(value StablePointsVector) (StablePointsVectorPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return StablePointsVectorPtr{}, allocErr
	}
	valueInPool := (*StablePointsVector)(s.state.alloc.ToRef(slice.data))
	*valueInPool = value
	ptr := StablePointsVectorPtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalStablePointsVectorPtrView) DeRef(allocPtr StablePointsVectorPtr) StablePointsVector {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*StablePointsVector)(ref)
	return *valuePtr
}

func (s *internalStablePointsVectorPtrView) ToRef(allocPtr StablePointsVectorPtr) *StablePointsVector {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*StablePointsVector)(ref)
	return valuePtr
}

type internalStablePointsVectorSliceView struct {
	state internalStablePointsVectorState
}

func (s *internalStablePointsVectorSliceView) Make(len int) ([]StablePointsVector, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]StablePointsVector)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalStablePointsVectorSliceView) MakeWithCapacity(length int, capacity int) ([]StablePointsVector, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeGoSlice(capacity)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]StablePointsVector)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalStablePointsVectorSliceView) Append(slice []StablePointsVector, elemsToAppend ...StablePointsVector) ([]StablePointsVector, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]StablePointsVector)(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *internalStablePointsVectorSliceView) growIfNecessary(slice []StablePointsVector, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar StablePointsVector
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
	dst := *(*[]StablePointsVector)(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *internalStablePointsVectorSliceView) makeGoSlice(len int) (*reflect.SliceHeader, error) {
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

type internalStablePointsVectorBufferView struct {
	state internalStablePointsVectorState
}

func (s *internalStablePointsVectorBufferView) Make(len int) (StablePointsVectorBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return StablePointsVectorBuffer{}, allocErr
	}
	return sliceHdr, nil
}

func (s *internalStablePointsVectorBufferView) MakeWithCapacity(length int,
	capacity int) (StablePointsVectorBuffer, error) {
	if capacity < length {
		return StablePointsVectorBuffer{}, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.state.makeSlice(capacity)
	if allocErr != nil {
		return StablePointsVectorBuffer{}, allocErr
	}
	sliceHdr.len = length
	return sliceHdr, nil
}

func (s *internalStablePointsVectorBufferView) Append(
	slice StablePointsVectorBuffer,
	elemsToAppend ...StablePointsVector,
) (StablePointsVectorBuffer, error) {

	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return StablePointsVectorBuffer{}, allocErr
	}
	target.len = slice.len + len(elemsToAppend)
	result := s.ToRef(target)
	copy(result[slice.len:], elemsToAppend)
	return target, nil
}

func (s *internalStablePointsVectorBufferView) ToRef(slice StablePointsVectorBuffer) []StablePointsVector {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(dataRef),
		Len:  slice.len,
		Cap:  slice.cap,
	}
	return *(*[]StablePointsVector)(unsafe.Pointer(&sliceHdr))
}

func (s *internalStablePointsVectorBufferView) growIfNecessary(
	slice StablePointsVectorBuffer,
	requiredLen int,
) (StablePointsVectorBuffer, error) {
	var tVar StablePointsVector
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
			return StablePointsVectorBuffer{}, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		sliceDataAddr := uintptr(s.state.alloc.ToRef(slice.data))
		nextPtrAddr := uintptr(s.state.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceDataAddr+(uintptr(slice.cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.state.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.state.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return StablePointsVectorBuffer{}, enhancingErr
			}
			slice.cap += requiredLen
			return slice, nil
		}
	}
	newDstSlice, allocErr := s.state.makeSlice(2 * (int(slice.cap) + requiredLen))
	if allocErr != nil {
		return StablePointsVectorBuffer{}, allocErr
	}
	dst := s.ToRef(newDstSlice)
	prev := s.ToRef(slice)
	copy(dst, prev)
	return newDstSlice, nil
}

type internalStablePointsVectorState struct {
	alloc            internalStablePointsVectorAllocator
	lastAllocatedPtr arena.Ptr
}

func (s *internalStablePointsVectorState) makeSlice(len int) (StablePointsVectorBuffer, error) {
	var tVar StablePointsVector
	tSize := unsafe.Sizeof(tVar)
	tAlignment := unsafe.Alignof(tVar)
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len)*tSize, tAlignment)
	if allocErr != nil {
		return StablePointsVectorBuffer{}, allocErr
	}
	s.lastAllocatedPtr = slicePtr
	sliceHdr := StablePointsVectorBuffer{
		data: slicePtr,
		len:  len,
		cap:  len,
	}
	return sliceHdr, nil
}
