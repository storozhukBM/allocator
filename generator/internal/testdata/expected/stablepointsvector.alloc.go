package etalon

import (
	"fmt"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalStablePointsVectorAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

// StablePointsVectorPtr, which basically represents an offset of the allocated value StablePointsVector
// inside one of the arenas.
//
// StablePointsVectorPtr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation methods please refer to StablePointsVectorView.Ptr methods.
//
// StablePointsVectorPtr can be converted to *StablePointsVector or dereferenced by using
// StablePointsVectorView.Ptr methods, but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
//
// For detailed documentation please refer to
// internalStablePointsVectorPtrView.DeRef
// and internalStablePointsVectorPtrView.ToRef
type StablePointsVectorPtr struct {
	ptr arena.Ptr
}

// StablePointsVectorBuffer is an analog to []StablePointsVector,
// but it represents a slice allocated inside one of the arenas.
// StablePointsVectorBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation and append methods please refer to StablePointsVectorView.Buffer methods.
//
// StablePointsVectorBuffer can be converted to []StablePointsVector
// by using StablePointsVectorView.Buffer.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
type StablePointsVectorBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

// Len is direct analog to len([]StablePointsVector)
func (s StablePointsVectorBuffer) Len() int {
	return s.len
}

// Cap is direct analog to cap([]StablePointsVector)
func (s StablePointsVectorBuffer) Cap() int {
	return s.cap
}

// SubSlice is an analog to []StablePointsVector[low:high]
// Returns sub-slice of the StablePointsVectorBuffer and panics in case of bounds out of range.
func (s StablePointsVectorBuffer) SubSlice(low int, high int) StablePointsVectorBuffer {
	inBounds := low >= 0 && low <= high && high <= s.cap
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: slice bounds out of range [%d:%d] with capacity %d",
			low, high, s.cap,
		))
	}
	var tVar StablePointsVector
	tSize := unsafe.Sizeof(tVar)
	type internalPtr struct {
		offset    uintptr
		bucketIdx uint8
		arenaMask uint16
	}
	currentPtr := *(*internalPtr)(unsafe.Pointer(&s.data))
	newPtr := internalPtr{
		offset:    currentPtr.offset + uintptr(low*int(tSize)),
		bucketIdx: currentPtr.bucketIdx,
		arenaMask: currentPtr.arenaMask,
	}
	return StablePointsVectorBuffer{
		data: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
		len:  high - low,
		cap:  s.cap - low,
	}
}

// Get is an analog to []StablePointsVector[idx]
// Returns StablePointsVectorPtr and panics in case of idx out of range.
func (s StablePointsVectorBuffer) Get(idx int) StablePointsVectorPtr {
	inBounds := idx >= 0 && idx < int(s.len)
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: index out of range [%d] with length %d",
			idx, s.len,
		))
	}
	var tVar StablePointsVector
	tSize := unsafe.Sizeof(tVar)
	type internalPtr struct {
		offset    uintptr
		bucketIdx uint8
		arenaMask uint16
	}
	currentPtr := *(*internalPtr)(unsafe.Pointer(&s.data))
	newPtr := internalPtr{
		offset:    currentPtr.offset + uintptr(idx*int(tSize)),
		bucketIdx: currentPtr.bucketIdx,
		arenaMask: currentPtr.arenaMask,
	}
	return StablePointsVectorPtr{
		ptr: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
	}
}

// StablePointsVectorView is an allocation view that can be constructed on top of the target allocator
// and then used to allocate StablePointsVector, its slices and buffers inside target allocator.
//
// StablePointsVectorView contains 3 subviews in form on fields.
//
// Ptr - subview to allocate and operate with StablePointsVectorPtr structures.
// Slice - to allocate []StablePointsVector inside target allocator.
// Buffer - to allocate and operate with StablePointsVectorBuffer inside target allocator.
type StablePointsVectorView struct {
	Ptr    internalStablePointsVectorPtrView
	Slice  internalStablePointsVectorSliceView
	Buffer internalStablePointsVectorBufferView
}

// NewStablePointsVectorView creates allocation view on top of target allocator
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

