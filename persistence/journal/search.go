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
) (offset uint64, rec []byte, ok bool, err error) {
	for {
		offset := begin>>1 + end>>1

		rec, ok, err := j.Get(ctx, offset)
		if err != nil {
			return 0, nil, false, err
		}
		if !ok {
			return 0, nil, false, fmt.Errorf("journal is corrupt: missing record at offset %d", offset)
		}

		n, err := cmp(ctx, rec)
		if err != nil {
			return 0, nil, false, err
		}

		if n < 0 {
			end = offset
		} else if n > 0 {
			begin = offset + 1
		} else {
			return offset, rec, true, nil
		}
	}
}
