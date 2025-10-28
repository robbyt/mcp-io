package capabilities

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NotifyProgress sends a progress update for long-running operations.
// Progress should be between 0.0 and total, where total represents completion.
func (s *Session) NotifyProgress(ctx context.Context, progress, total float64) error {
	params := &mcp.ProgressNotificationParams{
		Progress: progress,
		Total:    total,
	}
	return s.session.NotifyProgress(ctx, params)
}
