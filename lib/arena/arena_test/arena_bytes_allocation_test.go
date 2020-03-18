package arena_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/storozhukBM/allocator/lib/arena"
)

const requiredBytesForBytesAllocationTest = 64

type arenaByteAllocationCheckingStand struct{}

func (s *arenaByteAllocationCheckingStand) check(t *testing.T, target allocator) {
	alloc := arena.NewBytesView(target)
	arenaBytes, allocErr := alloc.MakeBytesWithCapacity(0, 8)
	failOnError(t, allocErr)
	assert(arenaBytes != arena.Bytes{}, "new slice can't be empty")

	{
		arenaBytes, allocErr = alloc.Append(arenaBytes, 1)
		failOnError(t, allocErr)
		buf := alloc.BytesToRef(arenaBytes)
		assert(bytes.Equal(buf, []byte{1}), "unexpected buffer state: %+v", buf)
		assert(arenaBytes.Len() == 1, "unexpected bytes state: %+v", arenaBytes)
		assert(arenaBytes.Cap() == 8, "unexpected bytes state: %+v", arenaBytes)
		t.Logf("bytes state: %v", arenaBytes)
		t.Logf("bytes as string state: %v", alloc.BytesToStringRef(arenaBytes))
	}
	{
		arenaBytes, allocErr = alloc.Append(arenaBytes, 2, 3)
		failOnError(t, allocErr)
		buf := alloc.BytesToRef(arenaBytes)
		assert(bytes.Equal(buf, []byte{1, 2, 3}), "unexpected buffer state: %+v", buf)
		assert(arenaBytes.Len() == 3, "unexpected bytes state: %+v", arenaBytes)
		assert(arenaBytes.Cap() == 8, "unexpected bytes state: %+v", arenaBytes)
		t.Logf("bytes state: %v", arenaBytes)
		t.Logf("bytes as string state: %v", alloc.BytesToStringRef(arenaBytes))
	}
	{
		arenaBytes, allocErr = alloc.Append(arenaBytes, 4, 5, 6, 7, 8, 9)
		failOnError(t, allocErr)
		buf := alloc.BytesToRef(arenaBytes)
		assert(
			bytes.Equal(buf, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}),
			"unexpected buffer state: %+v", buf,
		)
		assert(arenaBytes.Len() == 9, "unexpected bytes state: %+v", arenaBytes)
		assert(arenaBytes.Cap() >= 9, "unexpected bytes state: %+v", arenaBytes)
		t.Logf("bytes state: %v", arenaBytes)
		t.Logf("bytes as string state: %v", alloc.BytesToStringRef(arenaBytes))
	}
	fullOnHeapCopy := alloc.CopyBytesToHeap(arenaBytes)
	fullOnHeapCopyAsString := alloc.CopyBytesToStringOnHeap(arenaBytes)
	{
		assert(
			bytes.Equal(fullOnHeapCopy, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}),
			"unexpected buffer state: %+v", fullOnHeapCopy,
		)
		assert(len(fullOnHeapCopy) == 9, "unexpected bytes state: %+v", fullOnHeapCopy)
		assert(cap(fullOnHeapCopy) == 9, "unexpected bytes state: %+v", fullOnHeapCopy)
	}
	{
		buf := alloc.BytesToRef(arenaBytes)
		buf[0] = 78
		buf[7] = 65
	}
	{
		buf := alloc.BytesToRef(arenaBytes)
		expectedBytes := []byte{78, 2, 3, 4, 5, 6, 7, 65, 9}
		assert(bytes.Equal(buf, expectedBytes), "unexpected buffer state: %+v", buf)

		str := alloc.BytesToStringRef(arenaBytes)
		assert(str == string(expectedBytes), "unexpected buffer state: %+v", str)
		t.Logf("bytes as string state: %v", str)
	}
	{
		expectedBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
		assert(bytes.Equal(fullOnHeapCopy, expectedBytes), "unexpected buffer state: %+v", fullOnHeapCopy)
		assert(fullOnHeapCopyAsString == string(expectedBytes), "unexpected buffer state: %+v", fullOnHeapCopyAsString)
	}
	{
		arenaBytes, allocErr = alloc.AppendString(arenaBytes, "abc")
		failOnError(t, allocErr)
	}
	{
		buf := alloc.BytesToRef(arenaBytes)
		expectedBytes := []byte{78, 2, 3, 4, 5, 6, 7, 65, 9, 'a', 'b', 'c'}
		assert(bytes.Equal(buf, expectedBytes), "unexpected buffer state: %+v", buf)

		str := alloc.BytesToStringRef(arenaBytes)
		assert(str == string(expectedBytes), "unexpected buffer state: %+v", str)
		t.Logf("bytes as string state: %v", str)
	}

	const hello = "hello"
	{
		src := []byte(hello)
		embeddedBytes, allocErr := alloc.Embed(src)
		failOnError(t, allocErr)

		src[0] = 'g'

		assert(
			alloc.BytesToStringRef(embeddedBytes) == hello,
			"unexpected buffer state: %+v", embeddedBytes,
		)
	}
	{
		src := []byte(hello)
		embeddedBytes, allocErr := alloc.EmbedAsBytes(src)
		failOnError(t, allocErr)

		src[0] = 'g'
		assert(string(embeddedBytes) == hello, "unexpected buffer state: %+v", embeddedBytes)
	}
	{
		src := []byte(hello)
		embeddedString, allocErr := alloc.EmbedAsString(src)
		failOnError(t, allocErr)

		src[0] = 'g'
		assert(embeddedString == hello, "unexpected buffer state: %+v", embeddedString)
	}

	{
		resultBytes, allocErr := alloc.AppendString(arena.Bytes{}, "sailor")
		failOnError(t, allocErr)
		result := alloc.BytesToStringRef(resultBytes)
		assert(result == "sailor", "unexpected buffer state: %+v", result)
	}

	{
		value := []byte("123456789")
		bytesValue, allocErr := alloc.Embed(value)
		failOnError(t, allocErr)

		for l := -1; l < 10; l++ {
			for h := -1; h < 10; h++ {
				s.checkSlicing(alloc, bytesValue, value, l, h)
			}
		}
	}
}

