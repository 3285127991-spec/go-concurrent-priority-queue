// Command demo is a small CLI walkthrough of the priorityqueue package.
package main

import (
	"errors"
	"fmt"

	"priorityqueue"
)

func main() {
	q, err := priorityqueue.New(4)
	if err != nil {
		panic(err)
	}
	fmt.Printf("created queue: capacity=%d len=%d\n\n", q.Capacity(), q.Len())

	fmt.Println("-- Push 4 elements (capacity is 4) --")
	push(q, "alpha", 3, "first")
	push(q, "bravo", 7, "second")
	push(q, "charlie", 7, "third")
	push(q, "delta", 1, "fourth")

	fmt.Println("\n-- Rejections: full queue and duplicate ID --")
	push(q, "echo", 9, "fifth")      // full
	push(q, "alpha", 9, "duplicate") // duplicate ID

	fmt.Println("\n-- UpdatePriority: charlie 7 -> 9, missing ID --")
	fmt.Println("update charlie ->", q.UpdatePriority("charlie", 9))
	fmt.Println("update ghost   ->", q.UpdatePriority("ghost", 1))

	fmt.Println("\n-- Get / Peek (no removal) --")
	if it, ok := q.Get("bravo"); ok {
		fmt.Printf("Get(bravo) = %+v\n", it)
	}
	if it, ok := q.Peek(); ok {
		fmt.Printf("Peek()     = %+v  (len still %d)\n", it, q.Len())
	}

	fmt.Println("\n-- Snapshot (dequeue order: priority desc, FIFO on ties) --")
	for i, it := range q.Snapshot() {
		fmt.Printf("  [%d] %+v\n", i, it)
	}

	fmt.Println("\n-- Remove bravo, then Pop everything --")
	fmt.Println("remove bravo ->", q.Remove("bravo"))
	fmt.Println("remove bravo ->", q.Remove("bravo"), "(already gone)")
	for {
		it, ok := q.Pop()
		if !ok {
			break
		}
		fmt.Printf("Pop() = %+v\n", it)
	}
	if _, ok := q.Pop(); !ok {
		fmt.Println("Pop() on empty queue -> ok=false")
	}
	fmt.Printf("\nfinal: len=%d capacity=%d\n", q.Len(), q.Capacity())
}

func push(q *priorityqueue.Queue, id string, priority int, value string) {
	err := q.Push(id, priority, value)
	switch {
	case err == nil:
		fmt.Printf("Push(%-8s p=%d) ok\n", id, priority)
	case errors.Is(err, priorityqueue.ErrQueueFull):
		fmt.Printf("Push(%-8s p=%d) rejected: queue full\n", id, priority)
	case errors.Is(err, priorityqueue.ErrDuplicateID):
		fmt.Printf("Push(%-8s p=%d) rejected: duplicate ID\n", id, priority)
	default:
		fmt.Printf("Push(%-8s p=%d) error: %v\n", id, priority, err)
	}
}
