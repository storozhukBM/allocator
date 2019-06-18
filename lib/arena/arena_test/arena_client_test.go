package arena_test

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"testing"
)

func TestUninitializedBytesBuffer(t *testing.T) {
	t.Parallel()

	bytesBufferAllocationStand := &arenaByteBufferAllocationCheckingStand{}
	bytesBufferAllocationStand.check(t, nil)
}

func TestSimpleArenaWithoutConstructor(t *testing.T) {
	t.Parallel()
	a := &arena.Simple{}
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := &arena.Simple{}
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
	a := arena.New(arena.Options{InitialCapacity: requiredBytesForBasicTest})
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
	a := arena.New(arena.Options{InitialCapacity: requiredBytesForBasicTest, AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
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
	target := arena.New(arena.Options{InitialCapacity: requiredBytesForBasicTest})
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
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
	target := &arena.Dynamic{}
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)
	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, a)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, a)
}

func TestSimpleSubArenaOverRawArena(t *testing.T) {
	t.Parallel()
	target := arena.NewRawArena(requiredBytesForBasicTest + 8)
	a := arena.SubAllocator(target, arena.Options{})
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
	target := arena.New(arena.Options{InitialCapacity: 3 * requiredBytesForBasicTest})
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.SubAllocator(a, arena.Options{AllocationLimitInBytes: requiredBytesForBasicTest})
	otherStand := &basicArenaCheckingStand{}
	otherStand.check(t, other)

	restrictedArena := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 10})
	widerArena := arena.SubAllocator(restrictedArena, arena.Options{})
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
	target := arena.New(arena.Options{InitialCapacity: requiredBytesForBasicTest})
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
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
	a := arena.SubAllocator(nil, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
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
	a := arena.New(arena.Options{})
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

	other := arena.New(arena.Options{})
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
	a := &arena.Dynamic{}
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

	other := &arena.Dynamic{}
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
	a := arena.NewRawArena(requiredBytesForBasicTest + 8)
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

	other := arena.NewRawArena(requiredBytesForBytesAllocationTest)
	bytesAllocationStand := &arenaByteAllocationCheckingStand{}
	bytesAllocationStand.check(t, other)

	forLimits := arena.NewRawArena(requiredBytesForBytesAllocationTest)
	bytesAllocationLimitsStand := &arenaByteAllocationLimitsCheckingStand{}
	bytesAllocationLimitsStand.check(t, forLimits)
	bufferAllocationLimitStand := &arenaByteBufferLimitationsAllocationCheckingStand{}
	bufferAllocationLimitStand.check(t, forLimits)
}
