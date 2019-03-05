package allocator

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
	"time"
	"unsafe"
)

func TestZeroArenaToRef(t *testing.T) {
	ar := &Arena{}
	ref := ar.ToRef(APtr{})
	fmt.Printf("%+v\n", ref)
}

func TestArenaMaskGeneration(t *testing.T) {
	first := &Arena{}
	firstPtr, firstAllocErr := first.Alloc(1)
	failOnError(t, firstAllocErr)
	assert(firstPtr.arenaMask != 0, "mask can't be zero")

	second := &Arena{}
	secondPtr, secondAllocErr := second.Alloc(1)
	failOnError(t, secondAllocErr)
	assert(secondPtr.arenaMask != 0, "mask can't be zero")

	assert(firstPtr.arenaMask != secondPtr.arenaMask, "mask of different arenas should be different")
}

func TestWrongArenaToRef(t *testing.T) {
	first := &Arena{}
	_, firstAllocErr := first.Alloc(1)
	failOnError(t, firstAllocErr)

	second := &Arena{}
	secondPtr, secondAllocErr := second.Alloc(1)
	failOnError(t, secondAllocErr)

	panicHappened := false
	func() {
		defer func() {
			err := recover()
			if err != nil {
				panicHappened = true
			}
			errStr := err.(string)
			assert(errStr == "pointer isn't part of this arena", "panic should happen")
		}()
		ref := first.ToRef(secondPtr)
		fmt.Printf("this should never print: %v\n", ref)
	}()
	assert(panicHappened, "wrong arena to ptr panic should happen")
}

func TestAllocationInGeneral(t *testing.T) {
	ar := &Arena{}
	checkArenaState(ar, allocationResult{countOfBuckets: 1}, AOffset{})
	_, paddingAllocErr := ar.Alloc(3) // mess with padding
	failOnError(t, paddingAllocErr)
	checkArenaState(ar,
		allocationResult{countOfAllocations: 1, usedBytes: 3, overallCapacity: defaultFirstBucketSize, countOfBuckets: 1},
		AOffset{offset: 3, bucketIdx: 0, arenaMask: ar.target.arenaMask},
	)
	boss := &person{name: "Boss", age: 55}

	fmt.Printf("person size: %+v; person alignment: %+v\n", unsafe.Sizeof(person{}), unsafe.Alignof(person{}))
	fmt.Printf("deal size: %+v; deal alignment: %+v\n", unsafe.Sizeof(deal{}), unsafe.Alignof(deal{}))
	fmt.Printf("string size: %+v; string alignment: %+v\n", unsafe.Sizeof(""), unsafe.Alignof(""))
	fmt.Printf("bool size: %+v; bool alignment: %+v\n", unsafe.Sizeof(true), unsafe.Alignof(true))
	fmt.Printf("APtr size: %+v; APtr alignment: %+v\n", unsafe.Sizeof(APtr{}), unsafe.Alignof(APtr{}))

	cache := make(map[string]*person)
	for i := 1; i < 10001; i++ {
		aPtr, allocErr := ar.Alloc(unsafe.Sizeof(person{}))
		failOnError(t, allocErr)
		ref := ar.ToRef(aPtr)
		rawPtr := uintptr(ref)
		p := (*person)(unsafe.Pointer(rawPtr))
		p.name = strconv.Itoa(i)
		p.age = uint(i)
		if i%4 == 0 {
			p.manager = cache[strconv.Itoa(i-1)]
		} else {
			p.manager = boss
		}
		cache[p.name] = p
	}

	checkArenaState(ar,
		allocationResult{
			countOfAllocations: 10001,
			usedBytes:          (10000 * personSize) + 3,
			overallCapacity:    estimateSizeOfBuckets(5),
			countOfBuckets:     5,
		},
		AOffset{
			offset:    74272,
			bucketIdx: 4,
			arenaMask: ar.target.arenaMask,
		},
	)

	runtime.GC()
	time.Sleep(2 * time.Second)

	for name, p := range cache {
		expectedAge, parseErr := strconv.Atoi(name)
		failOnError(t, parseErr)
		assert(uint(expectedAge) == p.age, "unexpected age of person: %+v", p)
		if expectedAge%4 == 0 {
			assert(p.manager == cache[strconv.Itoa(expectedAge-1)], "unexpected manager of person: %+v; boss: %+v", p, p.manager)
		} else {
			assert(p.manager == boss, "unexpected manager of person: %+v; boss: %+v", p, p.manager)
		}
	}
}
