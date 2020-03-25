package alignment_bench_test

import (
	"fmt"
	"math/rand"
	"testing"
)

func calculateRequiredPadding(o uint32, targetAlignment uint32) uint32 {
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (o % targetAlignment)) % targetAlignment
}

func calculateRequiredPaddingMask(o uint32, targetAlignment uint32) uint32 {
	mask := targetAlignment - 1
	mask |= mask >> 1
	mask |= mask >> 2
	mask |= mask >> 4
	mask |= mask >> 8
	mask |= mask >> 16

	// go compiler should optimise it and use mask operations
	return (targetAlignment - (o & mask)) & mask
}

func calculateRequiredPaddingMaskUnsafe(o uint32, targetAlignment uint32) uint32 {
	mask := targetAlignment - 1
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (o & mask)) & mask
}

func calculateRequiredPaddingMaskUnsafeWithShort(o uint32, targetAlignment uint32) uint32 {
	if targetAlignment == 1 {
		return 0
	}
	mask := targetAlignment - 1
	// go compiler should optimise it and use mask operations
	return (targetAlignment - (o & mask)) & mask
}

func benchmarkAlignment(b *testing.B, alignmentFunc func(o, align uint32) uint32) {
	b.StopTimer()

	tableSize := 64
	idxMask := tableSize - 1
	os := make([]uint32, tableSize)
	aligns := make([]uint32, tableSize)
	for i := 0; i < tableSize; i++ {
		os[i] = uint32(rand.Intn(480000000000))
		aligns[i] = uint32(1 << uint32(rand.Intn(32)))
	}

	b.StartTimer()
	count := uint32(0)
	for i := 0; i < b.N; i++ {
		idx := i & idxMask
		result := alignmentFunc(os[idx], aligns[idx])
		count += result
	}
	b.StopTimer()
	if rand.Float64() < 0.00001 {
		fmt.Printf("%d\n", count)
	}
}

func BenchmarkAlign(b *testing.B) {
	benchmarkAlignment(b, calculateRequiredPadding)
}

func BenchmarkAlignMaskUnsafe(b *testing.B) {
	benchmarkAlignment(b, calculateRequiredPaddingMaskUnsafe)
}

func BenchmarkAlignMaskUnsafeWithShort(b *testing.B) {
	benchmarkAlignment(b, calculateRequiredPaddingMaskUnsafeWithShort)
}

func BenchmarkAlignMask(b *testing.B) {
	benchmarkAlignment(b, calculateRequiredPaddingMask)
}

func TestAlternatives(t *testing.T) {
	n := 100000
	for i := 0; i < n; i++ {
		o := uint32(rand.Intn(48))
		align := uint32(1 << uint32(rand.Intn(32)))
		def := calculateRequiredPadding(o, align)
		mask := calculateRequiredPaddingMask(o, align)
		unsfe := calculateRequiredPaddingMaskUnsafe(o, align)
		unsfeWS := calculateRequiredPaddingMaskUnsafeWithShort(o, align)
		if def != mask || mask != unsfe || unsfe != unsfeWS {
			fmt.Printf(
				"o: %v; align: %v; def: %v; mask: %v; unsfe: %v; unsfeWS: %v\n",
				o, align, def, mask, unsfe, unsfeWS,
			)
			t.FailNow()
		}
	}
}
