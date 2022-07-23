package aggregate

import "github.com/dogmatiq/configkit"

// Observer observes the lifecycle events of a supervisor.
type Observer interface {
	OnCommandReceived(h configkit.Identity, id string, cmd *Command)
	OnCommandDispatched(h configkit.Identity, id string, cmd *Command)
	OnCommandAck(h configkit.Identity, id string, cmd *Command)
	OnCommandNack(h configkit.Identity, id string, cmd *Command, reported, cause error)
	OnWorkerStart(h configkit.Identity, id string)
	OnWorkerLoaded(h configkit.Identity, id string)
	OnWorkerIdle(h configkit.Identity, id string)
	OnWorkerIdleAck(h configkit.Identity, id string)
	OnWorkerShutdown(h configkit.Identity, id string, err error)
}
