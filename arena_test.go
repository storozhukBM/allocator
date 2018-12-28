package arena

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
	"time"
	"unsafe"
)

type person struct {
	name    string
	age     uint
	manager *person
}

func TestName(t *testing.T) {
	ar := NewArena()
	bos := &person{name: "Boss", age: 55}
	cache := make(map[string]*person)
	for i := 1; i < 10000; i++ {
		p := (*person)(ar.Alloc(unsafe.Sizeof(person{})))
		p.name = strconv.Itoa(i)
		p.age = uint(i)
		if i%4 == 0 {
			p.manager = cache[strconv.Itoa(i-1)]
		} else {
			p.manager = bos
		}
		cache[p.name] = p
	}

	runtime.GC()
	time.Sleep(2 * time.Second)

	for name, p := range cache {
		expectedAge, parseErr := strconv.Atoi(name)
		failOnError(t, parseErr)
		assert(t, uint(expectedAge) == p.age, "unexpected age of person: %+v", p)
		if expectedAge%4 == 0 {
			assert(t, p.manager == cache[strconv.Itoa(expectedAge-1)], "unexpected manager of person: %+v; bos: %+v", p, p.manager)
		} else {
			assert(t, p.manager == bos, "unexpected manager of person: %+v; bos: %+v", p, p.manager)
		}
	}
	for _, b := range ar.buckets {
		fmt.Printf("%v;%v\n", len(b.buffer), b.offset)
	}
}

func assert(t *testing.T, condition bool, msg string, args ...interface{}) {
	if !condition {
		t.Errorf(msg, args...)
		t.FailNow()
	}
}

func failOnError(t *testing.T, e error) {
	if e != nil {
		t.Error(e)
		t.FailNow()
	}
}
