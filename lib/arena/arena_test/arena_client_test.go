package arena_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/storozhukBM/allocator/lib/arena"
)

func TestUninitializedBytesBuffer(t *testing.T) {
	t.Parallel()

	bytesBufferAllocationStand := &arenaByteBufferAllocationCheckingStand{}
	bytesBufferAllocationStand.check(t, nil)
}

func TestSimpleArenaWithoutConstructor(t *testing.T) {
	t.Parallel()
	a := &arena.GenericAllocator{}
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := &arena.GenericAllocator{}
	assert(other.ToRef(arena.Ptr{}) == nil, "should resolve to nil")

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
	growthStand := &arenaDynamicGrowthStand{}
	growthStand.check(t, a)
	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, a)
	bytesBufferAllocationStand := &arenaByteBufferAllocationCheckingStand{}
	bytesBufferAllocationStand.check(t, a)
}

func TestSimpleArenaWithInitialCapacity(t *testing.T) {
	t.Parallel()
	a := arena.NewGenericAllocator(arena.Options{InitialCapacity: requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
	growthStand := &arenaDynamicGrowthStand{}
	growthStand.check(t, a)
	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, a)
	bytesBufferAllocationStand := &arenaByteBufferAllocationCheckingStand{}
	bytesBufferAllocationStand.check(t, a)
}

func TestSimpleArenaWithInitialCapacityAndDelegatedClear(t *testing.T) {
	t.Parallel()
	a := arena.NewGenericAllocator(arena.Options{
		InitialCapacity:                    requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
	growthStand := &arenaDynamicGrowthStand{}
	growthStand.check(t, a)
	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, a)
	bytesBufferAllocationStand := &arenaByteBufferAllocationCheckingStand{}
	bytesBufferAllocationStand.check(t, a)
}

func TestSimpleArenaWithInitialCapacityAndAllocLimit(t *testing.T) {
	t.Parallel()
	a := arena.NewGenericAllocator(arena.Options{
		InitialCapacity:        requiredBytesForBasicTest,
		AllocationLimitInBytes: 2 * requiredBytesForBasicTest,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	allocSize := uintptr(a.Metrics().AvailableBytes + 1)
	ptr, allocErr := a.Alloc(allocSize, 1)
	assert(allocErr == arena.AllocationLimitError, "allocation limit should be triggered")
	assert(ptr == arena.Ptr{}, "ptr should be empty")

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)

	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)

	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleArenaWithInitialCapacityAndAllocLimitAndDelegatedClear(t *testing.T) {
	t.Parallel()
	a := arena.NewGenericAllocator(arena.Options{
		InitialCapacity:                    requiredBytesForBasicTest,
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	allocSize := uintptr(a.Metrics().AvailableBytes + 1)
	ptr, allocErr := a.Alloc(allocSize, 1)
	assert(allocErr == arena.AllocationLimitError, "allocation limit should be triggered")
	assert(ptr == arena.Ptr{}, "ptr should be empty")

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)

	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)

	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArena(t *testing.T) {
	t.Parallel()
	target := arena.NewGenericAllocator(arena.Options{InitialCapacity: requiredBytesForBasicTest})
	a := arena.NewSubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArenaAndDelegatedClear(t *testing.T) {
	t.Parallel()
	target := arena.NewGenericAllocator(arena.Options{InitialCapacity: requiredBytesForBasicTest})
	a := arena.NewSubAllocator(target, arena.Options{
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArenaAndNestedDelegatedClear(t *testing.T) {
	t.Parallel()
	target := arena.NewGenericAllocator(arena.Options{
		InitialCapacity:                    requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	a := arena.NewSubAllocator(target, arena.Options{
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArenaOverDynamicArena(t *testing.T) {
	t.Parallel()
	target := &arena.DynamicAllocator{}
	a := arena.NewSubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)
	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArenaOverDynamicArenaWithDelegatedClear(t *testing.T) {
	t.Parallel()
	target := &arena.DynamicAllocator{}
	a := arena.NewSubAllocator(target, arena.Options{
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)
	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArenaOverRawArena(t *testing.T) {
	t.Parallel()
	target := arena.NewRawAllocator(requiredBytesForBasicTest + 8)
	a := arena.NewSubAllocator(target, arena.Options{})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	allocSize := uintptr(a.Metrics().AvailableBytes + 1)
	ptr, allocErr := a.Alloc(allocSize, 1)
	assert(allocErr != nil, "allocation limit should be triggered")
	assert(ptr == arena.Ptr{}, "ptr should be empty")

	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArenaOverRawArenaWithDelegatedClear(t *testing.T) {
	t.Parallel()
	target := arena.NewRawAllocator(requiredBytesForBasicTest + 8)
	a := arena.NewSubAllocator(target, arena.Options{
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	allocSize := uintptr(a.Metrics().AvailableBytes + 1)
	ptr, allocErr := a.Alloc(allocSize, 1)
	assert(allocErr != nil, "allocation limit should be triggered")
	assert(ptr == arena.Ptr{}, "ptr should be empty")

	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleNestedSubArenas(t *testing.T) {
	t.Parallel()
	target := arena.NewGenericAllocator(arena.Options{InitialCapacity: 3 * requiredBytesForBasicTest})
	a := arena.NewSubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.NewSubAllocator(a, arena.Options{AllocationLimitInBytes: requiredBytesForBasicTest})
	otherStand := &basicArenaCheckingStand{}
	otherStand.check(t, other)

	restrictedArena := arena.NewSubAllocator(target, arena.Options{AllocationLimitInBytes: 10})
	widerArena := arena.NewSubAllocator(restrictedArena, arena.Options{})
	_, allocErr := widerArena.Alloc(1, 1)
	failOnError(t, allocErr)

	_, allocErr = widerArena.Alloc(10, 1)
	assert(allocErr == arena.AllocationLimitError, "allocation limit of parent arena should be triggered")

	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, target)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, a)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, a)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, a)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, other)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, other)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, other)
	}
}

func TestSimpleNestedSubArenasAndDelegatedClear(t *testing.T) {
	t.Parallel()
	target := arena.NewGenericAllocator(arena.Options{
		InitialCapacity: 3 * requiredBytesForBasicTest,
	})
	a := arena.NewSubAllocator(target, arena.Options{
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.NewSubAllocator(a, arena.Options{AllocationLimitInBytes: requiredBytesForBasicTest})
	otherStand := &basicArenaCheckingStand{}
	otherStand.check(t, other)

	restrictedArena := arena.NewSubAllocator(target, arena.Options{AllocationLimitInBytes: 10})
	widerArena := arena.NewSubAllocator(restrictedArena, arena.Options{})
	_, allocErr := widerArena.Alloc(1, 1)
	failOnError(t, allocErr)

	_, allocErr = widerArena.Alloc(10, 1)
	assert(allocErr == arena.AllocationLimitError, "allocation limit of parent arena should be triggered")

	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, target)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, a)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, a)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, a)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, other)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, other)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, other)
	}
}

func TestSimpleNestedSubArenasAndNestedDelegatedClear(t *testing.T) {
	t.Parallel()
	target := arena.NewGenericAllocator(arena.Options{
		InitialCapacity:                    3 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	a := arena.NewSubAllocator(target, arena.Options{
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.NewSubAllocator(a, arena.Options{AllocationLimitInBytes: requiredBytesForBasicTest})
	otherStand := &basicArenaCheckingStand{}
	otherStand.check(t, other)

	restrictedArena := arena.NewSubAllocator(target, arena.Options{AllocationLimitInBytes: 10})
	widerArena := arena.NewSubAllocator(restrictedArena, arena.Options{})
	_, allocErr := widerArena.Alloc(1, 1)
	failOnError(t, allocErr)

	_, allocErr = widerArena.Alloc(10, 1)
	assert(allocErr == arena.AllocationLimitError, "allocation limit of parent arena should be triggered")

	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, target)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, a)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, a)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, a)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, other)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, other)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, other)
	}
}

func TestSimpleConsecutiveSubArenas(t *testing.T) {
	t.Parallel()
	target := arena.NewGenericAllocator(arena.Options{InitialCapacity: requiredBytesForBasicTest})
	a := arena.NewSubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.NewSubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	otherStand := &basicArenaCheckingStand{}
	otherStand.check(t, other)

	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, target)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, a)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, a)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, a)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, other)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, other)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, other)
	}
}

