package service

import (
	"context"

	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

// ActionProcessor represents a modular action handler that can inspect a ZITADEL request
// and execute its specific business logic (e.g. flattening roles, enriching claims, validating requests).
type ActionProcessor interface {
	Name() string
	ShouldProcess(req *model.ActionRequest) bool
	Process(ctx context.Context, req *model.ActionRequest, opts model.FlattenOptions) (*model.ActionResponse, error)
}

// Dispatcher coordinates multiple ActionProcessors, merges their claim outputs,
// and returns a single unified ZITADEL Actions V2 response.
type Dispatcher struct {
	processors []ActionProcessor
}

// NewDispatcher creates a new ActionDispatcher.
func NewDispatcher(processors ...ActionProcessor) *Dispatcher {
	return &Dispatcher{
		processors: processors,
	}
}

// Register adds a new processor to the dispatcher.
func (d *Dispatcher) Register(processor ActionProcessor) {
	d.processors = append(d.processors, processor)
}

// Processors returns the list of registered processors.
func (d *Dispatcher) Processors() []ActionProcessor {
	return d.processors
}

// Dispatch executes all applicable processors for the given request and merges their responses.
func (d *Dispatcher) Dispatch(ctx context.Context, req *model.ActionRequest, opts model.FlattenOptions) (*model.ActionResponse, error) {
	merged := &model.ActionResponse{
		AppendClaims:    make([]model.ClaimItem, 0),
		AppendLogClaims: make([]string, 0),
		Claims:          make(map[string]any),
	}

	for _, p := range d.processors {
		if req != nil && !p.ShouldProcess(req) {
			continue
		}

		res, err := p.Process(ctx, req, opts)
		if err != nil {
			return nil, err
		}
		if res == nil {
			continue
		}

		// Merge append_claims
		merged.AppendClaims = append(merged.AppendClaims, res.AppendClaims...)
		merged.AppendLogClaims = append(merged.AppendLogClaims, res.AppendLogClaims...)

		// Merge groups convenience field if present
		if len(res.Groups) > 0 {
			merged.Groups = res.Groups
		}

		// Merge claims map
		for k, v := range res.Claims {
			merged.Claims[k] = v
		}
	}

	return merged, nil
}
