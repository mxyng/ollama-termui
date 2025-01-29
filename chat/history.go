package chat

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

type node[T any] struct {
	t    T
	next *node[T]
	prev *node[T]
}

type deque[T any] struct {
	head, tail *node[T]

	size int
}

func (d *deque[T]) PushBack(t T) {
	newNode := &node[T]{t: t}
	if d.tail == nil {
		d.head = newNode
		d.tail = newNode
	} else {
		newNode.prev = d.tail
		d.tail.next = newNode
		d.tail = newNode
	}
	d.size++
}

func (d *deque[T]) PopFront() (T, bool) {
	if d.head == nil {
		var zero T
		return zero, false
	}

	val := d.head.t
	d.head = d.head.next
	if d.head != nil {
		d.head.prev = nil
	} else {
		d.tail = nil
	}
	d.size--
	return val, true
}

type History struct {
	deque[string]
	cur        *node[string]
	maxSize    int
	saveOnPush bool
}

func LoadFromFile(maxSize int, saveOnPush bool) History {
	h := History{maxSize: maxSize, saveOnPush: saveOnPush}

	home, err := os.UserHomeDir()
	if err != nil {
		return h
	}

	f, err := os.OpenFile(filepath.Join(home, ".ollama", "history"), os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return h
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		h.Push(scanner.Text())
	}

	return h
}

func SaveToFile(h History) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	f, err := os.OpenFile(filepath.Join(home, ".ollama", "history"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	for i := h.head; i != nil; i = i.next {
		fmt.Fprintln(w, i.t)
	}
}

func (h *History) NextLine() string {
	if h.cur == nil {
		return ""
	} else if h.cur.next == nil {
		h.cur = nil
		return ""
	}

	h.cur = h.cur.next
	return h.cur.t
}

func (h *History) PreviousLine() string {
	if h.cur == nil {
		h.cur = h.tail
		return h.cur.t
	} else if h.cur.prev != nil {
		h.cur = h.cur.prev
		return h.cur.t
	}

	return ""
}

func (h *History) Push(s string) {
	if h.tail == nil || h.tail.t != s {
		h.PushBack(s)
		if h.size > h.maxSize {
			h.PopFront()
		}

		if h.saveOnPush {
			SaveToFile(*h)
		}

		h.cur = nil
	}
}
