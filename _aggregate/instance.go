package aggregate

import (
	"github.com/dogmatiq/dogma"
)

type Logger interface {
	Log(f string, v ...interface{})
}

type Instance struct {
	ID      string
	Handler dogma.AggregateMessageHandler
	Root    dogma.AggregateRoot
	Logger  Logger
}

func (i *Instance) HandleCommand(m dogma.Message) Transaction {
	sc := &scope{
		instanceID: i.ID,
		root:       i.Root,
		logger:     i.Logger,
	}

	i.Handler.HandleCommand(sc.root, sc, m)

	if sc.tx.Destroyed {
		i.Root = i.Handler.New()
	}

	return sc.tx
}
