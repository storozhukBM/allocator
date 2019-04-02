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

		tPtr, allocErr := AllocTimePtr(ar, time.Now())
		if allocErr != nil {
			panic(allocErr.Error())
		}
		fmt.Printf("%+v\n", tPtr)
		fmt.Printf("%+v\n", tPtr.DeRef(ar))

		tPtr.Set(ar, tPtr.DeRef(ar).Add(time.Hour))

		fmt.Printf("%+v\n", tPtr)
		fmt.Printf("%+v\n", tPtr.DeRef(ar))
	}

	{
		ar := &allocator.DynamicArena{}
		timeSize := reflect.TypeOf(time.Time{}).Size()
		aPtr, allocErr := ar.Alloc(timeSize)
		if allocErr != nil {
			panic(allocErr.Error())
		}
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

	{
		ar := &allocator.RawArena{}
		timeSize := reflect.TypeOf(time.Time{}).Size()
		aPtr, allocErr := ar.Alloc(timeSize)
		if allocErr != nil {
			panic(allocErr.Error())
		}
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

func AllocTimePtr(arena *allocator.Arena, target time.Time) (TimePtr, error) {
	timeSize := reflect.TypeOf(time.Time{}).Size()
	aPtr, allocErr := arena.Alloc(timeSize)
	if allocErr != nil {
		return TimePtr{}, allocErr
	}
	tmpPtr := (*time.Time)(arena.ToRef(aPtr))
	*tmpPtr = target
	return TimePtr(aPtr), nil
}

func (t TimePtr) DeRef(arena *allocator.Arena) time.Time {
	return *(*time.Time)(arena.ToRef(allocator.APtr(t)))
}

func (t TimePtr) Set(arena *allocator.Arena, target time.Time) {
	tmpPtr := (*time.Time)(arena.ToRef(allocator.APtr(t)))
	*tmpPtr = target
}
