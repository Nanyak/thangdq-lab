package embed

import (
	"context"
	"strings"
)

// Router sends video/* jobs to heavy (SQS/Lambda) and everything else to light (Redis worker).
type Router struct {
	light Publisher
	heavy Publisher
}

func NewRouter(light, heavy Publisher) *Router {
	return &Router{light: light, heavy: heavy}
}

func (r *Router) Publish(ctx context.Context, job Job) error {
	if strings.HasPrefix(job.MimeType, "video/") {
		return r.heavy.Publish(ctx, job)
	}
	return r.light.Publish(ctx, job)
}