func TestSimpleConsecutiveSubArenasWithDelegatedClear(t *testing.T) {
	t.Parallel()
	target := arena.NewGenericAllocator(arena.Options{
		InitialCapacity:                    requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	a := arena.NewSubAllocator(target, arena.Options{
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.NewSubAllocator(target, arena.Options{
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	otherStand := &basicArenaCheckingStand{}
	otherStand.check(t, other)

	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, target)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, a)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, a)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, a)
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, other)
		bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
		bytesAllocationLimitsStand.check(t, other)
		bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
		bufferAllocationLimitStand.check(t, other)
	}
}

func TestSimpleSubArenaOnNilTarget(t *testing.T) {
	t.Parallel()
	a := arena.NewSubAllocator(nil, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)

	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, a)

	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArenaOnNilTargetWithDelegatedClear(t *testing.T) {
	t.Parallel()
	a := arena.NewSubAllocator(nil, arena.Options{
		AllocationLimitInBytes:             2 * requiredBytesForBasicTest,
		DelegateClearToUnderlyingAllocator: true,
	})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)

	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, a)

	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleArena(t *testing.T) {
	t.Parallel()
	a := arena.NewGenericAllocator(arena.Options{})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	firstPtr, allocErr := a.Alloc(uintptr(a.Metrics().AvailableBytes+1), 1)
	assert(allocErr == nil, "simple arena should grow")
	assert(firstPtr != arena.Ptr{}, "firstPtr should not be empty")
	assert(a.ToRef(firstPtr) != nil, "firstPtr is not nil")

	secondPtr, allocErr := a.Alloc(uintptr(a.Metrics().AvailableBytes+1), 1)
	assert(allocErr == nil, "simple arena should grow")
	assert(secondPtr != arena.Ptr{}, "secondPtr should not be empty")
	assert(a.ToRef(secondPtr) != nil, "secondPtr is not nil")
	assert(a.ToRef(firstPtr) != nil, "firstPtr is still not nil")

	other := arena.NewGenericAllocator(arena.Options{})
	otherArenaPtr, allocErr := other.Alloc(1, 1)
	assert(allocErr == nil, "err should be nil")

	func() {
		defer func() {
			wrongArenaToRefPanic := recover()
			assert(wrongArenaToRefPanic != nil, "toRef on different arena should trigger panic")
		}()
		a.ToRef(otherArenaPtr)
	}()

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)

	growthStand := &arenaDynamicGrowthStand{}
	growthStand.check(t, a)

	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, a)

	bytesBufferAllocationStand := &arenaByteBufferAllocationCheckingStand{}
	bytesBufferAllocationStand.check(t, a)
}

