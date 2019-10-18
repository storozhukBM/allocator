package arena

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

func BenchmarkAlign(b *testing.B) {
	b.StopTimer()
	os := make([]uint32, b.N)
	aligns := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		os[i] = uint32(rand.Intn(480000000000))
		aligns[i] = uint32(1 << uint32(rand.Intn(32)))
	}
	b.StartTimer()
	count := uint32(0)
	for i := 0; i < b.N; i++ {
		result := calculateRequiredPadding(os[i], aligns[i])
		count += result
	}
	b.StopTimer()
	fmt.Printf("%d\n", count)
}

func BenchmarkAlignMaskUnsafe(b *testing.B) {
	b.StopTimer()
	os := make([]uint32, b.N)
	aligns := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		os[i] = uint32(rand.Intn(480000000000))
		aligns[i] = uint32(1 << uint32(rand.Intn(32)))
	}
	b.StartTimer()
	count := uint32(0)
	for i := 0; i < b.N; i++ {
		result := calculateRequiredPaddingMaskUnsafe(os[i], aligns[i])
		count += result
	}
	b.StopTimer()
	fmt.Printf("%d\n", count)
}

func BenchmarkAlignMask(b *testing.B) {
	b.StopTimer()
	os := make([]uint32, b.N)
	aligns := make([]uint32, b.N)
	for i := 0; i < b.N; i++ {
		os[i] = uint32(rand.Intn(480000000000))
		aligns[i] = uint32(1 << uint32(rand.Intn(32)))
	}
	b.StartTimer()
	count := uint32(0)
	for i := 0; i < b.N; i++ {
		result := calculateRequiredPaddingMask(os[i], aligns[i])
		count += result
	}
	b.StopTimer()
	fmt.Printf("%d\n", count)
}

func TestAlternatives(t *testing.T) {
	n := 100000
	for i := 0; i < n; i++ {
		o := uint32(rand.Intn(48))
		align := uint32(1 << uint32(rand.Intn(32)))
		def := calculateRequiredPadding(o, align)
		mask := calculateRequiredPaddingMask(o, align)
		unsfe := calculateRequiredPaddingMaskUnsafe(o, align)
		if def != mask || mask != unsfe {
			fmt.Printf(
				"o: %v; align: %v; def: %v; mask: %v; unsfe: %v\n",
				o, align, def, mask, unsfe,
			)
			t.FailNow()
		}
	}
}
