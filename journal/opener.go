package journal

import (
	"context"
)

// BinaryOpener is a journal opener that opens journals that contain opaque
// binary records.
type BinaryOpener = Opener[[]byte]

// Opener is an interface for accessing journals.
type Opener[R any] interface {
	// Open returns the journal at the given path.
	//
	// The path uniquely identifies the journal. It must not be empty. Each
	// element must be a non-empty UTF-8 string consisting solely of printable
	// Unicode characters, excluding whitespace. A printable character is any
	// character from the Letter, Mark, Number, Punctuation or Symbol
	// categories.
	Open(ctx context.Context, path ...string) (Journal[R], error)
}
