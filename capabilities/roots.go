package capabilities

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListRoots queries the client's workspace roots (directories, files, etc.).
// Servers can use this to discover what the client has access to.
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
