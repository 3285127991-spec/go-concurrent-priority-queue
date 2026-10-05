package priorityqueue

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

func mustNew(t *testing.T, capacity int) *Queue {
	t.Helper()
	q, err := New(capacity)
	if err != nil {
		t.Fatalf("New(%d) returned error: %v", capacity, err)
	}
	return q
}

func mustPush(t *testing.T, q *Queue, id string, priority int, value string) {
	t.Helper()
	if err := q.Push(id, priority, value); err != nil {
		t.Fatalf("Push(%q) returned error: %v", id, err)
	}
}

func popIDs(t *testing.T, q *Queue) []string {
	t.Helper()
	var ids []string
	for {
		it, ok := q.Pop()
		if !ok {
			return ids
		}
		ids = append(ids, it.ID)
	}
}

func TestNewInvalidCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1, -100} {
		q, err := New(capacity)
		if !errors.Is(err, ErrInvalidCapacity) {
			t.Errorf("New(%d): got err=%v, want ErrInvalidCapacity", capacity, err)
		}
		if q != nil {
			t.Errorf("New(%d): got non-nil queue on error", capacity)
		}
	}
}

func TestBasicPushPopLenCapacity(t *testing.T) {
	q := mustNew(t, 3)
	if q.Len() != 0 || q.Capacity() != 3 {
		t.Fatalf("new queue: Len=%d Capacity=%d, want 0/3", q.Len(), q.Capacity())
	}
	mustPush(t, q, "a", 1, "va")
	mustPush(t, q, "b", 2, "vb")
	if q.Len() != 2 {
		t.Fatalf("Len=%d, want 2", q.Len())
	}
	it, ok := q.Pop()
	if !ok || it.ID != "b" || it.Priority != 2 || it.Value != "vb" {
		t.Fatalf("Pop() = %+v, %v; want b/2/vb", it, ok)
	}
	if q.Len() != 1 {
		t.Fatalf("Len=%d after Pop, want 1", q.Len())
	}
}

func TestPopPeekEmptyQueue(t *testing.T) {
	q := mustNew(t, 1)
	if _, ok := q.Pop(); ok {
		t.Error("Pop on empty queue: got ok=true, want false")
	}
	if _, ok := q.Peek(); ok {
		t.Error("Peek on empty queue: got ok=true, want false")
	}
}

