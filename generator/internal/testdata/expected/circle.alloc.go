package etalon

import (
	"reflect"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalCircleAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type CirclePtr struct {
	ptr arena.Ptr
}

type CircleBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

func (s CircleBuffer) Len() int {
	return s.len
}

func (s CircleBuffer) Cap() int {
	return s.cap
}

type CircleView struct {
	Ptr    internalCirclePtrView
	Slice  internalCircleSliceView
	Buffer internalCircleBufferView
}

func NewCircleView(alloc internalCircleAllocator) *CircleView {
	if alloc == nil {
		state := internalCircleState{alloc: &arena.GenericAllocator{}}
		return &CircleView{
			Ptr:    internalCirclePtrView{state: state},
			Slice:  internalCircleSliceView{state: state},
			Buffer: internalCircleBufferView{state: state},
		}
	}
	state := internalCircleState{alloc: alloc}
	return &CircleView{
		Ptr:    internalCirclePtrView{state: state},
		Slice:  internalCircleSliceView{state: state},
		Buffer: internalCircleBufferView{state: state},
	}
}

type internalCirclePtrView struct {
	state internalCircleState
}

func (s *internalCirclePtrView) New() (CirclePtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return CirclePtr{}, allocErr
	}
	ptr := CirclePtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalCirclePtrView) Embed(value Circle) (CirclePtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return CirclePtr{}, allocErr
	}
	valueInPool := (*Circle)(s.state.alloc.ToRef(slice.data))
	*valueInPool = value
	ptr := CirclePtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalCirclePtrView) DeRef(allocPtr CirclePtr) Circle {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*Circle)(ref)
	return *valuePtr
}

func (s *internalCirclePtrView) ToRef(allocPtr CirclePtr) *Circle {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*Circle)(ref)
	return valuePtr
}

type internalCircleSliceView struct {
	state internalCircleState
}

func (s *internalCircleSliceView) Make(len int) ([]Circle, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]Circle)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalCircleSliceView) MakeWithCapacity(length int, capacity int) ([]Circle, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeGoSlice(capacity)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]Circle)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalCircleSliceView) Append(slice []Circle, elemsToAppend ...Circle) ([]Circle, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]Circle)(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *internalCircleSliceView) growIfNecessary(slice []Circle, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar Circle
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
	dst := *(*[]Circle)(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *internalCircleSliceView) makeGoSlice(len int) (*reflect.SliceHeader, error) {
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

type internalCircleBufferView struct {
	state internalCircleState
}

func (s *internalCircleBufferView) Make(len int) (CircleBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return CircleBuffer{}, allocErr
	}
	return sliceHdr, nil
}

func (s *internalCircleBufferView) MakeWithCapacity(length int,
	capacity int) (CircleBuffer, error) {
	if capacity < length {
		return CircleBuffer{}, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.state.makeSlice(capacity)
	if allocErr != nil {
		return CircleBuffer{}, allocErr
	}
	sliceHdr.len = length
	return sliceHdr, nil
}

func (s *internalCircleBufferView) Append(
	slice CircleBuffer,
	elemsToAppend ...Circle,
) (CircleBuffer, error) {

	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return CircleBuffer{}, allocErr
	}
	target.len = slice.len + len(elemsToAppend)
	result := s.ToRef(target)
	copy(result[slice.len:], elemsToAppend)
	return target, nil
}

func (s *internalCircleBufferView) ToRef(slice CircleBuffer) []Circle {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(dataRef),
		Len:  slice.len,
		Cap:  slice.cap,
	}
	return *(*[]Circle)(unsafe.Pointer(&sliceHdr))
}

func (s *internalCircleBufferView) growIfNecessary(
	slice CircleBuffer,
	requiredLen int,
) (CircleBuffer, error) {
	var tVar Circle
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
			return CircleBuffer{}, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		sliceDataAddr := uintptr(s.state.alloc.ToRef(slice.data))
		nextPtrAddr := uintptr(s.state.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceDataAddr+(uintptr(slice.cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.state.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.state.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return CircleBuffer{}, enhancingErr
			}
			slice.cap += requiredLen
			return slice, nil
		}
	}
	newDstSlice, allocErr := s.state.makeSlice(2 * (int(slice.cap) + requiredLen))
	if allocErr != nil {
		return CircleBuffer{}, allocErr
	}
	if slice.len > 0 {
		dst := s.ToRef(newDstSlice)
		prev := s.ToRef(slice)
		copy(dst, prev)
	}
	return newDstSlice, nil
}

type internalCircleState struct {
	alloc            internalCircleAllocator
	lastAllocatedPtr arena.Ptr
}

func (s *internalCircleState) makeSlice(len int) (CircleBuffer, error) {
	var tVar Circle
	tSize := unsafe.Sizeof(tVar)
	tAlignment := unsafe.Alignof(tVar)
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len)*tSize, tAlignment)
	if allocErr != nil {
		return CircleBuffer{}, allocErr
	}
	s.lastAllocatedPtr = slicePtr
	sliceHdr := CircleBuffer{
		data: slicePtr,
		len:  len,
		cap:  len,
	}
	return sliceHdr, nil
}
