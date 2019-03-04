package allocator

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"
)

func simpleConsecutivePersonsCase() allocationPath {
	return allocationPath{
		name: "simple consecutive persons",
		allocations: []allocation{{
			count: 15,
			target: allocationType{
				typeName: "person",
				typeVal:  reflect.TypeOf(person{}),
			},
			result: allocationResult{
				countOfAllocations: 15,
				usedBytes:          15 * personSize,
				overallCapacity:    defaultFirstBucketSize,
				countOfBuckets:     1,
			},
		}},
	}
}

func personAndDeal() allocationPath {
	return allocationPath{
		name: "person and deal",
		allocations: []allocation{
			{
				count: 1,
				target: allocationType{
					typeName: "person",
					typeVal:  reflect.TypeOf(person{}),
				},
				result: allocationResult{
					countOfAllocations: 1,
					usedBytes:          personSize,
					overallCapacity:    defaultFirstBucketSize,
					countOfBuckets:     1,
				},
			},
			{
				count: 1,
				target: allocationType{
					typeName: "deal",
					typeVal:  reflect.TypeOf(deal{}),
				},
				result: allocationResult{
					countOfAllocations: 2,
					usedBytes:          personSize + dealSize,
					overallCapacity:    defaultFirstBucketSize,
					countOfBuckets:     1,
				},
			},
		},
	}
}

func personAndBooleansAndPerson() allocationPath {
	return allocationPath{
		name: "person few booleans and one more person",
		allocations: []allocation{
			{
				count: 1,
				target: allocationType{
					typeName: "person",
					typeVal:  reflect.TypeOf(person{}),
				},
				result: allocationResult{
					countOfAllocations: 1,
					usedBytes:          personSize,
					overallCapacity:    defaultFirstBucketSize,
					countOfBuckets:     1,
				},
			},
			{
				count: 3,
				target: allocationType{
					typeName: "bool",
					typeVal:  reflect.TypeOf(true),
				},
				result: allocationResult{
					countOfAllocations: 4,
					usedBytes:          personSize + 3,
					overallCapacity:    defaultFirstBucketSize,
					countOfBuckets:     1,
				},
			},
			{
				count: 1,
				target: allocationType{
					typeName: "person",
					typeVal:  reflect.TypeOf(person{}),
				},
				result: allocationResult{
					countOfAllocations: 5,
					usedBytes:          personSize + 3 + personSize,
					overallCapacity:    defaultFirstBucketSize,
					countOfBuckets:     1,
				},
			},
		},
	}
}

func TestAllocationPath(t *testing.T) {
	cases := []allocationPath{
		simpleConsecutivePersonsCase(),
		personAndDeal(),
		personAndBooleansAndPerson(),
	}
	for _, path := range cases {
		caseName := strings.Replace(path.name, " ", "_", -1)
		t.Run(caseName, func(t *testing.T) {
			ar := &Arena{}
			checkArenaState(ar, allocationResult{countOfBuckets: 1})
			for _, alloc := range path.allocations {
				for i := 0; i < alloc.count; i++ {
					ptr := ar.Alloc(alloc.target.typeVal.Size())
					assert(ptr != APtr{}, "ptr is not nil")
				}
				checkArenaState(ar, alloc.result)
			}
		})
	}
}

func TestAllocationInGeneral(t *testing.T) {
	ar := &Arena{}
	checkArenaState(ar, allocationResult{countOfBuckets: 1})
	ar.Alloc(3) // mess with padding
	checkArenaState(ar, allocationResult{
		countOfAllocations: 1,
		usedBytes:          3,
		overallCapacity:    defaultFirstBucketSize,
		countOfBuckets:     1,
	})
	boss := &person{name: "Boss", age: 55}

	fmt.Printf("person size: %+v; person alignment: %+v\n", unsafe.Sizeof(person{}), unsafe.Alignof(person{}))
	fmt.Printf("deal size: %+v; deal alignment: %+v\n", unsafe.Sizeof(deal{}), unsafe.Alignof(deal{}))
	fmt.Printf("string size: %+v; string alignment: %+v\n", unsafe.Sizeof(""), unsafe.Alignof(""))
	fmt.Printf("bool size: %+v; bool alignment: %+v\n", unsafe.Sizeof(true), unsafe.Alignof(true))
	fmt.Printf("APtr size: %+v; APtr alignment: %+v\n", unsafe.Sizeof(APtr{}), unsafe.Alignof(APtr{}))

	cache := make(map[string]*person)
	for i := 1; i < 10001; i++ {
		aPtr := ar.Alloc(unsafe.Sizeof(person{}))
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

	checkArenaState(ar, allocationResult{
		countOfAllocations: 10001,
		usedBytes:          (10000 * personSize) + 3,
		overallCapacity:    estimateSizeOfBuckets(5),
		countOfBuckets:     5,
	})

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

func checkArenaState(arena *Arena, result allocationResult) {
	arenaStr := fmt.Sprintf("arena: %+v\n", arena)
	for _, bucket := range arena.target.buckets {
		arenaStr += fmt.Sprintf("%v\n", bucket)
	}
	assert(arena.CountOfAllocations() == result.countOfAllocations, "unnexpected count of allocations.\n exp: %+v\n act: %+v\n", result, arenaStr)
	assert(arena.UsedBytes() == result.usedBytes, "unnexpected used bytes.\n exp: %+v\n act: %+v\n", result, arenaStr)
	assert(arena.CountOfBuckets() == result.countOfBuckets, "unnexpected count of buckets.\n exp: %+v\n act: %+v\n", result, arenaStr)
	assert(arena.OverallCapacity() == result.overallCapacity, "unnexpected overall capacity.\n exp: %+v\n act: %+v\n", result, arenaStr)
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

// sum of geometric progression, where defaultFirstBucketSize is scale factor and 2 is common ration
func estimateSizeOfBuckets(countOfBuckets int) int {
	twoToThePowerOfBucketsCount := 1 << uint(countOfBuckets)
	return defaultFirstBucketSize * (1 - twoToThePowerOfBucketsCount) / -1
}

var (
	personSize = int(reflect.TypeOf(person{}).Size())
	dealSize   = int(reflect.TypeOf(deal{}).Size())
	stringSize = int(reflect.TypeOf("").Size())
	boolSize   = int(reflect.TypeOf(true).Size())
)

type person struct {
	name    string
	age     uint
	manager *person
}

type deal struct {
	author       person
	participants []person
	summary      string
	mainBody     string
}

type allocationType struct {
	typeName string
	typeVal  reflect.Type
}

type allocation struct {
	count  int
	target allocationType
	result allocationResult
}

type allocationResult struct {
	countOfAllocations int
	usedBytes          int
	overallCapacity    int
	countOfBuckets     int
}

type allocationPath struct {
	name        string
	allocations []allocation
}
