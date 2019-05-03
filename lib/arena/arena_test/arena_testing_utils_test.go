package arena_test

import (
	"fmt"
	"github.com/storozhukBM/allocator/lib/arena"
	"runtime"
	"testing"
	"unsafe"
)

type allocator interface {
	Alloc(size, padding uintptr) (arena.Ptr, error)
	ToRef(ptr arena.Ptr) unsafe.Pointer
	CurrentOffset() arena.Offset
	Metrics() arena.Metrics
}

type enhancedMetricsProducer interface {
	EnhancedMetrics() arena.EnhancedMetrics
}

const requiredBytesForTest = 48

type basicArenaCheckingStand struct {
	ptrStringsSet             stringsSetWithOrder
	offsetStringsSet          stringsSetWithOrder
	metricsStringsSet         stringsSetWithOrder
	enhancedMetricsStringsSet stringsSetWithOrder
	arenaStringsSet           stringsSetWithOrder
}

func (s *basicArenaCheckingStand) checkArenaStrIsUnique(t *testing.T, arena allocator) {
	arenaStr := fmt.Sprintf("%v", arena)
	assert(arenaStr != "", "can't be empty")
	metricsAreUnique := s.arenaStringsSet.addIfUnique(arenaStr)
	assert(metricsAreUnique, "arena str should be unique")
}

func (s *basicArenaCheckingStand) checkMetricsAreUnique(t *testing.T, metrics arena.Metrics) {
	assert(metrics.String() != "", "can't be empty")
	metricsAreUnique := s.metricsStringsSet.addIfUnique(metrics.String())
	assert(metricsAreUnique, "metrics should be unique")
}

func (s *basicArenaCheckingStand) checkEnhancedMetricsAreUnique(t *testing.T, arena allocator) {
	producer, ok := arena.(enhancedMetricsProducer)
	if !ok {
		return
	}
	assert(producer.EnhancedMetrics().String() != "", "can't be empty")
	metricsAreUnique := s.enhancedMetricsStringsSet.addIfUnique(producer.EnhancedMetrics().String())
	assert(metricsAreUnique, "enhanced metrics should be unique")
}

func (s *basicArenaCheckingStand) checkPointerIsUnique(t *testing.T, ptr arena.Ptr) {
	assert(ptr.String() != "", "can't be empty")
	ptrIsUnique := s.ptrStringsSet.addIfUnique(ptr.String())
	assert(ptrIsUnique, "ptr should be unique")
}

func (s *basicArenaCheckingStand) checkOffsetIsUnique(t *testing.T, offset arena.Offset) {
	assert(offset.String() != "", "can't be empty")
	offsetIsUnique := s.offsetStringsSet.addIfUnique(offset.String())
	assert(offsetIsUnique, "offset should be unique")
}

func (s *basicArenaCheckingStand) check(t *testing.T, target allocator) {
	{
		ptr, allocErr := target.Alloc(0, 1)
		failOnError(t, allocErr)
		s.checkPointerIsUnique(t, ptr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		assert(target.Metrics().UsedBytes == 0, "expect used bytes should be 0. instead: %v", target.Metrics())
	}
	{
		ptr, allocErr := target.Alloc(1, 1)
		failOnError(t, allocErr)
		assert(ptr.String() != "", "can't be empty")
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		assert(target.Metrics().UsedBytes == 1, "expect used bytes should be 1. instead: %v", target.Metrics())
	}

	{
		ptr, allocErr := target.Alloc(3, 1)
		failOnError(t, allocErr)
		s.checkPointerIsUnique(t, ptr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		assert(target.Metrics().UsedBytes == 4, "expect used bytes should be 4. instead: %v", target.Metrics())
	}
	{
		ptr, allocErr := target.Alloc(1, 1)
		failOnError(t, allocErr)
		s.checkPointerIsUnique(t, ptr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		assert(target.Metrics().UsedBytes == 5, "expect used bytes should be 5. instead: %v", target.Metrics())
	}
	{
		ptr, testAlignmentErr := target.Alloc(4, 4)
		failOnError(t, testAlignmentErr)
		s.checkPointerIsUnique(t, ptr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)
		assert(target.Metrics().UsedBytes == 12, "expect used bytes should be 12. instead: %v", target.Metrics())
	}
	{
		boss := &person{name: "Richard Bahman", age: 44}

		personPtr, allocErr := target.Alloc(unsafe.Sizeof(person{}), unsafe.Alignof(person{}))
		failOnError(t, allocErr)
		s.checkPointerIsUnique(t, personPtr)
		s.checkOffsetIsUnique(t, target.CurrentOffset())
		s.checkMetricsAreUnique(t, target.Metrics())
		s.checkEnhancedMetricsAreUnique(t, target)
		s.checkArenaStrIsUnique(t, target)

		ref := target.ToRef(personPtr)
		rawPtr := uintptr(ref)
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			p.name = "John Smith"
			p.age = 21
			p.manager = boss
		}
		runtime.GC()
		{
			p := (*person)(unsafe.Pointer(rawPtr))
			assert(p.name == "John Smith", "unexpected person state: %+v", p)
			assert(p.age == 21, "unexpected person state: %+v", p)
			assert(p.manager.name == "Richard Bahman", "unexpected person state: %+v", p)
			assert(p.manager.age == 44, "unexpected person state: %+v", p)
		}
	}
	for _, key := range s.ptrStringsSet.list {
		t.Logf("ptr: %v\n", key)
	}
	for _, key := range s.offsetStringsSet.list {
		t.Logf("offset: %v\n", key)
	}
	for _, key := range s.metricsStringsSet.list {
		t.Logf("metrics: %v\n", key)
	}
	for _, key := range s.enhancedMetricsStringsSet.list {
		t.Logf("enhanced metrics: %v\n", key)
	}
	for _, key := range s.arenaStringsSet.list {
		t.Logf("arena: %v\n", key)
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
		panic("unexpected error happened")
	}
}

type person struct {
	name    string
	age     uint
	manager *person
}

type stringsSetWithOrder struct {
	set  map[string]struct{}
	list []string
}

func (s *stringsSetWithOrder) addIfUnique(key string) bool {
	if s.set == nil {
		s.set = map[string]struct{}{}
	}
	_, notUnique := s.set[key]
	if notUnique {
		return false
	}
	s.set[key] = struct{}{}
	s.list = append(s.list, key)
	return true
}
