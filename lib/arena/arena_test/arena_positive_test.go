package arena_test

import (
	"fmt"
	"github.com/storozhukBM/allocator/lib/arena"
	"testing"
)

const requiredBytesForTest = 10

func TestCreteSimpleArena(t *testing.T) {
	a := arena.New(arena.Options{})
	basicArenaCheckingStand(t, a)
}

func TestCreteSimpleArenaWithInitialCapacity(t *testing.T) {
	a := arena.New(arena.Options{InitialCapacity: requiredBytesForTest})
	basicArenaCheckingStand(t, a)
}

func TestCreteSimpleArenaWithInitialCapacityAndAllocLimit(t *testing.T) {
	a := arena.New(arena.Options{InitialCapacity: requiredBytesForTest, AllocationLimitInBytes: 2 * requiredBytesForTest})
	basicArenaCheckingStand(t, a)
}

func TestCreteSimpleSubArena(t *testing.T) {
	target := arena.New(arena.Options{InitialCapacity: requiredBytesForTest, AllocationLimitInBytes: 2 * requiredBytesForTest})
	a := arena.SubAllocator(target, arena.Options{})
	basicArenaCheckingStand(t, a)
}

type allocator interface {
	Alloc(size, padding uintptr) (arena.Ptr, error)
	Metrics() arena.Metrics
}

func basicArenaCheckingStand(t *testing.T, target allocator) {
	{
		_, zeroAllocErr := target.Alloc(0, 1)
		failOnError(t, zeroAllocErr)
		assert(target.Metrics().UsedBytes == 0, "expect used bytes should be zero. instead: %v", target.Metrics())
	}
	{
		_, oneAllocErr := target.Alloc(1, 1)
		failOnError(t, oneAllocErr)
		assert(target.Metrics().UsedBytes == 1, "expect used bytes should be one. instead: %v", target.Metrics())
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