func TestPriorityOrderHighToLow(t *testing.T) {
	q := mustNew(t, 10)
	mustPush(t, q, "low", 1, "")
	mustPush(t, q, "mid", 5, "")
	mustPush(t, q, "high", 9, "")
	mustPush(t, q, "neg", -3, "")
	mustPush(t, q, "mid2", 4, "")
	want := []string{"high", "mid", "mid2", "low", "neg"}
	if got := popIDs(t, q); !reflect.DeepEqual(got, want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestSamePriorityFIFO(t *testing.T) {
	q := mustNew(t, 10)
	mustPush(t, q, "first", 5, "")
	mustPush(t, q, "second", 5, "")
	mustPush(t, q, "third", 5, "")
	mustPush(t, q, "fourth", 5, "")
	want := []string{"first", "second", "third", "fourth"}
	if got := popIDs(t, q); !reflect.DeepEqual(got, want) {
		t.Fatalf("pop order = %v, want FIFO %v", got, want)
	}
}

func TestUpdatePriorityKeepsFIFOOrder(t *testing.T) {
	q := mustNew(t, 10)
	mustPush(t, q, "a", 5, "")
	mustPush(t, q, "b", 5, "")
	mustPush(t, q, "c", 5, "")
	// Re-setting the same priority must not reshuffle FIFO order.
	if !q.UpdatePriority("b", 5) {
		t.Fatal("UpdatePriority(b) = false, want true")
	}
	// Raising then restoring a priority must not change the original seq.
	if !q.UpdatePriority("a", 9) {
		t.Fatal("UpdatePriority(a) = false, want true")
	}
	if !q.UpdatePriority("a", 5) {
		t.Fatal("UpdatePriority(a) = false, want true")
	}
	mustPush(t, q, "d", 5, "")
	want := []string{"a", "b", "c", "d"}
	if got := popIDs(t, q); !reflect.DeepEqual(got, want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestUpdatePriorityReorders(t *testing.T) {
	q := mustNew(t, 10)
	mustPush(t, q, "a", 1, "")
	mustPush(t, q, "b", 2, "")
	mustPush(t, q, "c", 3, "")
	if !q.UpdatePriority("a", 10) {
		t.Fatal("UpdatePriority(a) = false, want true")
	}
	if got, _ := q.Get("a"); got.Priority != 10 {
		t.Fatalf("Get(a).Priority = %d, want 10", got.Priority)
	}
	want := []string{"a", "c", "b"}
	if got := popIDs(t, q); !reflect.DeepEqual(got, want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestUpdatePriorityMissing(t *testing.T) {
	q := mustNew(t, 2)
	if q.UpdatePriority("ghost", 7) {
		t.Error("UpdatePriority on missing ID = true, want false")
	}
	mustPush(t, q, "a", 1, "")
	if q.UpdatePriority("ghost", 7) {
		t.Error("UpdatePriority on missing ID = true, want false")
	}
	if q.Len() != 1 {
		t.Errorf("Len=%d, want 1 (failed update must not mutate)", q.Len())
	}
}

func TestDuplicateIDRejected(t *testing.T) {
	q := mustNew(t, 5)
	mustPush(t, q, "a", 1, "orig")
	err := q.Push("a", 9, "clobber")
	if !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("Push duplicate: got err=%v, want ErrDuplicateID", err)
	}
	it, ok := q.Get("a")
	if !ok || it.Priority != 1 || it.Value != "orig" {
		t.Fatalf("Get(a) = %+v,%v; duplicate push must not modify existing", it, ok)
	}
	if q.Len() != 1 {
		t.Fatalf("Len=%d, want 1", q.Len())
	}
}

func TestCapacityBoundary(t *testing.T) {
	q := mustNew(t, 2)
	mustPush(t, q, "a", 1, "")
	mustPush(t, q, "b", 2, "")
	if err := q.Push("c", 3, ""); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("Push on full queue: got err=%v, want ErrQueueFull", err)
	}
	if q.Len() != 2 {
		t.Fatalf("Len=%d, want 2 (rejected push must not grow queue)", q.Len())
	}
	// Non-insert operations still work on a full queue.
	if !q.UpdatePriority("a", 100) {
		t.Error("UpdatePriority on full queue failed")
	}
	if !q.Remove("a") {
		t.Error("Remove on full queue failed")
	}
	// After removals there is room again.
	mustPush(t, q, "c", 3, "")
	if _, ok := q.Pop(); !ok {
		t.Fatal("Pop failed")
	}
	mustPush(t, q, "d", 4, "")
	if err := q.Push("e", 5, ""); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("Push on refilled queue: got err=%v, want ErrQueueFull", err)
	}
}

func TestEmptyIDRejected(t *testing.T) {
	q := mustNew(t, 1)
	if err := q.Push("", 1, "v"); !errors.Is(err, ErrEmptyID) {
		t.Fatalf("Push empty ID: got err=%v, want ErrEmptyID", err)
	}
	if q.Len() != 0 {
		t.Fatalf("Len=%d, want 0", q.Len())
	}
	// Empty-ID validation happens even when the queue is full.
	mustPush(t, q, "a", 1, "")
	if err := q.Push("", 1, "v"); !errors.Is(err, ErrEmptyID) {
		t.Fatalf("Push empty ID on full queue: got err=%v, want ErrEmptyID", err)
	}
}

func TestRemove(t *testing.T) {
	q := mustNew(t, 3)
	if q.Remove("ghost") {
		t.Error("Remove on empty queue = true, want false")
	}
	mustPush(t, q, "a", 1, "")
	mustPush(t, q, "b", 2, "")
	mustPush(t, q, "c", 3, "")
	if !q.Remove("b") {
		t.Fatal("Remove(b) = false, want true")
	}
	if q.Len() != 2 {
		t.Fatalf("Len=%d, want 2", q.Len())
	}
	if _, ok := q.Get("b"); ok {
		t.Error("Get(b) after Remove: ok=true, want false")
	}
	if q.Remove("b") {
		t.Error("Remove(b) twice = true, want false")
	}
	// The ID is free for reuse after removal.
	mustPush(t, q, "b", 9, "")
	want := []string{"b", "c", "a"}
	if got := popIDs(t, q); !reflect.DeepEqual(got, want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestPeekDoesNotRemove(t *testing.T) {
	q := mustNew(t, 3)
	mustPush(t, q, "a", 1, "va")
	mustPush(t, q, "b", 2, "vb")
	for i := 0; i < 3; i++ {
		it, ok := q.Peek()
		if !ok || it.ID != "b" || it.Value != "vb" {
			t.Fatalf("Peek #%d = %+v,%v; want b/vb", i, it, ok)
		}
	}
	if q.Len() != 2 {
		t.Fatalf("Len=%d after Peeks, want 2 (Peek must not remove)", q.Len())
	}
}

func TestGet(t *testing.T) {
	q := mustNew(t, 2)
	if _, ok := q.Get("nope"); ok {
		t.Error("Get on empty queue: ok=true, want false")
	}
	mustPush(t, q, "a", 7, "va")
	it, ok := q.Get("a")
	if !ok || it != (Item{ID: "a", Priority: 7, Value: "va"}) {
		t.Fatalf("Get(a) = %+v,%v", it, ok)
	}
	if q.Len() != 1 {
		t.Fatalf("Len=%d after Get, want 1 (Get must not remove)", q.Len())
	}
}

func TestSnapshotStableOrder(t *testing.T) {
	q := mustNew(t, 10)
	mustPush(t, q, "p3a", 3, "")
	mustPush(t, q, "p9", 9, "")
	mustPush(t, q, "p3b", 3, "")
	mustPush(t, q, "p5", 5, "")
	mustPush(t, q, "p3c", 3, "")
	want := []string{"p9", "p5", "p3a", "p3b", "p3c"}

	snap1 := q.Snapshot()
	var got []string
	for _, it := range snap1 {
		got = append(got, it.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Snapshot order = %v, want %v", got, want)
	}
	// Snapshot is repeatable and does not drain the queue.
	if snap2 := q.Snapshot(); !reflect.DeepEqual(snap1, snap2) {
		t.Fatalf("two Snapshots differ: %v vs %v", snap1, snap2)
	}
	if q.Len() != 5 {
		t.Fatalf("Len=%d after Snapshot, want 5", q.Len())
	}
	// Snapshot order matches actual dequeue order.
	if got := popIDs(t, q); !reflect.DeepEqual(got, want) {
		t.Fatalf("pop order = %v, want %v", got, want)
	}
}

func TestSnapshotEmpty(t *testing.T) {
	q := mustNew(t, 1)
	snap := q.Snapshot()
	if len(snap) != 0 {
		t.Fatalf("Snapshot on empty queue has %d items, want 0", len(snap))
	}
}

func TestReturnedCopiesAreIsolated(t *testing.T) {
	q := mustNew(t, 3)
	mustPush(t, q, "a", 1, "orig")
	mustPush(t, q, "b", 2, "keep")

	// Mutating a Get result must not affect the queue.
	it, _ := q.Get("a")
	it.Priority = 999
	it.Value = "hacked"
	it.ID = "zzz"
	fresh, _ := q.Get("a")
	if fresh != (Item{ID: "a", Priority: 1, Value: "orig"}) {
		t.Errorf("Get result mutation leaked into queue: %+v", fresh)
	}

	// Mutating a Peek result must not affect the queue.
	peeked, _ := q.Peek()
	peeked.Priority = -100
	peeked.Value = "hacked"
	again, _ := q.Peek()
	if again != (Item{ID: "b", Priority: 2, Value: "keep"}) {
		t.Errorf("Peek result mutation leaked into queue: %+v", again)
	}

	// Mutating Snapshot contents must not affect the queue.
	snap := q.Snapshot()
	snap[0].Priority = -100
	snap[0].Value = "hacked"
	snap[1].ID = "zzz"
	snap = append(snap, Item{ID: "extra", Priority: 1000})
	_ = snap
	if got := popIDs(t, q); !reflect.DeepEqual(got, []string{"b", "a"}) {
		t.Errorf("Snapshot mutation leaked into queue, pop order = %v", got)
	}
}

func TestInterleavedPushPopStress(t *testing.T) {
	q := mustNew(t, 50)
	rng := rand.New(rand.NewSource(42))
	var wantMax []int // naive reference model
	for i := 0; i < 5000; i++ {
		if rng.Intn(2) == 0 || len(wantMax) == 0 {
			id := fmt.Sprintf("id-%d", i)
			p := rng.Intn(100)
			if err := q.Push(id, p, ""); err == nil {
				wantMax = append(wantMax, p)
			}
		} else {
			it, ok := q.Pop()
			if !ok {
				t.Fatal("Pop failed while model non-empty")
			}
			// Model: remove the max; ties are irrelevant for priority check.
			maxIdx := 0
			for j, p := range wantMax {
				if p > wantMax[maxIdx] {
					maxIdx = j
				}
			}
			if it.Priority != wantMax[maxIdx] {
				t.Fatalf("step %d: Pop priority=%d, want %d", i, it.Priority, wantMax[maxIdx])
			}
			wantMax = append(wantMax[:maxIdx], wantMax[maxIdx+1:]...)
		}
	}
}
