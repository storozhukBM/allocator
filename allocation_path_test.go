package allocator

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
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
				countOfAllocations:  15,
				usedBytes:           15 * personSize,
				dataBytes:           15 * personSize,
				paddingOverhead:     0,
				overallCapacity:     defaultFirstBucketSize,
				countOfBuckets:      1,
				currentBucketIdx:    0,
				currentBucketOffset: 15 * personSize,
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
					countOfAllocations:  1,
					usedBytes:           personSize,
					dataBytes:           personSize,
					paddingOverhead:     0,
					overallCapacity:     defaultFirstBucketSize,
					countOfBuckets:      1,
					currentBucketIdx:    0,
					currentBucketOffset: personSize,
				},
			},
			{
				count: 1,
				target: allocationType{
					typeName: "deal",
					typeVal:  reflect.TypeOf(deal{}),
				},
				result: allocationResult{
					countOfAllocations:  2,
					usedBytes:           personSize + dealSize,
					dataBytes:           personSize + dealSize,
					paddingOverhead:     0,
					overallCapacity:     defaultFirstBucketSize,
					countOfBuckets:      1,
					currentBucketIdx:    0,
					currentBucketOffset: personSize + dealSize,
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
					countOfAllocations:  1,
					usedBytes:           personSize,
					dataBytes:           personSize,
					paddingOverhead:     0,
					overallCapacity:     defaultFirstBucketSize,
					countOfBuckets:      1,
					currentBucketIdx:    0,
					currentBucketOffset: personSize,
				},
			},
			{
				count: 3,
				target: allocationType{
					typeName: "bool",
					typeVal:  reflect.TypeOf(true),
				},
				result: allocationResult{
					countOfAllocations:  4,
					usedBytes:           personSize + 3,
					dataBytes:           personSize + 3,
					paddingOverhead:     0,
					overallCapacity:     defaultFirstBucketSize,
					countOfBuckets:      1,
					currentBucketIdx:    0,
					currentBucketOffset: personSize + 3,
				},
			},
			{
				count: 1,
				target: allocationType{
					typeName: "person",
					typeVal:  reflect.TypeOf(person{}),
				},
				result: allocationResult{
					countOfAllocations:  5,
					dataBytes:           personSize + 3 + personSize,
					usedBytes:           personSize + 3 + 5 + personSize,
					paddingOverhead:     5,
					overallCapacity:     defaultFirstBucketSize,
					countOfBuckets:      1,
					currentBucketIdx:    0,
					currentBucketOffset: personSize + 3 + 5 + personSize,
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
			fmt.Printf("case: %v\n", path.name)
			ar := &Arena{}
			checkArenaState(ar, allocationResult{countOfBuckets: 1}, AOffset{arenaMask: ar.target.arenaMask})
			for _, alloc := range path.allocations {
				fmt.Printf(
					"allocate %v [size: %v; align: %v] x %v \n",
					alloc.target.typeName, alloc.target.typeVal.Size(), alloc.target.typeVal.Align(), alloc.count,
				)
				for i := 0; i < alloc.count; i++ {
					ptr, allocErr := ar.Alloc(alloc.target.typeVal.Size(), uintptr(alloc.target.typeVal.Align()))
					failOnError(t, allocErr)
					assert(ptr != APtr{}, "ptr is not nil")
				}
				checkArenaState(ar,
					alloc.result,
					AOffset{
						offset:    uint32(alloc.result.currentBucketOffset),
						bucketIdx: uint8(alloc.result.currentBucketIdx),
						arenaMask: ar.target.arenaMask,
					},
				)
			}
		})
	}
}
