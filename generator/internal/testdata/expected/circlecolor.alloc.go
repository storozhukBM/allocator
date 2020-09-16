package etalon

import (
	"fmt"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internalCircleColorAllocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

// CircleColorPtr, which basically represents an offset of the allocated value CircleColor
// inside one of the arenas.
//
// CircleColorPtr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation methods please refer to CircleColorView.Ptr methods.
//
// CircleColorPtr can be converted to *CircleColor or dereferenced by using
// CircleColorView.Ptr methods, but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
//
// For detailed documentation please refer to
// internalCircleColorPtrView.DeRef
// and internalCircleColorPtrView.ToRef
type CircleColorPtr struct {
	ptr arena.Ptr
}

// CircleColorBuffer is an analog to []CircleColor,
// but it represents a slice allocated inside one of the arenas.
// CircleColorBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation and append methods please refer to CircleColorView.Buffer methods.
//
// CircleColorBuffer can be converted to []CircleColor
// by using CircleColorView.Buffer.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
type CircleColorBuffer struct {
	data arena.Ptr
	len  int
	cap  int
}

// Len is direct analog to len([]CircleColor)
func (s CircleColorBuffer) Len() int {
	return s.len
}

// Cap is direct analog to cap([]CircleColor)
func (s CircleColorBuffer) Cap() int {
	return s.cap
}

// SubSlice is an analog to []CircleColor[low:high]
// Returns sub-slice of the CircleColorBuffer and panics in case of bounds out of range.
func (s CircleColorBuffer) SubSlice(low int, high int) CircleColorBuffer {
	inBounds := low >= 0 && low <= high && high <= s.cap
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: slice bounds out of range [%d:%d] with capacity %d",
			low, high, s.cap,
		))
	}
	var tVar CircleColor
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
	return CircleColorBuffer{
		data: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
		len:  high - low,
		cap:  s.cap - low,
	}
}

// Get is an analog to []CircleColor[idx]
// Returns CircleColorPtr and panics in case of idx out of range.
func (s CircleColorBuffer) Get(idx int) CircleColorPtr {
	inBounds := idx >= 0 && idx < int(s.len)
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: index out of range [%d] with length %d",
			idx, s.len,
		))
	}
	var tVar CircleColor
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
	return CircleColorPtr{
		ptr: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
	}
}

// CircleColorView is an allocation view that can be constructed on top of the target allocator
// and then used to allocate CircleColor, its slices and buffers inside target allocator.
//
// CircleColorView contains 3 subviews in form on fields.
//
// Ptr - subview to allocate and operate with CircleColorPtr structures.
// Slice - to allocate []CircleColor inside target allocator.
// Buffer - to allocate and operate with CircleColorBuffer inside target allocator.
type CircleColorView struct {
	Ptr    internalCircleColorPtrView
	Slice  internalCircleColorSliceView
	Buffer internalCircleColorBufferView
}

// NewCircleColorView creates allocation view on top of target allocator
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

// New allocates CircleColor inside target allocator and returns CircleColorPtr to it.
// CircleColorPtr can be converted to *CircleColor or dereferenced by using other methods of this view.
func (s *internalCircleColorPtrView) New() (CircleColorPtr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return CircleColorPtr{}, allocErr
	}
	ptr := CircleColorPtr{ptr: slice.data}
	return ptr, nil
}

// Embed copies passed value inside target allocator, and returns CircleColorPtr to it.
// CircleColorPtr can be converted to *CircleColor or dereferenced by using other methods of this view.
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

// DeRef returns value of CircleColor referenced by CircleColorPtr.
func (s *internalCircleColorPtrView) DeRef(allocPtr CircleColorPtr) CircleColor {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*CircleColor)(ref)
	return *valuePtr
}

// ToRef converts CircleColorPtr to *CircleColor but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internalCircleColorPtrView) ToRef(allocPtr CircleColorPtr) *CircleColor {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*CircleColor)(ref)
	return valuePtr
}

type internalCircleColorSliceView struct {
	state internalCircleColorState
}

// Make is an analog to make([]CircleColor, len), but it allocates this slice in the underlying arena.
// Resulting []CircleColor can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
// For make([]CircleColor, len, cap) method please refer to the MakeWithCapacity.
func (s *internalCircleColorSliceView) Make(len int) ([]CircleColor, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]CircleColor)(unsafe.Pointer(sliceHdr)), nil
}

// MakeWithCapacity is an analog to make([]CircleColor, len, cap),
// but it allocates this slice in the underlying arena.
// Resulting []CircleColor can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function,
// or if you want all other contiguous allocations to happen in the same target allocator,
// please refer to the Append method.
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

// Append is an analog to append([]CircleColor, ...CircleColor),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

func (s *internalCircleColorSliceView) growIfNecessary(slice []CircleColor,
	requiredLen int) (*internalCircleColorSliceHeader, error) {
	var tVar CircleColor
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*internalCircleColorSliceHeader)(unsafe.Pointer(&slice))
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

func (s *internalCircleColorSliceView) makeGoSlice(len int) (*internalCircleColorSliceHeader, error) {
	valueSlice, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceRef := s.state.alloc.ToRef(valueSlice.data)
	sliceHdr := internalCircleColorSliceHeader{
		Data: uintptr(sliceRef),
		Len:  len,
		Cap:  len,
	}
	return &sliceHdr, nil
}

type internalCircleColorBufferView struct {
	state internalCircleColorState
}

// Make is an analog to make([]CircleColor, len),
// but it allocates this slice in the underlying arena,
// and returns CircleColorBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// CircleColorBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]CircleColor, len, cap)
// and append([]CircleColor, ...CircleColor) analogs
// please refer to other methods of this subview.
func (s *internalCircleColorBufferView) Make(len int) (CircleColorBuffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return CircleColorBuffer{}, allocErr
	}
	return sliceHdr, nil
}

// Make is an analog to make([]CircleColor, len, cap),
// but it allocates this slice in the underlying arena,
// and returns CircleColorBuffer which is a simple representation
// of a slice allocated inside one of the arenas.
//
// CircleColorBuffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]CircleColor, len)
// and append([]CircleColor, ...CircleColor) analogs
// please refer to other methods of this subview.
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

// Append is an analog to append([]CircleColor, ...CircleColor),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
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

// ToRef converts CircleColorBuffer to []CircleColor but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internalCircleColorBufferView) ToRef(slice CircleColorBuffer) []CircleColor {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := internalCircleColorSliceHeader{
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

	return s.grow(slice, requiredLen)
}

func (s *internalCircleColorBufferView) grow(
	slice CircleColorBuffer,
	requiredLen int,
) (CircleColorBuffer, error) {
	var tVar CircleColor
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
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
	if slice.len > 0 {
		dst := s.ToRef(newDstSlice)
		prev := s.ToRef(slice)
		copy(dst, prev)
	}
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

type internalCircleColorSliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
