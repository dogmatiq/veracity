package journal

import (
	"context"
)

// BinaryStore is a collection of journals.
//
// Each journal in the store is identified by a path, represented as a slice of
// string elements.
type BinaryStore interface {
	// Open returns the journal at the given path.
	//
	// The path uniquely identifies the journal. It must not be empty. Each
	// element must be a non-empty UTF-8 string consisting solely of printable
	// Unicode characters, excluding whitespace. A printable character is any
	// character from the Letter, Mark, Number, Punctuation or Symbol
	// categories.
	Open(ctx context.Context, path ...string) (BinaryJournal, error)
}

// Store is a database of journals.
//
// Each journal in the store is identified by a path, represented as a slice of
// string elements.
type Store[R any] interface {
	// Open returns the journal at the given path.
	//
	// The path uniquely identifies the journal. It must not be empty. Each
	// element must be a non-empty UTF-8 string consisting solely of printable
	// Unicode characters, excluding whitespace. A printable character is any
	// character from the Letter, Mark, Number, Punctuation or Symbol
	// categories.
	Open(ctx context.Context, path ...string) (BinaryJournal, error)
}
