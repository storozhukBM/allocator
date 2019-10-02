package generator

import (
	"text/template"
)

var embeddedTemplate = template.Must(template.New("embedded").Parse(`
package {{.PkgName}}
{{$ttName := .TargetTypeName}}

import (
	"reflect"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type internal{{.TypeNameWithUpperFirstLetter}}Allocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type {{$ttName}}Ptr struct {
	ptr arena.Ptr
}

type {{$ttName}}Buffer struct {
	data arena.Ptr
	len  int
	cap  int
}

func (s {{$ttName}}Buffer) Len() int {
	return s.len
}

func (s {{$ttName}}Buffer) Cap() int {
	return s.cap
}

type {{$ttName}}View struct {
	Ptr    internal{{.TypeNameWithUpperFirstLetter}}PtrView
	Slice  internal{{.TypeNameWithUpperFirstLetter}}SliceView
	Buffer internal{{.TypeNameWithUpperFirstLetter}}BufferView
}

{{ if .Exported}}
func New{{.TypeNameWithUpperFirstLetter}}View(alloc internal{{.TypeNameWithUpperFirstLetter}}Allocator) *{{$ttName}}View {
{{- else}}
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

func (s *internal{{.TypeNameWithUpperFirstLetter}}PtrView) New() ({{$ttName}}Ptr, error) {
	slice, allocErr := s.state.makeSlice(1)
	if allocErr != nil {
		return {{$ttName}}Ptr{}, allocErr
	}
	ptr := {{$ttName}}Ptr{ptr: slice.data}
	return ptr, nil
}

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

func (s *internal{{.TypeNameWithUpperFirstLetter}}PtrView) DeRef(allocPtr {{$ttName}}Ptr) ({{$ttName}}) {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*{{$ttName}})(ref)
	return *valuePtr
}

func (s *internal{{.TypeNameWithUpperFirstLetter}}PtrView) ToRef(allocPtr {{$ttName}}Ptr) (*{{$ttName}}) {
	ref := s.state.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*{{$ttName}})(ref)
	return valuePtr
}

type internal{{.TypeNameWithUpperFirstLetter}}SliceView struct {
	state internal{{.TypeNameWithUpperFirstLetter}}State
}

func (s *internal{{.TypeNameWithUpperFirstLetter}}SliceView) Make(len int) ([]{{$ttName}}, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]{{$ttName}})(unsafe.Pointer(sliceHdr)), nil
}

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

func (s *internal{{.TypeNameWithUpperFirstLetter}}SliceView) growIfNecessary(slice []{{$ttName}}, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar {{$ttName}}
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
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

func (s *internal{{.TypeNameWithUpperFirstLetter}}SliceView) makeGoSlice(len int) (*reflect.SliceHeader, error) {
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

type internal{{.TypeNameWithUpperFirstLetter}}BufferView struct {
	state internal{{.TypeNameWithUpperFirstLetter}}State
}

func (s *internal{{.TypeNameWithUpperFirstLetter}}BufferView) Make(len int) ({{$ttName}}Buffer, error) {
	sliceHdr, allocErr := s.state.makeSlice(len)
	if allocErr != nil {
		return {{$ttName}}Buffer{}, allocErr
	}
	return sliceHdr, nil
}

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

func (s *internal{{.TypeNameWithUpperFirstLetter}}BufferView) ToRef(slice {{$ttName}}Buffer) []{{$ttName}} {
	dataRef := s.state.alloc.ToRef(slice.data)
	sliceHdr := reflect.SliceHeader{
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
	dst := s.ToRef(newDstSlice)
	prev := s.ToRef(slice)
	copy(dst, prev)
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
`))
