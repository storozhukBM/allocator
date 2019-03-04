package main

import (
	"fmt"
	"github.com/storozhukBM/allocator"
	"reflect"
	"time"
)

func main() {
	{
		ar := &allocator.Arena{}

		tPtr := AllocTimePtr(ar, time.Now())
		fmt.Printf("%+v\n", tPtr)
		fmt.Printf("%+v\n", tPtr.DeRef(ar))

		tPtr.Set(ar, tPtr.DeRef(ar).Add(time.Hour))

		fmt.Printf("%+v\n", tPtr)
		fmt.Printf("%+v\n", tPtr.DeRef(ar))
	}

	{
		ar := &allocator.RawArena{}
		timeSize := reflect.TypeOf(time.Time{}).Size()
		aPtr := ar.Alloc(timeSize)
		timeRef := (*time.Time)(ar.ToRef(aPtr))
		*timeRef = time.Now()

		fmt.Printf("%+v\n", aPtr)
		fmt.Printf("%+v\n", *(*time.Time)(ar.ToRef(aPtr)))

		newTime := (*time.Time)(ar.ToRef(aPtr)).Add(time.Hour)
		newTimeRef := (*time.Time)(ar.ToRef(aPtr))
		*newTimeRef = newTime

		fmt.Printf("%+v\n", aPtr)
		fmt.Printf("%+v\n", *(*time.Time)(ar.ToRef(aPtr)))
	}
}

type TimePtr allocator.APtr

func AllocTimePtr(arena *allocator.Arena, target time.Time) TimePtr {
	timeSize := reflect.TypeOf(time.Time{}).Size()
	aPtr := arena.Alloc(timeSize)
	tmpPtr := (*time.Time)(arena.ToRef(aPtr))
	*tmpPtr = target
	return TimePtr(aPtr)
}

func (t TimePtr) DeRef(arena *allocator.Arena) time.Time {
	return *(*time.Time)(arena.ToRef(allocator.APtr(t)))
}

func (t TimePtr) Set(arena *allocator.Arena, target time.Time) {
	tmpPtr := (*time.Time)(arena.ToRef(allocator.APtr(t)))
	*tmpPtr = target
}
