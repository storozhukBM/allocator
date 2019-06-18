package arena_test

import (
	"bytes"
	"github.com/storozhukBM/allocator/lib/arena"
	"testing"
)

func TestBasicOperationsOnUninitializedBuffer(t *testing.T) {
	t.Parallel()
	buf := arena.NewBuffer(nil)
	assert(bytes.Equal(buf.Bytes(), nil), "not expected bytes state: %+v", buf.Bytes())
	assert(bytes.Equal(buf.CopyBytesToHeap(), nil), "not expected bytes state: %+v", buf.CopyBytesToHeap())
	assert(buf.String() == "", "not expected bytes state: %+v", buf.String())
	assert(buf.CopyBytesToStringOnHeap() == "", "not expected bytes state: %+v", buf.CopyBytesToStringOnHeap())
	assert(buf.ArenaBytes() == arena.Bytes{}, "not expected bytes state: %+v", buf.ArenaBytes())
}
