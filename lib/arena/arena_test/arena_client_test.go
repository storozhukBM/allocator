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
}

func TestSimpleArenaWithInitialCapacity(t *testing.T) {
	a := arena.New(arena.Options{InitialCapacity: requiredBytesForTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)
}

func TestSimpleArenaWithInitialCapacityAndAllocLimit(t *testing.T) {
	a := arena.New(arena.Options{InitialCapacity: requiredBytesForTest, AllocationLimitInBytes: 2 * requiredBytesForTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	allocSize := uintptr(a.Metrics().AvailableBytes + 1)
	ptr, allocErr := a.Alloc(allocSize, 1)
	assert(allocErr != nil, "allocation limit should be triggered")
	assert(ptr == arena.Ptr{}, "ptr should be empty")
}

func TestSimpleSubArena(t *testing.T) {
	target := arena.New(arena.Options{InitialCapacity: requiredBytesForTest})
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)
}

func TestSimpleNestedSubArenas(t *testing.T) {
	target := arena.New(arena.Options{InitialCapacity: 3 * requiredBytesForTest})
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.SubAllocator(a, arena.Options{AllocationLimitInBytes: requiredBytesForTest})
	otherStand := &basicArenaCheckingStand{}
	otherStand.check(t, other)

	restrictedArena := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 10})
	widerArena := arena.SubAllocator(restrictedArena, arena.Options{})
	_, allocErr := widerArena.Alloc(1, 1)
	failOnError(t, allocErr)

	_, allocErr = widerArena.Alloc(10, 1)
	assert(allocErr != nil, "allocation limit of parent arena should be triggered")
}

func TestSimpleConsecutiveSubArenas(t *testing.T) {
	target := arena.New(arena.Options{InitialCapacity: requiredBytesForTest})
	a := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForTest})
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	other := arena.SubAllocator(target, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForTest})
	otherStand := &basicArenaCheckingStand{}
	otherStand.check(t, other)
}

func TestSimpleSubArenaOnNilTarget(t *testing.T) {
	a := arena.SubAllocator(nil, arena.Options{AllocationLimitInBytes: 2 * requiredBytesForTest})
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
}

func TestRawArena(t *testing.T) {
	a := arena.NewRawArena(requiredBytesForTest)
	stand := &basicArenaCheckingStand{}
	stand.check(t, a)

	allocSize := uintptr(a.Metrics().AvailableBytes + 1)
	ptr, allocErr := a.Alloc(allocSize, 1)
	assert(allocErr != nil, "allocation limit should be triggered")
	assert(ptr == arena.Ptr{}, "ptr should be empty")
}
