package journal

import (
	"context"
	"fmt"
)

// Search performs a binary search to find the record for which cmp() returns 0.
func Search(
	ctx context.Context,
	j Journal,
	begin, end uint64,
	cmp func(ctx context.Context, rec []byte) (int, error),
) (ver uint64, rec []byte, ok bool, err error) {
	for {
		ver := begin>>1 + end>>1

		rec, ok, err := j.Get(ctx, ver)
		if err != nil {
			return 0, nil, false, err
		}
		if !ok {
			return 0, nil, false, fmt.Errorf("journal is corrupt: missing version %d", ver)
		}

		n, err := cmp(ctx, rec)
		if err != nil {
			return 0, nil, false, err
		}

		if n < 0 {
			end = ver
		} else if n > 0 {
			begin = ver + 1
		} else {
			return ver, rec, true, nil
		}
	}
}
