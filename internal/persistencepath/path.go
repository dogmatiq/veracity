package persistencepath

import "strings"

// Join returns returns a path from the given elements.
func Join(elements []string) string {
	if len(elements) == 0 {
		panic("path must not be empty")
	}

	var w strings.Builder

	for _, elem := range elements {
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
