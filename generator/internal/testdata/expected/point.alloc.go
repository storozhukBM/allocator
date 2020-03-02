package etalon

import (
	"reflect"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalPointAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type PointPtr struct {
	ptr arena.Ptr
}

type PointBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

func (s PointBuffer) Len() int {
	return s.len
}

func (s PointBuffer) Cap() int {
	return s.cap
}

type PointView struct {
	Ptr    internalPointPtrView
	Slice  internalPointSliceView
	Buffer internalPointBufferView
}

func NewPointView(alloc internalPointAllocator) *PointView {
	if alloc == nil {
		state := internalPointState{alloc: &arena.GenericAllocator{}}
		return &PointView{
			Ptr:    internalPointPtrView{state: state},
			Slice:  internalPointSliceView{state: state},
			Buffer: internalPointBufferView{state: state},
		}
	}
	state := internalPointState{alloc: alloc}
	return &PointView{
		Ptr:    internalPointPtrView{state: state},
		Slice:  internalPointSliceView{state: state},
		Buffer: internalPointBufferView{state: state},
	}
}

type internalPointPtrView struct {
	state internalPointState
}

func (s *internalPointPtrView) New() (PointPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return PointPtr{}, allocErr
	}
	ptr := PointPtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalPointPtrView) Embed(value Point) (PointPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return PointPtr{}, allocErr
	}
	valueInPool := (*Point)(s.state.alloc.ToRef(slice.data))
	*valueInPool = value
	ptr := PointPtr{ptr: slice.data}
	return ptr, nil
}

func (s *internalPointPtrView) DeRef(allocPtr PointPtr) Point {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*Point)(ref)
	return *valuePtr
}

func (s *internalPointPtrView) ToRef(allocPtr PointPtr) *Point {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*Point)(ref)
	return valuePtr
}

type internalPointSliceView struct {
	state internalPointState
}

func (s *internalPointSliceView) Make(len int) ([]Point, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]Point)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalPointSliceView) MakeWithCapacity(length int, capacity int) ([]Point, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeGoSlice(capacity)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]Point)(unsafe.Pointer(sliceHdr)), nil
}

func (s *internalPointSliceView) Append(slice []Point, elemsToAppend ...Point) ([]Point, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]Point)(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *internalPointSliceView) growIfNecessary(slice []Point, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar Point
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
	dst := *(*[]Point)(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *internalPointSliceView) makeGoSlice(len int) (*reflect.SliceHeader, error) {
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

type internalPointBufferView struct {
	state internalPointState
}

func (s *internalPointBufferView) Make(len int) (PointBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return PointBuffer{}, allocErr
	}
	return sliceHdr, nil
}

func (s *internalPointBufferView) MakeWithCapacity(length int,
	capacity int) (PointBuffer, error) {
	if capacity < length {
		return PointBuffer{}, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.state.makeSlice(capacity)
	if allocErr != nil {
		return PointBuffer{}, allocErr
	}
	sliceHdr.len = length
	return sliceHdr, nil
}

func (s *internalPointBufferView) Append(
	slice PointBuffer,
	elemsToAppend ...Point,
) (PointBuffer, error) {

	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return PointBuffer{}, allocErr
	}
	target.len = slice.len + len(elemsToAppend)
	result := s.ToRef(target)
	copy(result[slice.len:], elemsToAppend)
	return target, nil
}

func (s *internalPointBufferView) ToRef(slice PointBuffer) []Point {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(dataRef),
		Len:  slice.len,
		Cap:  slice.cap,
	}
	return *(*[]Point)(unsafe.Pointer(&sliceHdr))
}

func (s *internalPointBufferView) growIfNecessary(
	slice PointBuffer,
	requiredLen int,
) (PointBuffer, error) {
	var tVar Point
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
			return PointBuffer{}, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		sliceDataAddr := uintptr(s.state.alloc.ToRef(slice.data))
		nextPtrAddr := uintptr(s.state.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceDataAddr+(uintptr(slice.cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.state.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.state.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return PointBuffer{}, enhancingErr
			}
			slice.cap += requiredLen
			return slice, nil
		}
	}
	newDstSlice, allocErr := s.state.makeSlice(2 * (int(slice.cap) + requiredLen))
	if allocErr != nil {
		return PointBuffer{}, allocErr
	}
	if slice.len > 0 {
		dst := s.ToRef(newDstSlice)
		prev := s.ToRef(slice)
		copy(dst, prev)
	}
	return newDstSlice, nil
}

type internalPointState struct {
	alloc            internalPointAllocator
	lastAllocatedPtr arena.Ptr
}

func (s *internalPointState) makeSlice(len int) (PointBuffer, error) {
	var tVar Point
	tSize := unsafe.Sizeof(tVar)
	tAlignment := unsafe.Alignof(tVar)
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len)*tSize, tAlignment)
	if allocErr != nil {
		return PointBuffer{}, allocErr
	}
	s.lastAllocatedPtr = slicePtr
	sliceHdr := PointBuffer{
		data: slicePtr,
		len:  len,
		cap:  len,
	}
	return sliceHdr, nil
}