func TestDynamicArena(t *testing.T) {
	t.Parallel()
	a := &arena.DynamicAllocator{}
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	firstPtr, allocErr := a.Alloc(uintptr(a.Metrics().AvailableBytes+1), 1)
	assert(allocErr == nil, "dynamic arena should grow")
	assert(firstPtr != arena.Ptr{}, "firstPtr should not be empty")
	assert(a.ToRef(firstPtr) != nil, "firstPtr is not nil")

	secondPtr, allocErr := a.Alloc(uintptr(a.Metrics().AvailableBytes+1), 1)
	assert(allocErr == nil, "dynamic arena should grow")
	assert(secondPtr != arena.Ptr{}, "secondPtr should not be empty")
	assert(a.ToRef(secondPtr) != nil, "secondPtr is not nil")
	assert(a.ToRef(firstPtr) != nil, "firstPtr is still not nil")

	other := &arena.DynamicAllocator{}
	otherArenaPtr, allocErr := other.Alloc(1, 1)
	assert(allocErr == nil, "err should be nil")

	func() {
		defer func() {
			wrongArenaToRefPanic := recover()
			assert(wrongArenaToRefPanic != nil, "toRef on different arena should trigger panic")
		}()
		a.ToRef(otherArenaPtr)
	}()

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)

	growthStand := &arenaDynamicGrowthStand{}
	growthStand.check(t, a)

	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, a)

	bytesBufferAllocationStand := &arenaByteBufferAllocationCheckingStand{}
	bytesBufferAllocationStand.check(t, a)
}

func TestRawArena(t *testing.T) {
	t.Parallel()

	a := arena.NewRawAllocator(requiredBytesForBasicTest + 8)
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	{
		allocSize := uintptr(a.Metrics().AvailableBytes + 1)
		ptr, allocErr := a.Alloc(allocSize, 1)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(ptr == arena.Ptr{}, "ptr should be empty")
	}
	{
		_, allocErr := a.Alloc(1, 1)
		failOnError(t, allocErr)
		allocSize := uintptr(a.Metrics().AvailableBytes)
		ptr, allocErr := a.Alloc(allocSize, 8)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(ptr == arena.Ptr{}, "ptr should be empty")
	}

	other := arena.NewRawAllocator(requiredBytesForBytesAllocationTest)
	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, other)

	forLimits := arena.NewRawAllocator(requiredBytesForBytesAllocationTest)
	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, forLimits)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, forLimits)
}

func TestWrongOffsetToRef(t *testing.T) {
	t.Parallel()

	recoveryHappened := false
	func() {
		defer func() {
			p := recover()
			assert(p != nil, "p can't be nil")
			assert(
				strings.Contains(fmt.Sprintf("%v", p), "index out of range"),
				"index check should fail the execution",
			)
			recoveryHappened = true
		}()
		bigAlloc := arena.NewRawAllocator(64)
		smallAlloc := arena.NewRawAllocator(32)

		_, allocErr := bigAlloc.Alloc(33, 1)
		failOnError(t, allocErr)

		ptrFromBig, allocErr := bigAlloc.Alloc(1, 1)
		failOnError(t, allocErr)

		_ = smallAlloc.ToRef(ptrFromBig)
		assert(false, "this point should be unreachable due to panic")
	}()

	assert(recoveryHappened, "recovery should happen")
}
