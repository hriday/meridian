// Package collector is Meridian's in-memory UDP collector framework: a bounded,
// sequenced ring plus a listener lifecycle shared by the syslog and trap
// receivers. It is ephemeral by design — nothing is written to disk.
package collector

import "sync"

// Stamped is a ring item tagged with its monotonic sequence number.
type Stamped[T any] struct {
	Seq int `json:"seq"`
	Val T   `json:"val"`
}

// Ring keeps the last `cap` items, each with a monotonic Seq. Safe for one
// writer (the read loop) and concurrent readers (Poll).
type Ring[T any] struct {
	mu      sync.Mutex
	cap     int
	buf     []Stamped[T]
	nextSeq int
	dropped int
}

// NewRing returns a ring holding at most capacity items.
func NewRing[T any](capacity int) *Ring[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring[T]{cap: capacity}
}

// Add appends v, assigning the next Seq and evicting the oldest past capacity.
func (r *Ring[T]) Add(v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextSeq++
	r.buf = append(r.buf, Stamped[T]{Seq: r.nextSeq, Val: v})
	if over := len(r.buf) - r.cap; over > 0 {
		r.buf = append(r.buf[:0], r.buf[over:]...)
		r.dropped += over
	}
}

// Snapshot copies the retained items with Seq > afterSeq, plus the current held
// count and the lifetime eviction count. The copy is made under the lock so the
// caller can marshal it without blocking the writer.
func (r *Ring[T]) Snapshot(afterSeq int) (items []Stamped[T], total, dropped int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, it := range r.buf {
		if it.Seq > afterSeq {
			items = append(items, it)
		}
	}
	return items, len(r.buf), r.dropped
}
