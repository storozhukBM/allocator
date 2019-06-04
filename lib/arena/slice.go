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

func BytesToRef(alloc bytesAllocator, bytes Bytes) []byte {
	sliceBufferRef := alloc.ToRef(bytes.data)
	sliceHdr := reflect.SliceHeader{
		Data: uintptr(sliceBufferRef),
		Len:  int(bytes.len),
		Cap:  int(bytes.cap),
	}
	return *(*[]byte)(unsafe.Pointer(&sliceHdr))
}
