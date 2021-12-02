package aggregate

import (
	"context"

	"github.com/dogmatiq/dogma"
)

type TransactionLoader interface {
	LoadTransactionsSinceRevision(
		ctx context.Context,
		handlerKey, instanceID string,
		revision uint64,
	) (transactions []Transaction, done bool, err error)
}

type Transaction struct {
	Events    []dogma.Message
	Destroyed bool
}
