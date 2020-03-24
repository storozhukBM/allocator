package allocation_bench_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"unsafe"

	"github.com/storozhukBM/allocator/lib/arena"
)

const KB = 1024
const MB = 1024 * KB

type allocator interface {
	Alloc(size uintptr, alignment uintptr) (arena.Ptr, error)
	ToRef(p arena.Ptr) unsafe.Pointer
	Metrics() arena.Metrics
	Clear()
}

type benchAlloc interface {
	allocateBytes(size int) ([]byte, error)
	clear()
}

type directArenaAlloc struct {
	alloc     allocator
	bytesView *arena.BytesView
}

func newDirectArenaAlloc(allocProvider func() allocator) benchAlloc {
	alloc := allocProvider()
	bytesView := arena.NewBytesView(alloc)
	var a benchAlloc = &directArenaAlloc{
		alloc:     alloc,
		bytesView: bytesView,
	}
	return a
}

func (g *directArenaAlloc) allocateBytes(size int) ([]byte, error) {
	bytes, allocErr := g.bytesView.MakeBytes(size)
	if allocErr != nil {
		return nil, allocErr
	}
	return g.bytesView.BytesToRef(bytes), nil
}

func (g *directArenaAlloc) clear() {
	g.alloc.Clear()
}

type genericArenaAlloc struct {
	pool      *sync.Pool
	alloc     allocator
	bytesView *arena.BytesView
}

func newManagedArenaAlloc(allocProvider func() allocator) benchAlloc {
	pool := &sync.Pool{New: func() interface{} {
		return allocProvider()
	}}
	alloc := pool.Get().(allocator)
	bytesView := arena.NewBytesView(alloc)

	var a benchAlloc = &genericArenaAlloc{
		pool:      pool,
		alloc:     alloc,
		bytesView: bytesView,
	}
	return a
}

func (g *genericArenaAlloc) allocateBytes(size int) ([]byte, error) {
	bytes, allocErr := g.bytesView.MakeBytes(size)
	if allocErr != nil {
		return nil, allocErr
	}
	return g.bytesView.BytesToRef(bytes), nil
}

func (g *genericArenaAlloc) clear() {
	allocToClear := g.alloc

	g.alloc = g.pool.Get().(allocator)
	g.bytesView = arena.NewBytesView(g.alloc)

	go func() {
		allocToClear.Clear()
		g.pool.Put(allocToClear)
	}()
}

type internalAlloc struct{}

func (i *internalAlloc) allocateBytes(size int) ([]byte, error) {
	return make([]byte, size), nil
}

func (i *internalAlloc) clear() {
}

const liveSet = 32 * MB

func BenchmarkInternalAllocator(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := &internalAlloc{}
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkManagedRawAllocator(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newManagedArenaAlloc(func() allocator {
		return arena.NewRawAllocator(liveSet)
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkManagedDynamicAllocator(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newManagedArenaAlloc(func() allocator {
		return arena.NewDynamicAllocator()
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkManagedDynamicAllocatorWithPreAlloc(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newManagedArenaAlloc(func() allocator {
		return arena.NewDynamicAllocatorWithInitialCapacity(liveSet)
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkManagedGenericAllocatorWithPreAllocWithSubClean(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newManagedArenaAlloc(func() allocator {
		return arena.NewGenericAllocator(arena.Options{
			InitialCapacity:                    liveSet,
			DelegateClearToUnderlyingAllocator: true,
		})
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkRawAllocator(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newDirectArenaAlloc(func() allocator {
		return arena.NewRawAllocator(liveSet)
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkGenericAllocatorWithSubClean(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newDirectArenaAlloc(func() allocator {
		return arena.NewGenericAllocator(arena.Options{
			DelegateClearToUnderlyingAllocator: true,
		})
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkGenericAllocatorWithoutSubClean(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newDirectArenaAlloc(func() allocator {
		return arena.NewGenericAllocator(arena.Options{})
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkGenericAllocatorWithoutSubCleanWithLimit(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newDirectArenaAlloc(func() allocator {
		return arena.NewGenericAllocator(arena.Options{
			AllocationLimitInBytes: liveSet,
		})
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkGenericAllocatorWithPreAllocWithoutSubClean(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newDirectArenaAlloc(func() allocator {
		return arena.NewGenericAllocator(arena.Options{
			InitialCapacity: liveSet,
		})
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkGenericAllocatorWithPreAllocWithSubClean(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newDirectArenaAlloc(func() allocator {
		return arena.NewGenericAllocator(arena.Options{
			InitialCapacity:                    liveSet,
			DelegateClearToUnderlyingAllocator: true,
		})
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func BenchmarkDynamicAllocator(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	a := newDirectArenaAlloc(func() allocator {
		return arena.NewDynamicAllocator()
	})
	runBenchmark(b, a, liveSet-64*KB)
}

func runBenchmark(b *testing.B, a benchAlloc, liveSetSize uint) {
	differentSizeAllocationProfile(b, a, liveSetSize)
	//sameSizeAllocationProfile(b, a, liveSet)
}

var sizesMask = 64 - 1
var sizesSlice = make([]uint16, 64)
var readIdx = make([]uint16, 64)
var writeIdx = make([]uint16, 64)

func init() {
	for i := 0; i < 64; i++ {
		sizesSlice[i] = uint16((1 << (3 + rand.Intn(12))) * (1 + rand.Intn(3)))
		readIdx[i] = uint16(rand.Intn(int(sizesSlice[i])))
		writeIdx[i] = uint16(rand.Intn(int(sizesSlice[i])))
	}
}

func differentSizeAllocationProfile(b *testing.B, a benchAlloc, liveSetSize uint) {
	benchState := 0
	currentSize := 0
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		idx := i & sizesMask
		allocSize := sizesSlice[idx]

		currentSize += int(allocSize)
		if currentSize >= int(liveSetSize) {
			a.clear()
			currentSize = 0
		}

		bytes, allocErr := a.allocateBytes(int(allocSize))
		if allocErr != nil {
			b.Error(allocErr)
			b.FailNow()
		}
		bytes[writeIdx[idx]] = 234
		benchState += int(bytes[readIdx[idx]])
	}
	b.StopTimer()
	if rand.Float64() < 0.00001 {
		fmt.Printf("N: %d; %d\n", b.N, benchState)
	}
}

func sameSizeAllocationProfile(b *testing.B, a benchAlloc, liveSetSize uint) {
	runBenchmarkForSpecificSize(b, a, liveSetSize, 64)
}

func runBenchmarkForSpecificSize(b *testing.B, a benchAlloc, liveSetSize uint, sizeClass int) {
	rIdx := rand.Intn(sizeClass)
	wIdx := rand.Intn(sizeClass)

	benchState := 0
	currentSize := 0
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		currentSize += sizeClass
		if currentSize >= int(liveSetSize) {
			a.clear()
			currentSize = 0
		}
		bytes, allocErr := a.allocateBytes(sizeClass)
		if allocErr != nil {
			b.Error(allocErr)
			b.FailNow()
		}
		bytes[wIdx] = 234
		benchState += int(bytes[rIdx])
	}
	b.StopTimer()
	if rand.Float64() < 0.0001 {
		fmt.Printf("N: %d; %d\n", b.N, benchState)
	}
}
