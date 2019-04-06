package main

import (
	"fmt"
	"github.com/storozhukBM/allocator"
	"reflect"
	"time"
)

func main() {
	{
		ar := &allocator.SimpleArena{}

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

	timeType := reflect.TypeOf(time.Time{})
	{
		ar := &allocator.DynamicArena{}
		aPtr, allocErr := ar.Alloc(timeType.Size(), uintptr(timeType.Align()))
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
		timeSize := timeType.Size()
		aPtr, allocErr := ar.Alloc(timeSize, uintptr(timeType.Align()))
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

func AllocTimePtr(arena *allocator.SimpleArena, target time.Time) (TimePtr, error) {
	targetType := reflect.TypeOf(time.Time{})
	aPtr, allocErr := arena.Alloc(targetType.Size(), uintptr(targetType.Align()))
	if allocErr != nil {
		return TimePtr{}, allocErr
	}
	tmpPtr := (*time.Time)(arena.ToRef(aPtr))
	*tmpPtr = target
	return TimePtr(aPtr), nil
}

func (t TimePtr) DeRef(arena *allocator.SimpleArena) time.Time {
	return *(*time.Time)(arena.ToRef(allocator.APtr(t)))
}

func (t TimePtr) Set(arena *allocator.SimpleArena, target time.Time) {
	tmpPtr := (*time.Time)(arena.ToRef(allocator.APtr(t)))
	*tmpPtr = target
}
