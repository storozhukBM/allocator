package arena_test

import (
	"fmt"
	"github.com/storozhukBM/allocator/lib/arena"
	"runtime"
	"testing"
	"unsafe"
)

type allocator interface {
	Alloc(size, padding uintptr) (arena.Ptr, error)
	ToRef(ptr arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
}

const requiredBytesForTest = 48

func basicArenaCheckingStand(t *testing.T, target allocator) {
	{
		_, zeroAllocErr := target.Alloc(0, 1)
		failOnError(t, zeroAllocErr)
		assert(target.Metrics().UsedBytes == 0, "expect used bytes should be 0. instead: %v", target.Metrics())
	}
	{
		_, oneAllocErr := target.Alloc(1, 1)
		failOnError(t, oneAllocErr)
		assert(target.Metrics().UsedBytes == 1, "expect used bytes should be 1. instead: %v", target.Metrics())
	}
	{
		_, threeAllocErr := target.Alloc(3, 1)
		failOnError(t, threeAllocErr)
		assert(target.Metrics().UsedBytes == 4, "expect used bytes should be 4. instead: %v", target.Metrics())
	}
	{
		_, movePaddingAllocErr := target.Alloc(1, 1)
		failOnError(t, movePaddingAllocErr)
		assert(target.Metrics().UsedBytes == 5, "expect used bytes should be 5. instead: %v", target.Metrics())
	}
	{
		_, testAlignmentErr := target.Alloc(4, 4)
		failOnError(t, testAlignmentErr)
		assert(target.Metrics().UsedBytes == 12, "expect used bytes should be 12. instead: %v", target.Metrics())
	}
	{
		boss := &person{name: "Richard Bahman", age: 44}

		personPtr, allocErr := target.Alloc(unsafe.Sizeof(person{}), unsafe.Alignof(person{}))
		failOnError(t, allocErr)
		ref := target.ToRef(personPtr)
		rawPtr := uintptr(ref)
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			p.name = "John Smith"
			p.age = 21
			p.manager = boss
		}
		runtime.GC()
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			assert(p.name == "John Smith", "unexpected person state: %+v", p)
			assert(p.age == 21, "unexpected person state: %+v", p)
			assert(p.manager.name == "Richard Bahman", "unexpected person state: %+v", p)
			assert(p.manager.age == 44, "unexpected person state: %+v", p)
		}
	}
}

func assert(condition bool, msg string, args ...interface{}) {
	if !condition {
		fmt.Printf(msg, args...)
		fmt.Printf("\n")
		panic("assertion failed")
	}
}

func failOnError(t *testing.T, e error) {
	if e != nil {
		t.Error(e)
		t.FailNow()
	}
}

type person struct {
	name    string
	age     uint
	manager *person
}
