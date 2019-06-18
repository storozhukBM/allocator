package main

import (
	"fmt"
	"github.com/storozhukBM/allocator/lib/arena"
	"reflect"
	"time"
)

func main() {
	{
		ar := arena.NewGenericAllocator(arena.Options{})

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
		ar := &arena.DynamicAllocator{}
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
		ar := arena.NewRawAllocator(1024)
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

type TimePtr arena.Ptr

func AllocTimePtr(arena *arena.GenericAllocator, target time.Time) (TimePtr, error) {
	targetType := reflect.TypeOf(time.Time{})
	aPtr, allocErr := arena.Alloc(targetType.Size(), uintptr(targetType.Align()))
	if allocErr != nil {
		return TimePtr{}, allocErr
	}
	tmpPtr := (*time.Time)(arena.ToRef(aPtr))
	*tmpPtr = target
	return TimePtr(aPtr), nil
}

func (t TimePtr) DeRef(a *arena.GenericAllocator) time.Time {
	return *(*time.Time)(a.ToRef(arena.Ptr(t)))
}

func (t TimePtr) Set(a *arena.GenericAllocator, target time.Time) {
	tmpPtr := (*time.Time)(a.ToRef(arena.Ptr(t)))
	*tmpPtr = target
}
