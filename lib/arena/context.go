package arena

import "context"

type allocatorCtxKey string

const arenaCtxKey allocatorCtxKey = "_arCtxK"

// WithAllocator allows you to bind ctx with target allocator
// and than receive it from ctx using GetAllocator and GetAllocatorOrDefault methods.
func WithAllocator(ctx context.Context, allocator Allocator) context.Context {
	return context.WithValue(ctx, arenaCtxKey, allocator)
}

// GetAllocator allows you to receive allocator associated with this ctx.
// Returns allocator and true if there was some association.
func GetAllocator(ctx context.Context) (Allocator, bool) {
	value := ctx.Value(arenaCtxKey)
	if value == nil {
		return nil, false
	}
	allocator, ok := value.(Allocator)
	if !ok {
		return nil, false
	}
	return allocator, true
}

// GetAllocatorOrDefault allows you to receive allocator associated with this ctx.
// Returns associated allocator or defaultAllocator if there were to association.
func GetAllocatorOrDefault(ctx context.Context, defaultAllocator Allocator) Allocator {
	ctxAllocator, ok := GetAllocator(ctx)
	if !ok {
		return defaultAllocator
	}
	return ctxAllocator
}
