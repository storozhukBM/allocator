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

// coordinatePtr, which basically represents an offset of the allocated value coordinate
// inside one of the arenas.
//
// coordinatePtr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation methods please refer to coordinateView.Ptr methods.
//
// coordinatePtr can be converted to *coordinate or dereferenced by using
// coordinateView.Ptr methods, but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
//
// For detailed documentation please refer to
// internalCoordinatePtrView.DeRef
// and internalCoordinatePtrView.ToRef
type coordinatePtr struct {
	ptr arena.Ptr
}

// coordinateBuffer is an analog to []coordinate,
// but it represents a slice allocated inside one of the arenas.
// coordinateBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation and append methods please refer to coordinateView.Buffer methods.
//
// coordinateBuffer can be converted to []coordinate
// by using coordinateView.Buffer.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
type coordinateBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

// Len is direct analog to len([]coordinate)
func (s coordinateBuffer) Len() int {
	return s.len
}

// Cap is direct analog to cap([]coordinate)
func (s coordinateBuffer) Cap() int {
	return s.cap
}

// coordinateView is an allocation view that can be constructed on top of the target allocator
// and then used to allocate coordinate, its slices and buffers inside target allocator.
//
// coordinateView contains 3 subviews in form on fields.
//
// Ptr - subview to allocate and operate with coordinatePtr structures.
// Slice - to allocate []coordinate inside target allocator.
// Buffer - to allocate and operate with coordinateBuffer inside target allocator.
type coordinateView struct {
	Ptr    internalCoordinatePtrView
	Slice  internalCoordinateSliceView
	Buffer internalCoordinateBufferView
}

// newCoordinateView creates allocation view on top of target allocator
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

// New allocates coordinate inside target allocator and returns coordinatePtr to it.
// coordinatePtr can be converted to *coordinate or dereferenced by using other methods of this view.
func (s *internalCoordinatePtrView) New() (coordinatePtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return coordinatePtr{}, allocErr
	}
	ptr := coordinatePtr{ptr: slice.data}
	return ptr, nil
}

// Embed copies passed value inside target allocator, and returns coordinatePtr to it.
// coordinatePtr can be converted to *coordinate or dereferenced by using other methods of this view.
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

// DeRef returns value of coordinate referenced by coordinatePtr.
func (s *internalCoordinatePtrView) DeRef(allocPtr coordinatePtr) coordinate {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*coordinate)(ref)
	return *valuePtr
}

// ToRef converts coordinatePtr to *coordinate but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internalCoordinatePtrView) ToRef(allocPtr coordinatePtr) *coordinate {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*coordinate)(ref)
	return valuePtr
}

type internalCoordinateSliceView struct {
	state internalCoordinateState
}

// Make is an analog to make([]coordinate, len), but it allocates this slice in the underlying arena.
// Resulting []coordinate can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
// For make([]coordinate, len, cap) method please refer to the MakeWithCapacity.
func (s *internalCoordinateSliceView) Make(len int) ([]coordinate, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]coordinate)(unsafe.Pointer(sliceHdr)), nil
}

// MakeWithCapacity is an analog to make([]coordinate, len, cap),
// but it allocates this slice in the underlying arena.
// Resulting []coordinate can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
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

// Append is an analog to append([]coordinate, ...coordinate),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

// Make is an analog to make([]coordinate, len),
// but it allocates this slice in the underlying arena,
// and returns coordinateBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// coordinateBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]coordinate, len, cap)
// and append([]coordinate, ...coordinate) analogs
// please refer to other methods of this subview.
func (s *internalCoordinateBufferView) Make(len int) (coordinateBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return coordinateBuffer{}, allocErr
	}
	return sliceHdr, nil
}

// Make is an analog to make([]coordinate, len, cap),
// but it allocates this slice in the underlying arena,
// and returns coordinateBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// coordinateBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]coordinate, len)
// and append([]coordinate, ...coordinate) analogs
// please refer to other methods of this subview.
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

// Append is an analog to append([]coordinate, ...coordinate),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

// ToRef converts coordinateBuffer to []coordinate but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
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
	if slice.len > 0 {
		dst := s.ToRef(newDstSlice)
		prev := s.ToRef(slice)
		copy(dst, prev)
	}
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
