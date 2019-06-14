package arena_test

import (
	"bytes"
	"github.com/storozhukBM/allocator/lib/arena"
	"testing"
)

const requiredBytesForBytesAllocationTest = 64

type arenaByteAllocationCheckingStand struct {
	commonStandState
}

func (s *arenaByteAllocationCheckingStand) check(t *testing.T, target allocator) {
	arenaBytes, allocErr := arena.MakeBytesWithCapacity(target, 0, 8)
	failOnError(t, allocErr)
	assert(arenaBytes != arena.Bytes{}, "new slice can't be empty")

	{
		arenaBytes, allocErr = arena.Append(target, arenaBytes, 1)
		failOnError(t, allocErr)
		buf := arena.BytesToRef(target, arenaBytes)
		assert(bytes.Equal(buf, []byte{1}), "unexpected buffer state: %+v", buf)
		assert(arenaBytes.Len() == 1, "unexpected bytes state: %+v", arenaBytes)
		assert(arenaBytes.Cap() == 8, "unexpected bytes state: %+v", arenaBytes)
		t.Logf("bytes state: %v", arenaBytes)
		t.Logf("bytes as string state: %v", arena.BytesToStringRef(target, arenaBytes))
	}
	{
		arenaBytes, allocErr = arena.Append(target, arenaBytes, 2, 3)
		failOnError(t, allocErr)
		buf := arena.BytesToRef(target, arenaBytes)
		assert(bytes.Equal(buf, []byte{1, 2, 3}), "unexpected buffer state: %+v", buf)
		assert(arenaBytes.Len() == 3, "unexpected bytes state: %+v", arenaBytes)
		assert(arenaBytes.Cap() == 8, "unexpected bytes state: %+v", arenaBytes)
		t.Logf("bytes state: %v", arenaBytes)
		t.Logf("bytes as string state: %v", arena.BytesToStringRef(target, arenaBytes))
	}
	{
		arenaBytes, allocErr = arena.Append(target, arenaBytes, 4, 5, 6, 7, 8, 9)
		failOnError(t, allocErr)
		buf := arena.BytesToRef(target, arenaBytes)
		assert(
			bytes.Equal(buf, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}),
			"unexpected buffer state: %+v", buf,
		)
		assert(arenaBytes.Len() == 9, "unexpected bytes state: %+v", arenaBytes)
		assert(arenaBytes.Cap() >= 9, "unexpected bytes state: %+v", arenaBytes)
		t.Logf("bytes state: %v", arenaBytes)
		t.Logf("bytes as string state: %v", arena.BytesToStringRef(target, arenaBytes))
	}
	fullOnHeapCopy := arena.CopyBytesToHeap(target, arenaBytes)
	fullOnHeapCopyAsString := arena.CopyBytesToStringOnHeap(target, arenaBytes)
	{
		assert(
			bytes.Equal(fullOnHeapCopy, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}),
			"unexpected buffer state: %+v", fullOnHeapCopy,
		)
		assert(len(fullOnHeapCopy) == 9, "unexpected bytes state: %+v", fullOnHeapCopy)
		assert(cap(fullOnHeapCopy) == 9, "unexpected bytes state: %+v", fullOnHeapCopy)
	}
	{
		buf := arena.BytesToRef(target, arenaBytes)
		buf[0] = 78
		buf[7] = 65
	}
	{
		buf := arena.BytesToRef(target, arenaBytes)
		expectedBytes := []byte{78, 2, 3, 4, 5, 6, 7, 65, 9}
		assert(bytes.Equal(buf, expectedBytes), "unexpected buffer state: %+v", buf)

		str := arena.BytesToStringRef(target, arenaBytes)
		assert(str == string(expectedBytes), "unexpected buffer state: %+v", str)
		t.Logf("bytes as string state: %v", str)
	}
	{
		expectedBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
		assert(bytes.Equal(fullOnHeapCopy, expectedBytes), "unexpected buffer state: %+v", fullOnHeapCopy)
		assert(fullOnHeapCopyAsString == string(expectedBytes), "unexpected buffer state: %+v", fullOnHeapCopyAsString)
	}

	{
		src := []byte("hello")
		embeddedBytes, allocErr := arena.Embed(target, src)
		failOnError(t, allocErr)

		src[0] = 'g'

		assert(
			arena.BytesToStringRef(target, embeddedBytes) == "hello",
			"unexpected buffer state: %+v", embeddedBytes,
		)
	}
	{
		src := []byte("hello")
		embeddedBytes, allocErr := arena.EmbedAsBytes(target, src)
		failOnError(t, allocErr)

		src[0] = 'g'
		assert(string(embeddedBytes) == "hello", "unexpected buffer state: %+v", embeddedBytes)
	}
	{
		src := []byte("hello")
		embeddedString, allocErr := arena.EmbedAsString(target, src)
		failOnError(t, allocErr)

		src[0] = 'g'
		assert(embeddedString == "hello", "unexpected buffer state: %+v", embeddedString)
	}
}

type arenaByteAllocationLimitsCheckingStand struct {
	commonStandState
}

func (s *arenaByteAllocationLimitsCheckingStand) check(t *testing.T, target allocator) {
	{
		arenaBytes, allocErr := arena.MakeBytes(target, uintptr(target.Metrics().AvailableBytes+1))
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
	}
	{
		buf := make([]byte, target.Metrics().AvailableBytes+1)
		arenaBytes, allocErr := arena.Embed(target, buf)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
	}
	{
		buf := make([]byte, target.Metrics().AvailableBytes+1)
		arenaStr, allocErr := arena.EmbedAsString(target, buf)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaStr == "", "arenaBytes should be empty")
	}
	{
		buf := make([]byte, target.Metrics().AvailableBytes+1)
		arenaBytes, allocErr := arena.EmbedAsBytes(target, buf)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == nil, "arenaBytes should be empty")
	}
	{
		arenaBytes, allocErr := arena.MakeBytesWithCapacity(target, 0, uintptr(target.Metrics().AvailableBytes+1))
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
	}
	{
		arenaBytes, allocNoErr := arena.MakeBytesWithCapacity(target, 0, 1)
		failOnError(t, allocNoErr)
		assert(arenaBytes != arena.Bytes{}, "arenaBytes shouldn't be empty")

		toAppend := make([]byte, target.Metrics().AvailableBytes+1)
		arenaBytes, allocErr := arena.Append(target, arenaBytes, toAppend...)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "arenaBytes should be empty")
	}
	{
		arenaBytes, allocNoErr := arena.MakeBytesWithCapacity(target, 0, 0)
		failOnError(t, allocNoErr)
		assert(arenaBytes != arena.Bytes{}, "arenaBytes shouldn't be empty")

		toAppend := make([]byte, target.Metrics().AvailableBytes+1)
		arenaBytes, allocErr := arena.Append(target, arenaBytes, toAppend...)
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(arenaBytes == arena.Bytes{}, "bytes should be empty")
	}
}
