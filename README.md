# arena
Primitive arena allocator

TODO:
First Release scope
1. clear arenas to certain point
1. bind arena to context.Context (with leak detector in future)
1. byte slice allocation options
    1. capacity management
    1. append function
    1. separate hiding header that can be resolved to []byte
    1. full slice copy to general heap option
    1. string cast option
    1. copy to heap with to string cast
1. arena string allocation option from passed []byte
1. whole documentation with notion of unsafe semantics

Second Release
1. clear whole arena
1. instrumented arena
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


Build
```
go build ./...
```

Build with info
```
go build -gcflags -m ./...
```

Test
```
go test -v -race ./...
```

Coverage
```
go test -coverpkg=./... -coverprofile=coverage.out ./lib/arena/arena_test && go tool cover -html=coverage.out
```