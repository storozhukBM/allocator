package etalon

import (
	"reflect"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalCircleColorAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type CircleColorPtr struct {
	ptr arena.Ptr
}

type CircleColorBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

func (s CircleColorBuffer) Len() int {
	return s.len
}

func (s CircleColorBuffer) Cap() int {
	return s.cap
}

type CircleColorView struct {
	Ptr    internalCircleColorPtrView
	Slice  internalCircleColorSliceView
	Buffer internalCircleColorBufferView
}

func NewCircleColorView(alloc internalCircleColorAllocator) *CircleColorView {
	if alloc == nil {
		state := internalCircleColorState{alloc: &arena.GenericAllocator{}}
		return &CircleColorView{
			Ptr:    internalCircleColorPtrView{state: state},
			Slice:  internalCircleColorSliceView{state: state},
			Buffer: internalCircleColorBufferView{state: state},
		}
	}
	state := internalCircleColorState{alloc: alloc}
	return &CircleColorView{
		Ptr:    internalCircleColorPtrView{state: state},
		Slice:  internalCircleColorSliceView{state: state},
		Buffer: internalCircleColorBufferView{state: state},
	}
}

type internalCircleColorPtrView struct {
	state internalCircleColorState
}

func (s *internalCircleColorPtrView) New() (CircleColorPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return CircleColorPtr{}, allocErr
	}
	ptr := CircleColorPtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalCircleColorPtrView) Embed(value CircleColor) (CircleColorPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return CircleColorPtr{}, allocErr
	}
	valueInPool := (*CircleColor)(s.state.alloc.ToRef(slice.data))
	*valueInPool = value
	ptr := CircleColorPtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalCircleColorPtrView) DeRef(allocPtr CircleColorPtr) CircleColor {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*CircleColor)(ref)
	return *valuePtr
}

func (s *internalCircleColorPtrView) ToRef(allocPtr CircleColorPtr) *CircleColor {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*CircleColor)(ref)
	return valuePtr
}

type internalCircleColorSliceView struct {
	state internalCircleColorState
}

func (s *internalCircleColorSliceView) Make(len int) ([]CircleColor, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]CircleColor)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalCircleColorSliceView) MakeWithCapacity(length int, capacity int) ([]CircleColor, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeGoSlice(capacity)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]CircleColor)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalCircleColorSliceView) Append(slice []CircleColor, elemsToAppend ...CircleColor) ([]CircleColor, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]CircleColor)(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *internalCircleColorSliceView) growIfNecessary(slice []CircleColor, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar CircleColor
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
	dst := *(*[]CircleColor)(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *internalCircleColorSliceView) makeGoSlice(len int) (*reflect.SliceHeader, error) {
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

type internalCircleColorBufferView struct {
	state internalCircleColorState
}

func (s *internalCircleColorBufferView) Make(len int) (CircleColorBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return CircleColorBuffer{}, allocErr
	}
	return sliceHdr, nil
}

func (s *internalCircleColorBufferView) MakeWithCapacity(length int,
	capacity int) (CircleColorBuffer, error) {
	if capacity < length {
		return CircleColorBuffer{}, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.state.makeSlice(capacity)
	if allocErr != nil {
		return CircleColorBuffer{}, allocErr
	}
	sliceHdr.len = length
	return sliceHdr, nil
}

func (s *internalCircleColorBufferView) Append(
	slice CircleColorBuffer,
	elemsToAppend ...CircleColor,
) (CircleColorBuffer, error) {

	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return CircleColorBuffer{}, allocErr
	}
	target.len = slice.len + len(elemsToAppend)
	result := s.ToRef(target)
	copy(result[slice.len:], elemsToAppend)
	return target, nil
}

func (s *internalCircleColorBufferView) ToRef(slice CircleColorBuffer) []CircleColor {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(dataRef),
		Len:  slice.len,
		Cap:  slice.cap,
	}
	return *(*[]CircleColor)(unsafe.Pointer(&sliceHdr))
}

func (s *internalCircleColorBufferView) growIfNecessary(
	slice CircleColorBuffer,
	requiredLen int,
) (CircleColorBuffer, error) {
	var tVar CircleColor
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
			return CircleColorBuffer{}, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		sliceDataAddr := uintptr(s.state.alloc.ToRef(slice.data))
		nextPtrAddr := uintptr(s.state.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceDataAddr+(uintptr(slice.cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.state.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.state.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return CircleColorBuffer{}, enhancingErr
			}
			slice.cap += requiredLen
			return slice, nil
		}
	}
	newDstSlice, allocErr := s.state.makeSlice(2 * (int(slice.cap) + requiredLen))
	if allocErr != nil {
		return CircleColorBuffer{}, allocErr
	}
	dst := s.ToRef(newDstSlice)
	prev := s.ToRef(slice)
	copy(dst, prev)
	return newDstSlice, nil
}

type internalCircleColorState struct {
	alloc            internalCircleColorAllocator
	lastAllocatedPtr arena.Ptr
}

func (s *internalCircleColorState) makeSlice(len int) (CircleColorBuffer, error) {
	var tVar CircleColor
	tSize := unsafe.Sizeof(tVar)
	tAlignment := unsafe.Alignof(tVar)
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len)*tSize, tAlignment)
	if allocErr != nil {
		return CircleColorBuffer{}, allocErr
	}
	s.lastAllocatedPtr = slicePtr
	sliceHdr := CircleColorBuffer{
		data: slicePtr,
		len:  len,
		cap:  len,
	}
	return sliceHdr, nil
}
