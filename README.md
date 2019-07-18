# arena
Primitive arena allocator

TODO:
First Release scope
1. remove notion of offset from all arenas
1. bind arena to context.Context (with leak detector in future)
1. whole documentation with notion of unsafe semantics

Second Release
1. instrumented arena
1. create additional methods for allocation within limits that can accept to sizes (minSize, preferableSize).
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


Test
```
make test
```