// New allocates StablePointsVector inside target allocator and returns StablePointsVectorPtr to it.
// StablePointsVectorPtr can be converted to *StablePointsVector or dereferenced by using other methods of this view.
func (s *internalStablePointsVectorPtrView) New() (StablePointsVectorPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return StablePointsVectorPtr{}, allocErr
	}
	ptr := StablePointsVectorPtr{ptr: slice.data}
	return ptr, nil
}

// Embed copies passed value inside target allocator, and returns StablePointsVectorPtr to it.
// StablePointsVectorPtr can be converted to *StablePointsVector or dereferenced by using other methods of this view.
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

// DeRef returns value of StablePointsVector referenced by StablePointsVectorPtr.
func (s *internalStablePointsVectorPtrView) DeRef(allocPtr StablePointsVectorPtr) StablePointsVector {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*StablePointsVector)(ref)
	return *valuePtr
}

// ToRef converts StablePointsVectorPtr to *StablePointsVector but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internalStablePointsVectorPtrView) ToRef(allocPtr StablePointsVectorPtr) *StablePointsVector {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*StablePointsVector)(ref)
	return valuePtr
}

type internalStablePointsVectorSliceView struct {
	state internalStablePointsVectorState
}

// Make is an analog to make([]StablePointsVector, len), but it allocates this slice in the underlying arena.
// Resulting []StablePointsVector can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
// For make([]StablePointsVector, len, cap) method please refer to the MakeWithCapacity.
func (s *internalStablePointsVectorSliceView) Make(len int) ([]StablePointsVector, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]StablePointsVector)(unsafe.Pointer(sliceHdr)), nil
}

// MakeWithCapacity is an analog to make([]StablePointsVector, len, cap),
// but it allocates this slice in the underlying arena.
// Resulting []StablePointsVector can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
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

// Append is an analog to append([]StablePointsVector, ...StablePointsVector),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

func (s *internalStablePointsVectorSliceView) growIfNecessary(slice []StablePointsVector,
	requiredLen int) (*internalStablePointsVectorSliceHeader, error) {
	var tVar StablePointsVector
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*internalStablePointsVectorSliceHeader)(unsafe.Pointer(&slice))
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

func (s *internalStablePointsVectorSliceView) makeGoSlice(len int) (*internalStablePointsVectorSliceHeader, error) {
	valueSlice, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceRef := s.state.alloc.ToRef(valueSlice.data)
	sliceHdr := internalStablePointsVectorSliceHeader{
		Data: uintptr(sliceRef),
		Len:  len,
		Cap:  len,
	}
	return &sliceHdr, nil
}

type internalStablePointsVectorBufferView struct {
	state internalStablePointsVectorState
}

// Make is an analog to make([]StablePointsVector, len),
// but it allocates this slice in the underlying arena,
// and returns StablePointsVectorBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// StablePointsVectorBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]StablePointsVector, len, cap)
// and append([]StablePointsVector, ...StablePointsVector) analogs
// please refer to other methods of this subview.
func (s *internalStablePointsVectorBufferView) Make(len int) (StablePointsVectorBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return StablePointsVectorBuffer{}, allocErr
	}
	return sliceHdr, nil
}

// Make is an analog to make([]StablePointsVector, len, cap),
// but it allocates this slice in the underlying arena,
// and returns StablePointsVectorBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// StablePointsVectorBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]StablePointsVector, len)
// and append([]StablePointsVector, ...StablePointsVector) analogs
// please refer to other methods of this subview.
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

// Append is an analog to append([]StablePointsVector, ...StablePointsVector),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

// ToRef converts StablePointsVectorBuffer to []StablePointsVector but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internalStablePointsVectorBufferView) ToRef(slice StablePointsVectorBuffer) []StablePointsVector {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := internalStablePointsVectorSliceHeader{
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

	return s.grow(slice, requiredLen)
}

func (s *internalStablePointsVectorBufferView) grow(
	slice StablePointsVectorBuffer,
	requiredLen int,
) (StablePointsVectorBuffer, error) {
	var tVar StablePointsVector
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
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
	if slice.len > 0 {
		dst := s.ToRef(newDstSlice)
		prev := s.ToRef(slice)
		copy(dst, prev)
	}
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

type internalStablePointsVectorSliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
