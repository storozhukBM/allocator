package etalon

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalCircleAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

// CirclePtr, which basically represents an offset of the allocated value Circle
// inside one of the arenas.
//
// CirclePtr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation methods please refer to CircleView.Ptr methods.
//
// CirclePtr can be converted to *Circle or dereferenced by using
// CircleView.Ptr methods, but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
//
// For detailed documentation please refer to
// internalCirclePtrView.DeRef
// and internalCirclePtrView.ToRef
type CirclePtr struct {
	ptr arena.Ptr
}

// CircleBuffer is an analog to []Circle,
// but it represents a slice allocated inside one of the arenas.
// CircleBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation and append methods please refer to CircleView.Buffer methods.
//
// CircleBuffer can be converted to []Circle
// by using CircleView.Buffer.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
type CircleBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

// Len is direct analog to len([]Circle)
func (s CircleBuffer) Len() int {
	return s.len
}

// Cap is direct analog to cap([]Circle)
func (s CircleBuffer) Cap() int {
	return s.cap
}

// SubSlice is an analog to []Circle[low:high]
// Returns sub-slice of the CircleBuffer and panics in case of bounds out of range.
func (s CircleBuffer) SubSlice(low int, high int) CircleBuffer {
	inBounds := low >= 0 && low <= high && high <= int(s.len)
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: slice bounds out of range [%d:%d] with length %d",
			low, high, s.len,
		))
	}
	var tVar Circle
	tSize := unsafe.Sizeof(tVar)
	type internalPtr struct {
		offset    uint32
		bucketIdx uint8
		arenaMask uint16
	}
	currentPtr := *(*internalPtr)(unsafe.Pointer(&s.data))
	newPtr := internalPtr{
		offset:    currentPtr.offset + uint32(low*int(tSize)),
		bucketIdx: currentPtr.bucketIdx,
		arenaMask: currentPtr.arenaMask,
	}
	return CircleBuffer{
		data: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
		len:  high - low,
		cap:  s.cap - low,
	}
}

// Get is an analog to []Circle[idx]
// Returns CirclePtr and panics in case of idx out of range.
func (s CircleBuffer) Get(idx int) CirclePtr {
	inBounds := idx >= 0 && idx < int(s.len)
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: index out of range [%d] with length %d",
			idx, s.len,
		))
	}
	var tVar Circle
	tSize := unsafe.Sizeof(tVar)
	type internalPtr struct {
		offset    uint32
		bucketIdx uint8
		arenaMask uint16
	}
	currentPtr := *(*internalPtr)(unsafe.Pointer(&s.data))
	newPtr := internalPtr{
		offset:    currentPtr.offset + uint32(idx*int(tSize)),
		bucketIdx: currentPtr.bucketIdx,
		arenaMask: currentPtr.arenaMask,
	}
	return CirclePtr{
		ptr: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
	}
}

// CircleView is an allocation view that can be constructed on top of the target allocator
// and then used to allocate Circle, its slices and buffers inside target allocator.
//
// CircleView contains 3 subviews in form on fields.
//
// Ptr - subview to allocate and operate with CirclePtr structures.
// Slice - to allocate []Circle inside target allocator.
// Buffer - to allocate and operate with CircleBuffer inside target allocator.
type CircleView struct {
	Ptr    internalCirclePtrView
	Slice  internalCircleSliceView
	Buffer internalCircleBufferView
}

// NewCircleView creates allocation view on top of target allocator
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

// New allocates Circle inside target allocator and returns CirclePtr to it.
// CirclePtr can be converted to *Circle or dereferenced by using other methods of this view.
func (s *internalCirclePtrView) New() (CirclePtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return CirclePtr{}, allocErr
	}
	ptr := CirclePtr{ptr: slice.data}
	return ptr, nil
}

// Embed copies passed value inside target allocator, and returns CirclePtr to it.
// CirclePtr can be converted to *Circle or dereferenced by using other methods of this view.
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

// DeRef returns value of Circle referenced by CirclePtr.
func (s *internalCirclePtrView) DeRef(allocPtr CirclePtr) Circle {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*Circle)(ref)
	return *valuePtr
}

// ToRef converts CirclePtr to *Circle but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internalCirclePtrView) ToRef(allocPtr CirclePtr) *Circle {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*Circle)(ref)
	return valuePtr
}

type internalCircleSliceView struct {
	state internalCircleState
}

// Make is an analog to make([]Circle, len), but it allocates this slice in the underlying arena.
// Resulting []Circle can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
// For make([]Circle, len, cap) method please refer to the MakeWithCapacity.
func (s *internalCircleSliceView) Make(len int) ([]Circle, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]Circle)(unsafe.Pointer(sliceHdr)), nil
}

// MakeWithCapacity is an analog to make([]Circle, len, cap),
// but it allocates this slice in the underlying arena.
// Resulting []Circle can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
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

// Append is an analog to append([]Circle, ...Circle),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

// Make is an analog to make([]Circle, len),
// but it allocates this slice in the underlying arena,
// and returns CircleBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// CircleBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]Circle, len, cap)
// and append([]Circle, ...Circle) analogs
// please refer to other methods of this subview.
func (s *internalCircleBufferView) Make(len int) (CircleBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return CircleBuffer{}, allocErr
	}
	return sliceHdr, nil
}

// Make is an analog to make([]Circle, len, cap),
// but it allocates this slice in the underlying arena,
// and returns CircleBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// CircleBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]Circle, len)
// and append([]Circle, ...Circle) analogs
// please refer to other methods of this subview.
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

// Append is an analog to append([]Circle, ...Circle),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

// ToRef converts CircleBuffer to []Circle but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
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
