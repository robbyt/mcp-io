package capabilities

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Root represents a filesystem root that defines the boundaries of where servers can operate.
// The URI must be a file:// URI pointing to an accessible filesystem location.
type Root struct {
	URI  string // file:// URI identifying the root location
	Name string // Optional human-readable name for display purposes
}

// SupportsRoots returns true if the client supports filesystem roots with change notifications.
//
// NOTE: Due to a Go SDK design limitation, this only detects clients that support
// both roots AND change notifications (ListChanged=true). Clients that support roots
// without notifications cannot be distinguished from clients with no roots support.
//
// If this returns false, you can still try calling ListRoots() - it may work if the
// client supports roots but not change notifications. Handle the error appropriately.
//
// See: https://github.com/modelcontextprotocol/go-sdk/issues/607
func (s *sessionCapability) SupportsRoots() bool {
	return s.session.InitializeParams().Capabilities.Roots.ListChanged
}

// ListRoots retrieves the filesystem roots exposed by the client.
// Roots define the boundaries of where servers can operate within the filesystem,
// allowing servers to understand which directories and files they have access to.
func (s *sessionCapability) ListRoots(ctx context.Context) ([]*Root, error) {
	params := &mcp.ListRootsParams{}
	result, err := s.session.ListRoots(ctx, params)
	if err != nil {
		return nil, err
	}

	roots := make([]*Root, len(result.Roots))
	for i, r := range result.Roots {
		roots[i] = &Root{
			URI:  r.URI,
			Name: r.Name,
		}
	}
	return roots, nil
}
