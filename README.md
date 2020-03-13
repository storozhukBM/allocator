# allocator. WIP. This library is under construction.
![Go](https://github.com/storozhukBM/allocator/workflows/Go/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/storozhukBM/allocator)](https://goreportcard.com/report/github.com/storozhukBM/allocator) [![GoDoc](https://godoc.org/github.com/storozhukBM/allocator?status.svg)](http://godoc.org/github.com/storozhukBM/allocator)
#
Primitive arena allocator

List build targets
```
go run ./cmd/internal -h
```

Verify
```
go run ./cmd/internal verify
```
Or
```
go run ./cmd/internal itself
./make verify
```

TODO:
First Release scope
1. whole documentation with notion of unsafe semantics
1. mention thread safety in documentation, and share of arena allocated resources between goroutines
1. add sub-slicing to the generated code and arena.Bytes
1. documentation for the generated code

Second Release
1. arena map on top of linear hashing alg
1. instrumented arena
1. create additional methods for allocation within limits that can accept to sizes (minSize, preferableSize)
1. close arena function
1. arena leak detector
1. to ref pointers leak detector
1. thread safe arena registry:
    1. with whole registry allocation limit
    1. by type arena pools
    1. metrics  

Done:
1. Raw arena implementation
1. General arena wrapper with basic metrics
1. Support fetch current allocation offset
1. Preallocate arena buffer
1. Arena options
1. Wrap arenas into each other
1. Set allocation limits
1. Clear whole arena
1. Byte slice allocation options
    1. Capacity management
    1. Append function
    1. Separate hiding header that can be resolved to []byte
    1. Full slice copy to general heap option
    1. String cast option
    1. Copy to heap with to string cast
    1. Arena string allocation option from passed []byte
    1. Optimization of append to consecutive byte slices where we try to fit allocation in currently available buffer
1. Code generation - take into account the observability of specified structure
1. Option to clean a underlying arena during clear in Generic allocator
1. Make sure that Append works on top of "empty" slices
1. Tests with '-d=checkptr'
1. Make arena.Buffer.WriteString throw panic on allocation error to bo compatible with bytes.Buffer
