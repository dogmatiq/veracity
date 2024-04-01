package eventstream

import (
	"slices"
	"time"
)

const (
	// maxCacheAge is the maximum age of an event that will be retained in the
	// cache of recent events.
	maxCacheAge = 1 * time.Minute

	// maxCacheCapacity is the maximum number of events that will be retained in
	// the cache of recent events.
	maxCacheCapacity = 1000
)

// findInCache returns the index of the event with the given offset in the cache
// of recent events, or -1 if the event is not in the cache.
func (w *worker) findInCache(offset Offset) int {
	begin := w.nextOffset - Offset(len(w.recentEvents))

	if begin <= offset && offset < w.nextOffset {
		return int(offset - begin)
	}

	return -1
}

// growCache grows the cache capacity to fit an additional n events. It removes
// old events if necessary.
//
// It returns the number of events that may be added to the cache.
func (w *worker) growCache(n int) int {
	begin := 0
	end := len(w.recentEvents)

	if end > maxCacheCapacity {
		panic("cache is over capacity, always use appendToCache() to add events")
	}

	if n >= maxCacheCapacity {
		// We've requested the entire cache, so just clear it entirely.
		end = 0
		n = maxCacheCapacity
	} else {
		// Otherwise, first remove any events that are older than the cache TTL.
		for index, event := range w.recentEvents[begin:end] {
			createdAt := event.Envelope.CreatedAt.AsTime()

			if time.Since(createdAt) < maxCacheAge {
				begin += index
				break
			}
		}

		// Then, if we still don't have enough space, remove the oldest events.
		capacity := end - begin + n
		if capacity > maxCacheCapacity {
			begin += capacity - maxCacheCapacity
		}
	}

	// Note, the slice indices are computed without modifying the slice so that
	// we only perform a single copy operation.
	copy(w.recentEvents, w.recentEvents[begin:end])

	w.recentEvents = w.recentEvents[:end-begin]
	w.recentEvents = slices.Grow(w.recentEvents, n)

	return n
}

// appendEventToCache appends the given event to the cache of recent events.
func (w *worker) appendEventToCache(event Event) {
	if len(w.recentEvents) == cap(w.recentEvents) {
		panic("cache is at capacity, call purgeCache() before appending")
	}
	w.recentEvents = append(w.recentEvents, event)
}
