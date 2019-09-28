package generator

import (
	"text/template"
)

var embeddedTemplate = template.Must(template.New("embedded").Parse(`
package {{.PkgName}}
{{$ttName := .TargetTypeName}}

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"reflect"
	"unsafe"
)

type internal{{.TypeNameWithUpperFirstLetter}}Allocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

type {{$ttName}}View struct {
	alloc            internal{{.TypeNameWithUpperFirstLetter}}Allocator
	lastAllocatedPtr arena.Ptr
}

type {{$ttName}}Ptr struct {
	ptr arena.Ptr
}

type {{$ttName}}Slice struct {
	data arena.Ptr
	len  int
	cap  int
}

func (s {{$ttName}}Slice) Len() int {
	return s.len
}

func (s {{$ttName}}Slice) Cap() int {
	return s.cap
}

{{- if .Exported}}
func New{{.TypeNameWithUpperFirstLetter}}View(alloc internal{{.TypeNameWithUpperFirstLetter}}Allocator) *{{$ttName}}View {
{{- else}}
func new{{.TypeNameWithUpperFirstLetter}}View(alloc internal{{.TypeNameWithUpperFirstLetter}}Allocator) *{{$ttName}}View {
{{- end}}
	if alloc == nil {
		return &{{$ttName}}View{alloc: &arena.GenericAllocator{}}
	}
	return &{{$ttName}}View{alloc: alloc}
}

func (s *{{$ttName}}View) New() ({{$ttName}}Ptr, error) {
	slice, allocErr := s.makeSlice(1)
	if allocErr != nil {
		return {{$ttName}}Ptr{}, allocErr
	}
	ptr := {{$ttName}}Ptr{ptr: slice.data}
	return ptr, nil
}

func (s *{{$ttName}}View) Embed(value {{$ttName}}) ({{$ttName}}Ptr, error) {
	slice, allocErr := s.makeSlice(1)
	if allocErr != nil {
		return {{$ttName}}Ptr{}, allocErr
	}
	valueInPool := (*{{$ttName}})(s.alloc.ToRef(slice.data))
	*valueInPool = value
	ptr := {{$ttName}}Ptr{ptr: slice.data}
	return ptr, nil
}

func (s *{{$ttName}}View) DeRef(allocPtr {{$ttName}}Ptr) ({{$ttName}}) {
	ref := s.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*{{$ttName}})(ref)
	return *valuePtr
}

func (s *{{$ttName}}View) ToRef(allocPtr {{$ttName}}Ptr) (*{{$ttName}}) {
	ref := s.alloc.ToRef(allocPtr.ptr)
	valuePtr := (*{{$ttName}})(ref)
	return valuePtr
}

func (s *{{$ttName}}View) MakeSlice(len int) ({{$ttName}}Slice, error) {
	sliceHdr, allocErr := s.makeSlice(len)
	if allocErr != nil {
		return {{$ttName}}Slice{}, allocErr
	}
	return sliceHdr, nil
}

func (s *{{$ttName}}View) MakeSliceWithCapacity(length int, capacity int) ({{$ttName}}Slice, error) {
	if capacity < length {
		return {{$ttName}}Slice{}, arena.AllocationInvalidArgumentError
	}
	sliceHdr, allocErr := s.makeSlice(capacity)
	if allocErr != nil {
		return {{$ttName}}Slice{}, allocErr
	}
	sliceHdr.len = length
	return sliceHdr, nil
}

func (s *{{$ttName}}View) AppendSlice(
		slice {{$ttName}}Slice,
		elemsToAppend ...{{$ttName}},
) ({{$ttName}}Slice, error) {

	target, allocErr := s.growSliceIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return {{$ttName}}Slice{}, allocErr
	}
	target.len = slice.len + len(elemsToAppend)
	result := s.SliceToRef(target)
	copy(result[slice.len:], elemsToAppend)
	return target, nil
}

func (s *{{$ttName}}View) SliceToRef(slice {{$ttName}}Slice) []{{$ttName}} {
	dataRef := s.alloc.ToRef(slice.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(dataRef),
		Len:  slice.len,
		Cap:  slice.cap,
	}
	return *(*[]{{$ttName}})(unsafe.Pointer(&sliceHdr))
}

func (s *{{$ttName}}View) Make(len int) ([]{{$ttName}}, error) {
	sliceHdr, allocErr := s.makeGoSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	return *(*[]{{$ttName}})(unsafe.Pointer(sliceHdr)), nil
}

func (s *{{$ttName}}View) MakeWithCapacity(length int, capacity int) ([]{{$ttName}}, error) {
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

func (s *{{$ttName}}View) Append(slice []{{$ttName}}, elemsToAppend ...{{$ttName}}) ([]{{$ttName}}, error) {
	target, allocErr := s.growIfNecessary(slice, len(elemsToAppend))
	if allocErr != nil {
		return nil, allocErr
	}
	target.Len = len(slice) + len(elemsToAppend)
	result := *(*[]{{$ttName}})(unsafe.Pointer(target))
	copy(result[len(slice):], elemsToAppend)
	return result, nil
}

func (s *{{$ttName}}View) growIfNecessary(slice []{{$ttName}}, requiredLen int) (*reflect.SliceHeader, error) {
	var tVar {{$ttName}}
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	sliceHdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	availableSizeInBytes := int(sliceHdr.Cap - sliceHdr.Len) * int(tSize)
	if availableSizeInBytes >= requiredSizeInBytes {
		return sliceHdr, nil
	}

	emptyPtr := arena.Ptr{}
	if s.lastAllocatedPtr != emptyPtr && sliceHdr.Data == uintptr(s.alloc.ToRef(s.lastAllocatedPtr)) {
		nextPtr, probeAllocErr := s.alloc.Alloc(0, 1)
		if probeAllocErr != nil {
			return nil, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		nextPtrAddr := uintptr(s.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceHdr.Data+(uintptr(sliceHdr.Cap) * tSize)
		if nextAllocationIsRightAfterTargetSlice && s.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
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

func (s *{{$ttName}}View) growSliceIfNecessary(
		slice {{$ttName}}Slice,
		requiredLen int,
) ({{$ttName}}Slice, error) {
	var tVar {{$ttName}}
	tSize := unsafe.Sizeof(tVar)
	requiredSizeInBytes := requiredLen * int(tSize)
	availableSizeInBytes := int(slice.cap-slice.len) * int(tSize)
	if availableSizeInBytes >= requiredSizeInBytes {
		return slice, nil
	}

	emptyPtr := arena.Ptr{}
	if s.lastAllocatedPtr != emptyPtr && slice.data == s.lastAllocatedPtr {
		nextPtr, probeAllocErr := s.alloc.Alloc(0, 1)
		if probeAllocErr != nil {
			return {{$ttName}}Slice{}, probeAllocErr
		}
		// current allocation offset is the same as previous
		// we can try to just enhance current buffer
		sliceDataAddr := uintptr(s.alloc.ToRef(slice.data))
		nextPtrAddr := uintptr(s.alloc.ToRef(nextPtr))
		nextAllocationIsRightAfterTargetSlice := nextPtrAddr == sliceDataAddr+(uintptr(slice.cap)*tSize)
		if nextAllocationIsRightAfterTargetSlice && s.alloc.Metrics().AvailableBytes >= requiredSizeInBytes {
			_, enhancingErr := s.alloc.Alloc(uintptr(requiredSizeInBytes), 1)
			if enhancingErr != nil {
				return {{$ttName}}Slice{}, enhancingErr
			}
			slice.cap += requiredLen
			return slice, nil
		}
	}
	newDstSlice, allocErr := s.makeSlice(2 * (int(slice.cap) + requiredLen))
	if allocErr != nil {
		return {{$ttName}}Slice{}, allocErr
	}
	dst := s.SliceToRef(newDstSlice)
	prev := s.SliceToRef(slice)
	copy(dst, prev)
	return newDstSlice, nil
}

func (s *{{$ttName}}View) makeGoSlice(len int) (*reflect.SliceHeader, error) {
	valueSlice, allocErr := s.makeSlice(len)
	if allocErr != nil {
		return nil, allocErr
	}
	sliceRef := s.alloc.ToRef(valueSlice.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(sliceRef),
		Len:  len,
		Cap:  len,
	}
	return &sliceHdr, nil
}

func (s *{{$ttName}}View) makeSlice(len int) ({{$ttName}}Slice, error) {
	var tVar {{$ttName}}
	tSize := unsafe.Sizeof(tVar)
	tAlignment := unsafe.Alignof(tVar)
	slicePtr, allocErr := s.alloc.Alloc(uintptr(len)*tSize, tAlignment)
	if allocErr != nil {
		return {{$ttName}}Slice{}, allocErr
	}
	s.lastAllocatedPtr = slicePtr
	sliceHdr := {{$ttName}}Slice{
		data: slicePtr,
		len:  len,
		cap:  len,
	}
	return sliceHdr, nil
}
`))
