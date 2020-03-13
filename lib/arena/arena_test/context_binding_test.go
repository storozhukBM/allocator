package arena_test

import (
	"context"
	"testing"

	"github.com/storozhukBM/allocator/lib/arena"
)

func TestContextBinding(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	outOfContextArena := arena.NewGenericAllocator(arena.Options{
		DelegateClearToUnderlyingAllocator: true,
	})
	ctx = arena.WithAllocator(ctx, arena.NewSubAllocator(outOfContextArena, arena.Options{}))
	a, ok := arena.GetAllocator(ctx)
	assert(ok, "should be true")
	assert(a != nil, "a is provided")

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
	growthStand := &arenaDynamicGrowthStand{}
	growthStand.check(t, a)
	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, a)
	bytesBufferAllocationStand := &arenaByteBufferWithErrorAllocationCheckingStand{}
	bytesBufferAllocationStand.check(t, a)
	bytesBufferWithPanicAllocationStand := &arenaByteBufferWithPanicAllocationCheckingStand{}
	bytesBufferWithPanicAllocationStand.check(t, a)

	ctx = arena.WithAllocator(ctx, arena.NewSubAllocator(outOfContextArena, arena.Options{
		AllocationLimitInBytes: 1,
	}))
	a = arena.GetAllocatorOrDefault(ctx, nil)
	assert(a != nil, "a is provided")
	alloc, allocErr := a.Alloc(2, 1)
	assert(allocErr != nil, "should be limit error")
	assert(alloc == arena.Ptr{}, "should be empty")
}
