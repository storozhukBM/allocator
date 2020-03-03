package arena_test

import (
	"bytes"
	"testing"

	"github.com/storozhukBM/allocator/lib/arena"
)

func TestBasicOperationsOnUninitializedBuffer(t *testing.T) {
	t.Parallel()
	nilStr := "<nil>"
	{
		buf := arena.NewBufferWithError(nil)
		assert(bytes.Equal(buf.Bytes(), nil), "not expected bytes state: %+v", buf.Bytes())
		assert(bytes.Equal(buf.CopyBytesToHeap(), nil), "not expected bytes state: %+v", buf.CopyBytesToHeap())
		assert(buf.String() == nilStr, "not expected bytes state: %+v", buf.String())
		assert(buf.CopyBytesToStringOnHeap() == nilStr, "not expected bytes state: %+v", buf.CopyBytesToStringOnHeap())
		assert(buf.ArenaBytes() == arena.Bytes{}, "not expected bytes state: %+v", buf.ArenaBytes())
	}
	{
		buf := arena.NewBuffer(nil)
		assert(bytes.Equal(buf.Bytes(), nil), "not expected bytes state: %+v", buf.Bytes())
		assert(bytes.Equal(buf.CopyBytesToHeap(), nil), "not expected bytes state: %+v", buf.CopyBytesToHeap())
		assert(buf.String() == nilStr, "not expected bytes state: %+v", buf.String())
		assert(buf.CopyBytesToStringOnHeap() == nilStr, "not expected bytes state: %+v", buf.CopyBytesToStringOnHeap())
		assert(buf.ArenaBytes() == arena.Bytes{}, "not expected bytes state: %+v", buf.ArenaBytes())
	}
}
