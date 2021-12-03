package persistence

import (
	"context"
	"net/url"
	"strings"
)

// A KeyValueStore is a simple associative database that creates mappings
// between UTF-8 keys and opaque binary values.
//
// Keys are structured as Unix-like paths, including a leading slash. Each path
// component is a URL encoded UTF-8 string.
//
// The store MAY parse keys to structure data heirarchically, however two keys
// must only considered equivalent if they are equal byte-for-byte.
type KeyValueStore interface {
	// Set associates the value v with the key k.
	//
	// Any value already associated with k is replaced.
	//
	// Setting v to an empty (or nil) slice deletes any existing association.
	Set(ctx context.Context, k string, v []byte) error

	// Get returns the value associated with the key k.
	//
	// It returns an empty value if there is no value associated with k. There
	// is no distinction made between an empty slice and a nil slice.
	Get(ctx context.Context, k string) (v []byte, err error)
}

// Key returns a key containing slash-separated atoms. Each atom is URL encoded.
func Key(atoms ...string) string {
	var b strings.Builder

	for _, atom := range atoms {
		b.WriteRune('/')
		b.WriteString(url.PathEscape(atom))
	}

	return b.String()
}
