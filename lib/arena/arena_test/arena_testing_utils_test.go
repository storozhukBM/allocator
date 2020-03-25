package arena_test

import (
	"fmt"
	"runtime/debug"
	"testing"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

type allocator interface {
	Alloc(size, alignment uintptr) (arena.Ptr, error)
	AllocUnaligned(size uintptr) (arena.Ptr, error)
	ToRef(ptr arena.Ptr) unsafe.Pointer
	CurrentOffset() arena.Offset
	Stats() arena.Stats
	Metrics() arena.Metrics
	Clear()
}

type enhancedMetricsProducer interface {
	EnhancedMetrics() arena.EnhancedMetrics
}

type commonStandState struct {
	ptrStringsSet             stringsSetWithOrder
	offsetStringsSet          stringsSetWithOrder
	metricsStringsSet         stringsSetWithOrder
	enhancedMetricsStringsSet stringsSetWithOrder
	arenaStringsSet           stringsSetWithOrder
}

func (s *commonStandState) checkArenaStrIsUnique(t *testing.T, arena allocator) {
	arenaStr := fmt.Sprintf("%v", arena)
	assert(arenaStr != "", "can't be empty")
	metricsAreUnique := s.arenaStringsSet.addIfUnique(arenaStr)
	assert(metricsAreUnique, "arena str should be unique. target: %v", arenaStr)
}

func (s *commonStandState) checkMetricsAreUnique(t *testing.T, metrics arena.Metrics) {
	assert(metrics.String() != "", "can't be empty")
	metricsAreUnique := s.metricsStringsSet.addIfUnique(metrics.String())
	assert(metricsAreUnique, "metrics should be unique. target: %v", metrics.String())
}

func (s *commonStandState) checkEnhancedMetricsAreUnique(t *testing.T, arena allocator) {
	producer, ok := arena.(enhancedMetricsProducer)
	if !ok {
		return
	}
	assert(producer.EnhancedMetrics().String() != "", "can't be empty")
	metricsAreUnique := s.enhancedMetricsStringsSet.addIfUnique(producer.EnhancedMetrics().String())
	assert(metricsAreUnique, "enhanced metrics should be unique. target: %v", producer.EnhancedMetrics().String())
}

func (s *commonStandState) checkPointerIsUnique(t *testing.T, ptr arena.Ptr) {
	assert(ptr.String() != "", "can't be empty")
	ptrIsUnique := s.ptrStringsSet.addIfUnique(ptr.String())
	assert(ptrIsUnique, "ptr should be unique. target: %v", ptr.String())
}

func (s *commonStandState) checkOffsetIsUnique(t *testing.T, offset arena.Offset) {
	assert(offset.String() != "", "can't be empty")
	offsetIsUnique := s.offsetStringsSet.addIfUnique(offset.String())
	assert(offsetIsUnique, "offset should be unique. target: %v", offset.String())
}

func (s *commonStandState) printStandState(t *testing.T) {
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
		debug.PrintStack()
		t.FailNow()
	}
}

type person struct {
	Name    string
	Age     uint
	Manager *person
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
