package aggregate

import (
	"context"
	"fmt"

	envelopespec "github.com/dogmatiq/interopspec/envelopespec"
)

type CommandExecutor struct {
	Supervisors map[string]*Supervisor
}

func (e *CommandExecutor) ExecuteCommand(
	ctx context.Context,
	hk, id string,
	env *envelopespec.Envelope,
) error {
	if sup, ok := e.Supervisors[hk]; ok {
		return sup.ExecuteCommand(ctx, id, env)
	}

	return fmt.Errorf("unrecognized handler key: %s", hk)
}
