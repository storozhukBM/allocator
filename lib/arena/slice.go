package arena

import (
	"reflect"
	"unsafe"
)

type bytesAllocator interface {
	Alloc(size uintptr, alignment uintptr) (Ptr, error)
	ToRef(p Ptr) unsafe.Pointer
}

func MakeBytes(alloc bytesAllocator, len uintptr) (Bytes, error) {
	slicePtr, allocErr := alloc.Alloc(uintptr(len), 8)
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	return Bytes{
		data: slicePtr,
		len:  len,
		cap:  len,
	}, nil
}

func MakeBytesWithCapacity(alloc bytesAllocator, length uintptr, capacity uintptr) (Bytes, error) {
	if capacity < length {
		return Bytes{}, AllocationInvalidArgumentError
	}
	bytes, allocErr := MakeBytes(alloc, capacity)
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	bytes.len = length
	return bytes, nil
}

func Append(alloc bytesAllocator, bytesSlice Bytes, bytesToAppend ...byte) (Bytes, error) {
	target := bytesSlice
	availableSize := int(target.cap - target.len)

	if availableSize < len(bytesToAppend) {
		newSize := max(2*(int(target.cap)+len(bytesToAppend)), 2*int(target.cap))
		newTarget, allocErr := MakeBytes(alloc, uintptr(newSize))
		if allocErr != nil {
			return Bytes{}, allocErr
		}
		copy(BytesToRef(alloc, newTarget), BytesToRef(alloc, target))
		target = newTarget
	}

	target.len = bytesSlice.len + uintptr(len(bytesToAppend))
	copy(BytesToRef(alloc, target)[bytesSlice.len:], bytesToAppend)
	return target, nil
}

func Embed(alloc bytesAllocator, src []byte) (Bytes, error) {
	result, allocErr := MakeBytes(alloc, uintptr(len(src)))
	if allocErr != nil {
		return Bytes{}, allocErr
	}
	resultAsSlice := BytesToRef(alloc, result)
	copy(resultAsSlice, src)
	return result, nil
}

func EmbedAsBytes(alloc bytesAllocator, src []byte) ([]byte, error) {
	bytes, allocErr := Embed(alloc, src)
	if allocErr != nil {
		return nil, allocErr
	}
	return BytesToRef(alloc, bytes), nil
}

func EmbedAsString(alloc bytesAllocator, src []byte) (string, error) {
	bytes, allocErr := Embed(alloc, src)
	if allocErr != nil {
		return "", allocErr
	}
	return BytesToStringRef(alloc, bytes), nil
}

func BytesToRef(alloc bytesAllocator, bytes Bytes) []byte {
	sliceHdr := bytesToSliceHeader(alloc, bytes)
	return *(*[]byte)(unsafe.Pointer(&sliceHdr))
}

func BytesToStringRef(alloc bytesAllocator, bytes Bytes) string {
	sliceHdr := bytesToSliceHeader(alloc, bytes)
	return *(*string)(unsafe.Pointer(&sliceHdr))
}

func CopyBytesToHeap(alloc bytesAllocator, bytes Bytes) []byte {
	sliceFromArena := BytesToRef(alloc, bytes)
	copyOnHeap := make([]byte, bytes.len)
	copy(copyOnHeap, sliceFromArena)
	return copyOnHeap
}

func CopyBytesToStringOnHeap(alloc bytesAllocator, bytes Bytes) string {
	sliceFromArena := BytesToRef(alloc, bytes)
	copyOnHeap := make([]byte, bytes.len)
	copy(copyOnHeap, sliceFromArena)
	return *(*string)(unsafe.Pointer(&copyOnHeap))
}

func bytesToSliceHeader(alloc bytesAllocator, bytes Bytes) reflect.SliceHeader {
	sliceBufferRef := alloc.ToRef(bytes.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(sliceBufferRef),
		Len:  int(bytes.len),
		Cap:  int(bytes.cap),
	}
	return sliceHdr
}