func (s *arenaByteAllocationCheckingStand) checkSlicing(allocator *arena.BytesView, bytesValue arena.Bytes, value []byte, low int, high int) {
	assert(
		string(value) == allocator.BytesToStringRef(bytesValue),
		"exp: `%v`;\nact:`%v`", string(value), allocator.BytesToStringRef(bytesValue),
	)

	expected := s.subSlice(value, low, high)
	actual := s.subSliceBytes(allocator, bytesValue, low, high)

	outOfBoundsPrefix := "runtime error: slice bounds out of range ["
	if strings.HasPrefix(expected, outOfBoundsPrefix) && strings.HasPrefix(actual, outOfBoundsPrefix) {
		return
	}
	assert(
		expected == actual, "value: `%v`; l: %v; h: %v;\nexp: %v\nact: %v",
		value, low, high, expected, actual,
	)
}

func (s *arenaByteAllocationCheckingStand) subSlice(value []byte, low int, high int) (result string) {
	defer func() {
		err := recover()
		if err != nil {
			result = fmt.Sprintf("%v", err)
		}
	}()
	subslice := value[low:high]
	expectedResult := s.printSliceProperties(subslice)
	return expectedResult
}

func (s *arenaByteAllocationCheckingStand) subSliceBytes(allocator *arena.BytesView, value arena.Bytes, low int, high int) (result string) {
	defer func() {
		err := recover()
		if err != nil {
			result = fmt.Sprintf("%v", err)
		}
	}()
	subslice := value.SubSlice(low, high)
	expectedResult := s.printSliceProperties(allocator.BytesToRef(subslice))
	return expectedResult
}

func (s *arenaByteAllocationCheckingStand) printSliceProperties(slice []byte) string {
	return fmt.Sprintf("len: %v; cap: %v; value: `%v`", len(slice), cap(slice), string(slice))
}

type arenaByteAllocationLimitsCheckingStand struct{}

func (s *arenaByteAllocationLimitsCheckingStand) check(t *testing.T, target allocator) {
	alloc := arena.NewBytesView(target)
	{
		arenaBytes, allocErr := alloc.MakeBytes(target.Metrics().AvailableBytes + 1)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
	}
	{
		buf := make([]byte, target.Metrics().AvailableBytes+1)
		arenaBytes, allocErr := alloc.Embed(buf)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
	}
	{
		buf := make([]byte, target.Metrics().AvailableBytes+1)
		arenaStr, allocErr := alloc.EmbedAsString(buf)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaStr == "", "arenaBytes should be empty")
	}
	{
		buf := make([]byte, target.Metrics().AvailableBytes+1)
		arenaBytes, allocErr := alloc.EmbedAsBytes(buf)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == nil, "arenaBytes should be empty")
	}
	{
		arenaBytes, allocErr := alloc.MakeBytesWithCapacity(0, target.Metrics().AvailableBytes+1)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
	}
	{
		arenaBytes, allocNoErr := alloc.MakeBytesWithCapacity(0, 1)
		failOnError(t, allocNoErr)
		assert(arenaBytes != arena.Bytes{}, "arenaBytes shouldn't be empty")

		toAppend := make([]byte, target.Metrics().AvailableBytes+1)
		arenaBytes, allocErr := alloc.Append(arenaBytes, toAppend...)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
	}
	{
		arenaBytes, allocNoErr := alloc.MakeBytesWithCapacity(0, 0)
		failOnError(t, allocNoErr)
		assert(arenaBytes != arena.Bytes{}, "arenaBytes shouldn't be empty")

		toAppend := make([]byte, target.Metrics().AvailableBytes+1)
		arenaBytes, allocErr := alloc.Append(arenaBytes, toAppend...)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "bytes should be empty")
	}
}
