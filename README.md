# arena
Primitive arena allocator

TODO:
First Release scope
1. cover with tests "Wrong arena deref"
1. preallocate arena buffer
1. arena options
1. set allocation limits
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