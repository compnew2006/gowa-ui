package whatsmeow

import (
	"context"
	"fmt"

	"github.com/compnew2006/whatomate/internal/queue"
)

// ProcessInboundMediaJob executes async inbound-media recovery for a queued job.
func (a *WhatsmeowAdapter) ProcessInboundMediaJob(ctx context.Context, job *queue.InboundMediaJob) error {
	if a == nil || a.manager == nil {
		return fmt.Errorf("whatsmeow adapter manager is not initialized")
	}
	return a.manager.ProcessInboundMediaRecoveryJob(ctx, job)
}
