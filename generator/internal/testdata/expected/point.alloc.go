package etalon

import (
	"fmt"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalPointAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

// PointPtr, which basically represents an offset of the allocated value Point
// inside one of the arenas.
//
// PointPtr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation methods please refer to PointView.Ptr methods.
//
// PointPtr can be converted to *Point or dereferenced by using
// PointView.Ptr methods, but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
//
// For detailed documentation please refer to
// internalPointPtrView.DeRef
// and internalPointPtrView.ToRef
type PointPtr struct {
	ptr arena.Ptr
}

// PointBuffer is an analog to []Point,
// but it represents a slice allocated inside one of the arenas.
// PointBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation and append methods please refer to PointView.Buffer methods.
//
// PointBuffer can be converted to []Point
// by using PointView.Buffer.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
type PointBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

// Len is direct analog to len([]Point)
func (s PointBuffer) Len() int {
	return s.len
}

// Cap is direct analog to cap([]Point)
func (s PointBuffer) Cap() int {
	return s.cap
}

// SubSlice is an analog to []Point[low:high]
// Returns sub-slice of the PointBuffer and panics in case of bounds out of range.
func (s PointBuffer) SubSlice(low int, high int) PointBuffer {
	inBounds := low >= 0 && low <= high && high <= s.cap
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: slice bounds out of range [%d:%d] with capacity %d",
			low, high, s.cap,
		))
	}
	var tVar Point
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
	return PointBuffer{
		data: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
		len:  high - low,
		cap:  s.cap - low,
	}
}

// Get is an analog to []Point[idx]
// Returns PointPtr and panics in case of idx out of range.
func (s PointBuffer) Get(idx int) PointPtr {
	inBounds := idx >= 0 && idx < int(s.len)
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: index out of range [%d] with length %d",
			idx, s.len,
		))
	}
	var tVar Point
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
	return PointPtr{
		ptr: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
	}
}

// PointView is an allocation view that can be constructed on top of the target allocator
// and then used to allocate Point, its slices and buffers inside target allocator.
//
// PointView contains 3 subviews in form on fields.
//
// Ptr - subview to allocate and operate with PointPtr structures.
// Slice - to allocate []Point inside target allocator.
// Buffer - to allocate and operate with PointBuffer inside target allocator.
type PointView struct {
	Ptr    internalPointPtrView
	Slice  internalPointSliceView
	Buffer internalPointBufferView
}

// NewPointView creates allocation view on top of target allocator
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

// New allocates Point inside target allocator and returns PointPtr to it.
// PointPtr can be converted to *Point or dereferenced by using other methods of this view.
func (s *internalPointPtrView) New() (PointPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return PointPtr{}, allocErr
	}
	ptr := PointPtr{ptr: slice.data}
	return ptr, nil
}

// Embed copies passed value inside target allocator, and returns PointPtr to it.
// PointPtr can be converted to *Point or dereferenced by using other methods of this view.
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

// DeRef returns value of Point referenced by PointPtr.
func (s *internalPointPtrView) DeRef(allocPtr PointPtr) Point {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*Point)(ref)
	return *valuePtr
}

// ToRef converts PointPtr to *Point but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internalPointPtrView) ToRef(allocPtr PointPtr) *Point {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*Point)(ref)
	return valuePtr
}

type internalPointSliceView struct {
	state internalPointState
}

// Make is an analog to make([]Point, len), but it allocates this slice in the underlying arena.
// Resulting []Point can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
// For make([]Point, len, cap) method please refer to the MakeWithCapacity.
func (s *internalPointSliceView) Make(len int) ([]Point, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]Point)(unsafe.Pointer(sliceHdr)), nil
}

// MakeWithCapacity is an analog to make([]Point, len, cap),
// but it allocates this slice in the underlying arena.
// Resulting []Point can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
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

// Append is an analog to append([]Point, ...Point),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

func (s *internalPointSliceView) growIfNecessary(slice []Point,
	requiredLen int) (*internalPointSliceHeader, error) {
	var tVar Point
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*internalPointSliceHeader)(unsafe.Pointer(&slice))
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

func (s *internalPointSliceView) makeGoSlice(len int) (*internalPointSliceHeader, error) {
	valueSlice, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceRef := s.state.alloc.ToRef(valueSlice.data)
	sliceHdr := internalPointSliceHeader{
		Data: uintptr(sliceRef),
		Len:  len,
		Cap:  len,
	}
	return &sliceHdr, nil
}

type internalPointBufferView struct {
	state internalPointState
}

// Make is an analog to make([]Point, len),
// but it allocates this slice in the underlying arena,
// and returns PointBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// PointBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]Point, len, cap)
// and append([]Point, ...Point) analogs
// please refer to other methods of this subview.
func (s *internalPointBufferView) Make(len int) (PointBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return PointBuffer{}, allocErr
	}
	return sliceHdr, nil
}

// Make is an analog to make([]Point, len, cap),
// but it allocates this slice in the underlying arena,
// and returns PointBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// PointBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]Point, len)
// and append([]Point, ...Point) analogs
// please refer to other methods of this subview.
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

// Append is an analog to append([]Point, ...Point),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

// ToRef converts PointBuffer to []Point but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internalPointBufferView) ToRef(slice PointBuffer) []Point {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := internalPointSliceHeader{
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

	return s.grow(slice, requiredLen)
}

func (s *internalPointBufferView) grow(
	slice PointBuffer,
	requiredLen int,
) (PointBuffer, error) {
	var tVar Point
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
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

type internalPointSliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
