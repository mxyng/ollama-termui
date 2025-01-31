package chat

import (
	"iter"
	"time"
)

type bucket struct {
	time  time.Time
	count int
}

type Metrics struct {
	buckets deque[bucket]
	round   time.Duration
}

func (m *Metrics) Add(createdAt time.Time, values ...int) {
	n := 1
	if len(values) > 0 {
		n = values[0]
	}

	truncated := createdAt.Truncate(m.round)
	if m.buckets.tail != nil && m.buckets.tail.t.time.Equal(truncated) {
		m.buckets.tail.t.count += n
		return
	}

	m.buckets.PushBack(bucket{time: truncated, count: n})
}

func (m *Metrics) After() iter.Seq[int] {
	return func(yield func(int) bool) {
		for cur := m.buckets.tail; cur != nil; cur = cur.prev {
			if !yield(cur.t.count) {
				break
			}
		}

		return
	}
}

func Reduce[T comparable](s iter.Seq[T], fn func(_, _ T) T) (t T) {
	next, stop := iter.Pull(s)
	defer stop()

	for v, ok := next(); ok; v, ok = next() {
		t = fn(t, v)
	}

	return t
}

func (m Metrics) Rate() (f float64) {
	if m.buckets.size > 1 {
		f := Reduce(m.After(), func(sum, v int) int {
			return sum + v
		})

		return float64(f) / m.buckets.tail.t.time.Sub(m.buckets.head.t.time).Seconds()
	}

	return 0
}

func (m *Metrics) Reset() {
	m.buckets = deque[bucket]{}
}
