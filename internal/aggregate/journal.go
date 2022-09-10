package aggregate

import "fmt"

// journalKey returns the journal key to use for a specific aggregate instance.
func journalKey(hk, id string) string {
	return fmt.Sprintf("aggregate/%s/%s", hk, id)
}
