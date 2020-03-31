module github.com/storozhukBM/allocator

go 1.14

require (
	github.com/storozhukBM/allocator/generator v0.0.0-00010101000000-000000000000 // indirect
	github.com/storozhukBM/allocator/lib/arena v0.0.0-00010101000000-000000000000 // indirect
	make v0.0.0 // indirect
)

replace github.com/storozhukBM/allocator/lib/arena => ./lib/arena/

replace github.com/storozhukBM/allocator/generator => ./generator

replace make => ./cmd/internal
