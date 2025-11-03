package capabilities

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NotifyProgress sends a progress update for long-running operations.
// Progress should be between 0.0 and total, where total represents completion.
//
// Basic usage:
//
//	session.NotifyProgress(ctx, 5, 10)
//
// With options:
//
//	session.NotifyProgress(ctx, 5, 10,
//	    WithProgressToken(reqCtx.GetProgressToken()),
//	    WithProgressMessage("Processing file 5 of 10"))
//
// See README.md Progress Notifications section for complete examples.
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
