package arena_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/storozhukBM/allocator/lib/arena"
)

type arenaByteBufferWithPanicAllocationCheckingStand struct{}

func (s *arenaByteBufferWithPanicAllocationCheckingStand) check(t *testing.T, target allocator) {
	value := generateRandomValue()
	for i := 0; i < 15; i++ {
		arenaBuf := arena.NewBuffer(target)
		assert(arenaBuf.Len() == 0, "expect initial len to be 0")
		assert(arenaBuf.Cap() == 0, "expect initial cap to be 0")
		{
			encoder := json.NewEncoder(arenaBuf)
			for j := 0; j < 100; j++ {
				encodingErr := encoder.Encode(value)
				failOnError(t, encodingErr)
			}
		}
		heapBuf := bytes.NewBuffer(nil)
		{
			encoder := json.NewEncoder(heapBuf)
			for j := 0; j < 100; j++ {
				encodingErr := encoder.Encode(value)
				failOnError(t, encodingErr)
			}
		}
		assert(bytes.Equal(arenaBuf.Bytes(), heapBuf.Bytes()), "unnexpected buffer state")
	}
	{
		buf := arena.NewBuffer(target)
		assert(buf.Len() == 0, "expect initial len to be 0")
		assert(buf.Cap() == 0, "expect initial cap to be 0")
		n, allocErr := buf.WriteString("hello")
		failOnError(t, allocErr)
		assert(n == 5, "expect 5 bytes: %+v", n)
		assert(buf.Len() == 5, "expect len to be 5")
		assert(buf.Cap() >= 5, "expect cap to be >= 5")

		allocErr = buf.WriteByte(' ')
		failOnError(t, allocErr)
		assert(buf.Len() == 6, "expect len to be 6")
		assert(buf.Cap() >= 6, "expect cap to be >= 6")

		n, allocErr = buf.Write([]byte("sailor"))
		failOnError(t, allocErr)
		assert(n == 6, "expect 6 bytes: %+v", n)
		assert(buf.Len() == 12, "expect len to be 12")
		assert(buf.Cap() >= 12, "expect cap to be >= 12")

		helloSailor := "hello sailor"
		assert(bytes.Equal(buf.Bytes(), []byte(helloSailor)), "not expected bytes state: %+v", buf.Bytes())
		assert(
			bytes.Equal(buf.CopyBytesToHeap(), []byte(helloSailor)),
			"not expected bytes state: %+v", buf.CopyBytesToHeap(),
		)
		assert(buf.String() == helloSailor, "not expected bytes state: %+v", buf.String())
		assert(buf.CopyBytesToStringOnHeap() == helloSailor, "not expected bytes state: %+v", buf.CopyBytesToStringOnHeap())
		assert(buf.Len() == 12, "expect len to be 12")
		assert(buf.Cap() >= 12, "expect cap to be >= 12")
	}
}

type arenaByteBufferWithPanicLimitationsAllocationCheckingStand struct{}

func (s *arenaByteBufferWithPanicLimitationsAllocationCheckingStand) check(t *testing.T, target allocator) {
	initialAvailableBytes := target.Metrics().AvailableBytes
	buf := arena.NewBuffer(target)

	func() {
		defer func() {
			allocErr := recover()
			assert(allocErr == bytes.ErrTooLarge, "allocation limit should be triggered")
			assert(initialAvailableBytes == target.Metrics().AvailableBytes, "no allocation should happen: %+v", target.Metrics())
		}()
		_, _ = buf.Write(make([]byte, initialAvailableBytes+1))
	}()

	func() {
		defer func() {
			allocErr := recover()
			assert(allocErr == bytes.ErrTooLarge, "allocation limit should be triggered")
			assert(initialAvailableBytes == target.Metrics().AvailableBytes, "no allocation should happen: %+v", target.Metrics())
		}()
		_, _ = buf.WriteString(string(make([]byte, initialAvailableBytes+1)))
	}()

	n, allocErr := buf.Write(make([]byte, initialAvailableBytes))
	failOnError(t, allocErr)
	assert(n == initialAvailableBytes, "allocation should happen: %+v", n)

	func() {
		defer func() {
			allocErr := recover()
			assert(allocErr == bytes.ErrTooLarge, "allocation limit should be triggered")
		}()
		_ = buf.WriteByte(' ')
	}()
}
