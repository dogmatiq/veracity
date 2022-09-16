package pathkey

import "strings"

// New returns a string key derived from a path.
func New(path []string) string {
	if len(path) == 0 {
		panic("path must not be empty")
	}

	var w strings.Builder

	for _, elem := range path {
		if len(elem) == 0 {
			panic("path element must not be empty")
		}

		if w.Len() > 0 {
			w.WriteByte('/')
		}

		for _, r := range elem {
			if r == '/' || r == '\\' {
				w.WriteByte('\\')
			}

			w.WriteRune(r)
		}
	}

	return w.String()
}
