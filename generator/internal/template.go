package generator

const embeddedTemplate = `
package {{.PkgName}}
{{$ttName := .TargetTypeName}}

import (
	"fmt"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internal{{.TypeNameWithUpperFirstLetter}}Allocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

// {{$ttName}}Ptr, which basically represents an offset of the allocated value {{$ttName}}
// inside one of the arenas.
//
// {{$ttName}}Ptr is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation methods please refer to {{$ttName}}View.Ptr methods.
//
// {{$ttName}}Ptr can be converted to *{{$ttName}} or dereferenced by using 
// {{$ttName}}View.Ptr methods, but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
//
// For detailed documentation please refer to
// internal{{.TypeNameWithUpperFirstLetter}}PtrView.DeRef
// and internal{{.TypeNameWithUpperFirstLetter}}PtrView.ToRef
type {{$ttName}}Ptr struct {
	ptr arena.Ptr
}

// {{$ttName}}Buffer is an analog to []{{$ttName}}, 
// but it represents a slice allocated inside one of the arenas.
// {{$ttName}}Buffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For allocation and append methods please refer to {{$ttName}}View.Buffer methods.
//
// {{$ttName}}Buffer can be converted to []{{$ttName}}
// by using {{$ttName}}View.Buffer.ToRef method,
// but we'd suggest to do it right before use to eliminate its visibility scope
// and potentially prevent it's escaping to the heap.
type {{$ttName}}Buffer struct {
	data arena.Ptr
	len  int
	cap  int
}

// Len is direct analog to len([]{{$ttName}})
func (s {{$ttName}}Buffer) Len() int {
	return s.len
}

// Cap is direct analog to cap([]{{$ttName}})
func (s {{$ttName}}Buffer) Cap() int {
	return s.cap
}

// SubSlice is an analog to []{{$ttName}}[low:high]
// Returns sub-slice of the {{$ttName}}Buffer and panics in case of bounds out of range.
func (s {{$ttName}}Buffer) SubSlice(low int, high int) {{$ttName}}Buffer {
	inBounds := low >= 0 && low <= high && high <= s.cap
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: slice bounds out of range [%d:%d] with capacity %d",
			low, high, s.cap,
		))
	}
	var tVar {{$ttName}}
	tSize := unsafe.Sizeof(tVar)
	type internalPtr struct{
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
	return {{$ttName}}Buffer{
		data: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
		len: high - low,
		cap: s.cap - low,
	}
}

// Get is an analog to []{{$ttName}}[idx]
// Returns {{$ttName}}Ptr and panics in case of idx out of range.
func (s {{$ttName}}Buffer) Get(idx int) {{$ttName}}Ptr {
	inBounds := idx >= 0 && idx < int(s.len)
	if !inBounds {
		panic(fmt.Errorf(
			"runtime error: index out of range [%d] with length %d",
			idx, s.len,
		))
	}
	var tVar {{$ttName}}
	tSize := unsafe.Sizeof(tVar)
	type internalPtr struct{
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
	return {{$ttName}}Ptr{
		ptr: *(*arena.Ptr)(unsafe.Pointer(&newPtr)),
	}
}

// {{$ttName}}View is an allocation view that can be constructed on top of the target allocator
// and then used to allocate {{$ttName}}, its slices and buffers inside target allocator.
//
// {{$ttName}}View contains 3 subviews in form on fields.
//
// Ptr - subview to allocate and operate with {{$ttName}}Ptr structures.
// Slice - to allocate []{{$ttName}} inside target allocator.
// Buffer - to allocate and operate with {{$ttName}}Buffer inside target allocator.
type {{$ttName}}View struct {
	Ptr    internal{{.TypeNameWithUpperFirstLetter}}PtrView
	Slice  internal{{.TypeNameWithUpperFirstLetter}}SliceView
	Buffer internal{{.TypeNameWithUpperFirstLetter}}BufferView
}

{{ if .Exported}}
// New{{.TypeNameWithUpperFirstLetter}}View creates allocation view on top of target allocator
func New{{.TypeNameWithUpperFirstLetter}}View(alloc internal{{.TypeNameWithUpperFirstLetter}}Allocator) *{{$ttName}}View {
{{- else}}
// new{{.TypeNameWithUpperFirstLetter}}View creates allocation view on top of target allocator
func new{{.TypeNameWithUpperFirstLetter}}View(alloc internal{{.TypeNameWithUpperFirstLetter}}Allocator) *{{$ttName}}View {
{{- end}}
	if alloc == nil {
		state := internal{{.TypeNameWithUpperFirstLetter}}State{alloc: &arena.GenericAllocator{}}
		return &{{$ttName}}View{
			Ptr:    internal{{.TypeNameWithUpperFirstLetter}}PtrView{state: state},
			Slice:  internal{{.TypeNameWithUpperFirstLetter}}SliceView{state: state},
			Buffer: internal{{.TypeNameWithUpperFirstLetter}}BufferView{state: state},
		}
	}
	state := internal{{.TypeNameWithUpperFirstLetter}}State{alloc: alloc}
	return &{{$ttName}}View{
		Ptr:    internal{{.TypeNameWithUpperFirstLetter}}PtrView{state: state},
		Slice:  internal{{.TypeNameWithUpperFirstLetter}}SliceView{state: state},
		Buffer: internal{{.TypeNameWithUpperFirstLetter}}BufferView{state: state},
	}
}

type internal{{.TypeNameWithUpperFirstLetter}}PtrView struct {
	state internal{{.TypeNameWithUpperFirstLetter}}State
}

// New allocates {{$ttName}} inside target allocator and returns {{$ttName}}Ptr to it.
// {{$ttName}}Ptr can be converted to *{{$ttName}} or dereferenced by using other methods of this view.
func (s *internal{{.TypeNameWithUpperFirstLetter}}PtrView) New() ({{$ttName}}Ptr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return {{$ttName}}Ptr{}, allocErr
	}
	ptr := {{$ttName}}Ptr{ptr: slice.data}
	return ptr, nil
}

// Embed copies passed value inside target allocator, and returns {{$ttName}}Ptr to it.
// {{$ttName}}Ptr can be converted to *{{$ttName}} or dereferenced by using other methods of this view.
func (s *internal{{.TypeNameWithUpperFirstLetter}}PtrView) Embed(value {{$ttName}}) ({{$ttName}}Ptr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return {{$ttName}}Ptr{}, allocErr
	}
	valueInPool := (*{{$ttName}})(s.state.alloc.ToRef(slice.data))
	*valueInPool = value
	ptr := {{$ttName}}Ptr{ptr: slice.data}
	return ptr, nil
}

// DeRef returns value of {{$ttName}} referenced by {{$ttName}}Ptr.
func (s *internal{{.TypeNameWithUpperFirstLetter}}PtrView) DeRef(allocPtr {{$ttName}}Ptr) ({{$ttName}}) {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*{{$ttName}})(ref)
	return *valuePtr
}

// ToRef converts {{$ttName}}Ptr to *{{$ttName}} but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internal{{.TypeNameWithUpperFirstLetter}}PtrView) ToRef(allocPtr {{$ttName}}Ptr) (*{{$ttName}}) {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*{{$ttName}})(ref)
	return valuePtr
}

type internal{{.TypeNameWithUpperFirstLetter}}SliceView struct {
	state internal{{.TypeNameWithUpperFirstLetter}}State
}

// Make is an analog to make([]{{$ttName}}, len), but it allocates this slice in the underlying arena.
// Resulting []{{$ttName}} can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function, 
// or if you want all other contiguous allocations to happen in the same target allocator, 
// please refer to the Append method. 
// For make([]{{$ttName}}, len, cap) method please refer to the MakeWithCapacity.
func (s *internal{{.TypeNameWithUpperFirstLetter}}SliceView) Make(len int) ([]{{$ttName}}, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]{{$ttName}})(unsafe.Pointer(sliceHdr)), nil
}

// MakeWithCapacity is an analog to make([]{{$ttName}}, len, cap),
// but it allocates this slice in the underlying arena.
// Resulting []{{$ttName}} can be used in the same way as any Go slice can be used.
//
// You can append to it using Go builtin function, 
// or if you want all other contiguous allocations to happen in the same target allocator, 
// please refer to the Append method.
func (s *internal{{.TypeNameWithUpperFirstLetter}}SliceView) MakeWithCapacity(length int, capacity int) ([]{{$ttName}}, error) {
	if capacity < length {
		return nil, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeGoSlice(capacity)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceHdr.Len = length
	return *(*[]{{$ttName}})(unsafe.Pointer(sliceHdr)), nil
}

// Append is an analog to append([]{{$ttName}}, ...{{$ttName}}),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
func (s *internal{{.TypeNameWithUpperFirstLetter}}SliceView) Append(slice []{{$ttName}}, elemsToAppend ...{{$ttName}}) ([]{{$ttName}}, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]{{$ttName}})(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *internal{{.TypeNameWithUpperFirstLetter}}SliceView) growIfNecessary(slice []{{$ttName}}, 
requiredLen int) (*internal{{.TypeNameWithUpperFirstLetter}}SliceHeader, error) {
	var tVar {{$ttName}}
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*internal{{.TypeNameWithUpperFirstLetter}}SliceHeader)(unsafe.Pointer(&slice))
	availableSizeInBytes := int(sliceHdr.Cap - sliceHdr.Len) * int(tSize)
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
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceHdr.Data+(uintptr(sliceHdr.Cap) * tSize)
		if nextAllocationIsRightAfterTargetSlice && s.state.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.state.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return nil, enhancingErr
			}
			sliceHdr.Cap += requiredLen
			return sliceHdr, nil
		}
	}
	newDstSlice, allocErr := s.makeGoSlice(2*(int(sliceHdr.Cap)+requiredLen))
	if allocErr != nil {
		return nil, allocErr
	}
	dst := *(*[]{{$ttName}})(unsafe.Pointer(newDstSlice))
	copy(dst, slice)
	return newDstSlice, nil
}

func (s *internal{{.TypeNameWithUpperFirstLetter}}SliceView) makeGoSlice(len int) (*internal{{.TypeNameWithUpperFirstLetter}}SliceHeader, error) {
	valueSlice, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceRef := s.state.alloc.ToRef(valueSlice.data)
	sliceHdr := internal{{.TypeNameWithUpperFirstLetter}}SliceHeader{
		Data: uintptr(sliceRef),
		Len:  len,
		Cap:  len,
	}
	return &sliceHdr, nil
}

type internal{{.TypeNameWithUpperFirstLetter}}BufferView struct {
	state internal{{.TypeNameWithUpperFirstLetter}}State
}

// Make is an analog to make([]{{$ttName}}, len), 
// but it allocates this slice in the underlying arena,
// and returns {{$ttName}}Buffer which is a simple representation
// of a slice allocated inside one of the arenas.
// 
// {{$ttName}}Buffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]{{$ttName}}, len, cap) 
// and append([]{{$ttName}}, ...{{$ttName}}) analogs
// please refer to other methods of this subview.
func (s *internal{{.TypeNameWithUpperFirstLetter}}BufferView) Make(len int) ({{$ttName}}Buffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return {{$ttName}}Buffer{}, allocErr
	}
	return sliceHdr, nil
}

// Make is an analog to make([]{{$ttName}}, len, cap), 
// but it allocates this slice in the underlying arena,
// and returns {{$ttName}}Buffer which is a simple representation
// of a slice allocated inside one of the arenas.
// 
// {{$ttName}}Buffer is a simple struct that should be passed by value and
// is not considered by Go runtime as a legit pointer type.
// So the GC can skip it during the concurrent mark phase.
//
// For make([]{{$ttName}}, len) 
// and append([]{{$ttName}}, ...{{$ttName}}) analogs
// please refer to other methods of this subview.
func (s *internal{{.TypeNameWithUpperFirstLetter}}BufferView) MakeWithCapacity(length int, 
capacity int) ({{$ttName}}Buffer, error) {
	if capacity < length {
		return {{$ttName}}Buffer{}, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.state.makeSlice(capacity)
	if allocErr != nil {
		return {{$ttName}}Buffer{}, allocErr
	}
	sliceHdr.len = length
	return sliceHdr, nil
}

// Append is an analog to append([]{{$ttName}}, ...{{$ttName}}),
// but in case if allocations necessary to proceed with append it allocates this new in the underlying arena.
func (s *internal{{.TypeNameWithUpperFirstLetter}}BufferView) Append(
		slice {{$ttName}}Buffer,
		elemsToAppend ...{{$ttName}},
) ({{$ttName}}Buffer, error) {

	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return {{$ttName}}Buffer{}, allocErr
	}
	target.len = slice.len + len(elemsToAppend)
	result := s.ToRef(target)
	copy(result[slice.len:], elemsToAppend)
	return target, nil
}

// ToRef converts {{$ttName}}Buffer to []{{$ttName}} but we'd suggest to do it right before use
// to eliminate its visibility scope and potentially prevent it's escaping to the heap.
func (s *internal{{.TypeNameWithUpperFirstLetter}}BufferView) ToRef(slice {{$ttName}}Buffer) []{{$ttName}} {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := internal{{.TypeNameWithUpperFirstLetter}}SliceHeader{
		Data: uintptr(dataRef),
		Len:  slice.len,
		Cap:  slice.cap,
	}
	return *(*[]{{$ttName}})(unsafe.Pointer(&sliceHdr))
}

func (s *internal{{.TypeNameWithUpperFirstLetter}}BufferView) growIfNecessary(
		slice {{$ttName}}Buffer,
		requiredLen int,
) ({{$ttName}}Buffer, error) {
	var tVar {{$ttName}}
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
			return {{$ttName}}Buffer{}, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		sliceDataAddr := uintptr(s.state.alloc.ToRef(slice.data))
		nextPtrAddr := uintptr(s.state.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceDataAddr+(uintptr(slice.cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.state.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.state.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return {{$ttName}}Buffer{}, enhancingErr
			}
			slice.cap += requiredLen
			return slice, nil
		}
	}
	newDstSlice, allocErr := s.state.makeSlice(2 * (int(slice.cap) + requiredLen))
	if allocErr != nil {
		return {{$ttName}}Buffer{}, allocErr
	}
	if slice.len > 0 {
		dst := s.ToRef(newDstSlice)
		prev := s.ToRef(slice)
		copy(dst, prev)
	}
	return newDstSlice, nil
}

type internal{{.TypeNameWithUpperFirstLetter}}State struct {
	alloc            internal{{.TypeNameWithUpperFirstLetter}}Allocator
	lastAllocatedPtr arena.Ptr
}

func (s *internal{{.TypeNameWithUpperFirstLetter}}State) makeSlice(len int) ({{$ttName}}Buffer, error) {
	var tVar {{$ttName}}
	tSize := unsafe.Sizeof(tVar)
	tAlignment := unsafe.Alignof(tVar)
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len)*tSize, tAlignment)
	if allocErr != nil {
		return {{$ttName}}Buffer{}, allocErr
	}
	s.lastAllocatedPtr = slicePtr
	sliceHdr := {{$ttName}}Buffer{
		data: slicePtr,
		len:  len,
		cap:  len,
	}
	return sliceHdr, nil
}

type internal{{.TypeNameWithUpperFirstLetter}}SliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
`
