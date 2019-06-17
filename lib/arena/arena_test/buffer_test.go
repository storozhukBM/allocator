package arena_test

import (
	"encoding/json"
	"github.com/storozhukBM/allocator/lib/arena"
	"math/rand"
	"strconv"
	"testing"
)

func TestAllocationToBuffer(t *testing.T) {
	target := &arena.Simple{}
	value := map[string]interface{}{}
	subValue := value
	for i := 0; i < 100; i++ {
		if rand.Float32() < 0.2 {
			k := rand.Int()
			newTarget := make(map[string]interface{})
			subValue[strconv.Itoa(k)] = newTarget
			subValue = newTarget
		}
		k := rand.Int()
		v := rand.Int()
		subValue[strconv.Itoa(k)] = v
	}
	for i := 0; i < 15; i++ {
		arenaBuf := arena.NewBuffer(target)
		encoder := json.NewEncoder(arenaBuf)
		for j := 0; j < 100; j++ {
			encodingErr := encoder.Encode(value)
			failOnError(t, encodingErr)
		}
	}
	t.Logf("arena after buffer %v", target.EnhancedMetrics())
}
