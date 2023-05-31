package instrumentedpersistence

import (
	"fmt"
	"sync/atomic"

	"github.com/google/uuid"
)

var handleCounter atomic.Uint64

// handleID returns a unique identifier for an open instance of a journal or
// keyspace.
//
// It includes a counter component for easy visual identification by humans, and
// a UUID component for global correlation in observability tools.
func handleID() string {
	return fmt.Sprintf(
		"#%d %s",
		handleCounter.Add(1),
		uuid.NewString(),
	)
}
