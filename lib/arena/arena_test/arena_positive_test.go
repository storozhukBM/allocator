package arena_test

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"testing"
)

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
	target := arena.New(arena.Options{InitialCapacity: requiredBytesForTest})
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForTest})
	basicArenaCheckingStand(t, a)
}

func TestCreteDynamicArena(t *testing.T) {
	a := &arena.Dynamic{}
	basicArenaCheckingStand(t, a)
}

func TestCreteRawArena(t *testing.T) {
	a := arena.NewRawArena(requiredBytesForTest)
	basicArenaCheckingStand(t, a)
}
