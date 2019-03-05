package allocator

import (
	"fmt"
	"reflect"
	"testing"
)

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

	currentBucketIdx    int
	currentBucketOffset int
}

type allocationPath struct {
	name        string
	allocations []allocation
}

func checkArenaState(arena *Arena, result allocationResult, expectedOffset AOffset) {
	arenaStr := fmt.Sprintf("arena: %+v\n", arena)
	for _, bucket := range arena.target.buckets {
		arenaStr += fmt.Sprintf("%v\n", bucket)
	}
	assert(arena.CountOfAllocations() == result.countOfAllocations, "unnexpected count of allocations.\n exp: %+v\n act: %+v\n", result, arenaStr)
	assert(arena.UsedBytes() == result.usedBytes, "unnexpected used bytes.\n exp: %+v\n act: %+v\n", result, arenaStr)
	assert(arena.CountOfBuckets() == result.countOfBuckets, "unnexpected count of buckets.\n exp: %+v\n act: %+v\n", result, arenaStr)
	assert(arena.OverallCapacity() == result.overallCapacity, "unnexpected overall capacity.\n exp: %+v\n act: %+v\n", result, arenaStr)

	actualOffset := arena.CurrentOffset()
	assert(expectedOffset == actualOffset, "offset mismatch.\n exp: %+v\n act: %+v\n", expectedOffset, actualOffset)
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
