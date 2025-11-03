package capabilities

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NotifyProgress sends a progress update for long-running operations.
// Progress should be between 0.0 and total, where total represents completion.
// Use functional options to specify additional fields like progress token and message.
//
// Basic usage (no options):
//
//	session.NotifyProgress(ctx, 5.0, 10.0)
//
// With progress token for request-specific tracking:
//
//	session.NotifyProgress(ctx, 5.0, 10.0,
//	    WithProgressToken(reqCtx.GetProgressToken()))
//
// With token and descriptive message:
//
//	session.NotifyProgress(ctx, 5.0, 10.0,
//	    WithProgressToken(token),
//	    WithProgressMessage("Processing file 5 of 10"))
func (s *Session) NotifyProgress(ctx context.Context, progress, total float64, opts ...ProgressOption) error {
	cfg := &progressConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	params := &mcp.ProgressNotificationParams{
		Progress: progress,
		Total:    total,
	}

	if cfg.token != nil {
		params.ProgressToken = cfg.token
	}
	if cfg.message != "" {
		params.Message = cfg.message
	}
	if cfg.meta != nil {
		params.Meta = cfg.meta
	}

	return s.session.NotifyProgress(ctx, params)
}
