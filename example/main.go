package main

import (
	"fmt"
	"github.com/storozhukBM/allocator"
	"reflect"
	"time"
)

func main() {
	ar := &allocator.Arena{}

	tPtr := AllocTimePtr(ar, time.Now())
	fmt.Printf("%+v\n", tPtr)
	fmt.Printf("%+v\n", tPtr.DeRef(ar))

	tPtr.Set(ar, tPtr.DeRef(ar).Add(time.Hour))

	fmt.Printf("%+v\n", tPtr)
	fmt.Printf("%+v\n", tPtr.DeRef(ar))
}

type TimePtr allocator.APtr

func AllocTimePtr(arena *allocator.Arena, target time.Time) TimePtr {
	timeSize := reflect.TypeOf(time.Time{}).Size()
	aPtr := arena.Alloc(timeSize)
	tmpPtr := (*time.Time)(aPtr.ToRef(arena))
	*tmpPtr = target
	return TimePtr(aPtr)
}

func (t TimePtr) DeRef(arena *allocator.Arena) time.Time {
	return *(*time.Time)(allocator.APtr(t).ToRef(arena))
}

func (t TimePtr) Set(arena *allocator.Arena, target time.Time) {
	tmpPtr := (*time.Time)(allocator.APtr(t).ToRef(arena))
	*tmpPtr = target
}
