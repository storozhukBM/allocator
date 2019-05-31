package arena_test

import (
	"github.com/storozhukBM/allocator/lib/arena"
	"testing"
)

func TestSimpleArenaWithoutConstructor(t *testing.T) {
	a := &arena.Simple{}
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := &arena.Simple{}
	assert(other.ToRef(arena.Ptr{}) == nil, "should resolve to nil")

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)

	growthStand := &arenaDynamicGrowthStand{}
	growthStand.check(t, a)
}

func TestSimpleArenaWithInitialCapacity(t *testing.T) {
	a := arena.New(arena.Options{InitialCapacity: requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
	growthStand := &arenaDynamicGrowthStand{}
	growthStand.check(t, a)
}

func TestSimpleArenaWithInitialCapacityAndAllocLimit(t *testing.T) {
	a := arena.New(arena.Options{InitialCapacity: requiredBytesForBasicTest, AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	allocSize := uintptr(a.Metrics().AvailableBytes + 1)
	ptr, allocErr := a.Alloc(allocSize, 1)
	assert(allocErr == arena.AllocationLimitError, "allocation limit should be triggered")
	assert(ptr == arena.Ptr{}, "ptr should be empty")

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
}

func TestSimpleSubArena(t *testing.T) {
	target := arena.New(arena.Options{InitialCapacity: requiredBytesForBasicTest})
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	maskStand := &arenaMaskCheckingStand{}
	maskStand.check(t, a)
}

func TestSimpleSubArenaOverDynamicArena(t *testing.T) {
	target := &arena.Dynamic{}
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForMaskTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)
}

func TestSimpleSubArenaOverRawArena(t *testing.T) {
	target := arena.NewRawArena(requiredBytesForMaskTest)
	a := arena.SubAllocator(target, arena.Options{})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	allocSize := uintptr(a.Metrics().AvailableBytes + 1)
	ptr, allocErr := a.Alloc(allocSize, 1)
	assert(allocErr != nil, "allocation limit should be triggered")
	assert(ptr == arena.Ptr{}, "ptr should be empty")
}

func TestSimpleNestedSubArenas(t *testing.T) {
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
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, other)
	}
}

func TestSimpleConsecutiveSubArenas(t *testing.T) {
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
	}
	{
		maskStand := &arenaMaskCheckingStand{}
		maskStand.check(t, other)
	}
}

func TestSimpleSubArenaOnNilTarget(t *testing.T) {
	a := arena.SubAllocator(nil, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForBasicTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)
}

func TestSimpleArena(t *testing.T) {
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
}

func TestDynamicArena(t *testing.T) {
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
}

func TestRawArena(t *testing.T) {
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
}
