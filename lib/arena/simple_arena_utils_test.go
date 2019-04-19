package arena

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
	dataBytes          int
	usedBytes          int
	paddingOverhead    int

	currentBucketIdx    int
	currentBucketOffset int
}

type allocationPath struct {
	name        string
	allocations []allocation
}

func checkSimpleArena(arena *Simple, result allocationResult, expectedOffset Offset) {
	arenaStr := fmt.Sprintf("arena: %+v\n", arena)
	assertMsg := fmt.Sprintf("\n exp: %+v\n act: %+v\n", result, arenaStr)
	metrics := arena.EnhancedMetrics()
	assert(metrics.CountOfAllocations == result.countOfAllocations, "unnexpected count of allocations %v", assertMsg)
	assert(metrics.UsedBytes == result.usedBytes, "unnexpected used bytes %v", assertMsg)
	assert(metrics.DataBytes == result.dataBytes, "unnexpected data bytes %v", assertMsg)
	assert(metrics.PaddingOverhead == result.paddingOverhead, "unnexpected padding overhead %v", assertMsg)

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
	personSize  = int(reflect.TypeOf(person{}).Size())
	personAlign = int(reflect.TypeOf(person{}).Align())

	dealSize  = int(reflect.TypeOf(deal{}).Size())
	dealAlign = int(reflect.TypeOf(deal{}).Align())

	stringSize  = int(reflect.TypeOf("").Size())
	stringAlign = int(reflect.TypeOf("").Align())

	boolSize  = int(reflect.TypeOf(true).Size())
	boolAlign = int(reflect.TypeOf(true).Align())
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
