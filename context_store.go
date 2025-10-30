package mcpio

import "context"

// ContextStore defines the interface for storing and retrieving request context data.
// Implementations can customize the context key used for storage, enabling test isolation.
type ContextStore interface {
	// Store injects a RequestContext into the context
	Store(ctx context.Context, reqCtx *RequestContext) context.Context

	// Retrieve extracts a RequestContext from the context
	// Returns nil if no request context is available
	Retrieve(ctx context.Context) *RequestContext
}

// contextStore implements ContextStore with a configurable context key.
type contextStore struct {
	key any
}

// NewContextStore creates a ContextStore with the specified context key.
// The key is used to store and retrieve request context data from Go contexts.
//
// Most applications should use the default key:
//
//	store := mcpio.NewContextStore(mcpio.DefaultContextKey)
//
// Custom keys are rarely needed. Use them only when you need to maintain multiple
// independent context stores in the same application (e.g., for testing edge cases
// with isolated mock stores)
func NewContextStore(key any) ContextStore {
	return &contextStore{key: key}
}

// Store injects a RequestContext into the context using the configured key.
func (s *contextStore) Store(ctx context.Context, reqCtx *RequestContext) context.Context {
	return context.WithValue(ctx, s.key, reqCtx)
}

// Retrieve extracts a RequestContext from the context using the configured key.
// Returns nil if no request context is available.
func (s *contextStore) Retrieve(ctx context.Context) *RequestContext {
	if reqCtx, ok := ctx.Value(s.key).(*RequestContext); ok {
		return reqCtx
	}
	return nil
}

// DefaultContextKey is the default context key used for production request context storage.
// Export this so users can create their own stores with the same key if needed.
var DefaultContextKey = mcpContextKey{}
