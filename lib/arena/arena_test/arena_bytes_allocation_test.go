package arena_test

import (
	"bytes"
	"github.com/storozhukBM/allocator/lib/arena"
	"testing"
)

const requiredBytesForBytesAllocationTest = 36

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
	}
	{
		arenaBytes, allocErr = arena.Append(target, arenaBytes, 2, 3)
		failOnError(t, allocErr)
		buf := arena.BytesToRef(target, arenaBytes)
		assert(bytes.Equal(buf, []byte{1, 2, 3}), "unexpected buffer state: %+v", buf)
		assert(arenaBytes.Len() == 3, "unexpected bytes state: %+v", arenaBytes)
		assert(arenaBytes.Cap() == 8, "unexpected bytes state: %+v", arenaBytes)
		t.Logf("bytes state: %v", arenaBytes)
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
	}
	{
		buf := arena.BytesToRef(target, arenaBytes)
		buf[0] = 5
		buf[7] = 1
	}
	{
		buf := arena.BytesToRef(target, arenaBytes)
		assert(
			bytes.Equal(buf, []byte{5, 2, 3, 4, 5, 6, 7, 1, 9}),
			"unexpected buffer state: %+v", buf,
		)
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
