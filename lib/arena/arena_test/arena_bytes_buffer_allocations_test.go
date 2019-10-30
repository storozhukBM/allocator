package arena_test

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"strconv"
	"testing"

	"github.com/storozhukBM/allocator/lib/arena"
)

type arenaByteBufferAllocationCheckingStand struct{}

func (s *arenaByteBufferAllocationCheckingStand) check(t *testing.T, target allocator) {
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

		assert(bytes.Equal(buf.Bytes(), []byte("hello sailor")), "not expected bytes state: %+v", buf.Bytes())
		assert(bytes.Equal(buf.CopyBytesToHeap(), []byte("hello sailor")), "not expected bytes state: %+v", buf.CopyBytesToHeap())
		assert(buf.String() == "hello sailor", "not expected bytes state: %+v", buf.String())
		assert(buf.CopyBytesToStringOnHeap() == "hello sailor", "not expected bytes state: %+v", buf.CopyBytesToStringOnHeap())
		assert(buf.Len() == 12, "expect len to be 12")
		assert(buf.Cap() >= 12, "expect cap to be >= 12")
	}
}

type arenaByteBufferLimitationsAllocationCheckingStand struct{}

func (s *arenaByteBufferLimitationsAllocationCheckingStand) check(t *testing.T, target allocator) {
	initialAvailableBytes := target.Metrics().AvailableBytes
	buf := arena.NewBuffer(target)
	{
		n, allocErr := buf.Write(make([]byte, initialAvailableBytes+1))
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(n == 0, "no allocation should happen: %+v", n)
		assert(initialAvailableBytes == target.Metrics().AvailableBytes, "no allocation should happen: %+v", target.Metrics())
	}
	{
		n, allocErr := buf.WriteString(string(make([]byte, initialAvailableBytes+1)))
		assert(allocErr != nil, "allocation limit should be triggered")
		assert(n == 0, "no allocation should happen: %+v", n)
		assert(initialAvailableBytes == target.Metrics().AvailableBytes, "no allocation should happen: %+v", target.Metrics())
	}
	{
		n, allocErr := buf.Write(make([]byte, initialAvailableBytes))
		failOnError(t, allocErr)
		assert(n == initialAvailableBytes, "allocation should happen: %+v", n)
	}
	{
		allocErr := buf.WriteByte(' ')
		assert(allocErr != nil, "allocation limit should be triggered")
	}
}

func generateRandomValue() map[string]interface{} {
	value := map[string]interface{}{}
	subValue := value
	for i := 0; i < 100; i++ {
		if rand.Float32() < 0.2 {
			k := "key:" + strconv.Itoa(rand.Int())
			newTarget := make(map[string]interface{})
			subValue[k] = newTarget
			subValue = newTarget
		}
		k := rand.Int()
		v := rand.NormFloat64()
		subValue[strconv.Itoa(k)] = v
	}
	return value
}
