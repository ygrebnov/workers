package workers

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func assertEqualInts(t *testing.T, got, want []int) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected results: got=%v want=%v", got, want)
	}
}

func assertEmptyInts(t *testing.T, got []int) {
	t.Helper()
	if len(got) != 0 {
		t.Fatalf("expected empty results, got=%v", got)
	}
}

func ev[R any](idx int, val R, present bool) completionEvent[R] {
	return completionEvent[R]{idx: idx, val: val, present: present}
}

func runReorderer[R any](t *testing.T, events []completionEvent[R], resultsCap int) (results []R, closeFn func()) {
	t.Helper()
	eCh := make(chan completionEvent[R], len(events))
	rCh := make(chan R, resultsCap)

	r := newReorderer[R](eCh, rCh)
	done := make(chan struct{})
	go func() {
		r.run(context.Background())
		close(done)
	}()

	// feed events
	for _, e := range events {
		eCh <- e
	}
	close(eCh)

	// wait for reorderer to finish
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("reorderer did not finish in time")
	}

	// harvest results known to have been produced
	out := make([]R, 0, resultsCap)
	for i := 0; i < resultsCap; i++ {
		select {
		case v := <-rCh:
			out = append(out, v)
		default:
			return out, func() { close(rCh) }
		}
	}
	return out, func() { close(rCh) }
}

func TestReorderer_InOrder(t *testing.T) {
	res, cleanup := runReorderer[int](t, []completionEvent[int]{
		ev(0, 1, true),
		ev(1, 2, true),
	}, 4)
	defer cleanup()
	assertEqualInts(t, res, []int{1, 2})
}

func TestReorderer_OutOfOrder_BufferThenFlush(t *testing.T) {
	res, cleanup := runReorderer[int](t, []completionEvent[int]{
		ev(1, 2, true), // buffered first
		ev(0, 1, true), // unlocks 0 then 1
	}, 4)
	defer cleanup()
	assertEqualInts(t, res, []int{1, 2})
}

func TestReorderer_NoResultAdvances(t *testing.T) {
	res, cleanup := runReorderer[int](t, []completionEvent[int]{
		ev(0, 10, true), // emits 10
		ev(2, 20, true), // buffered (waiting for idx1)
		ev(1, 0, false), // advances cursor, unlocks 20
	}, 4)
	defer cleanup()
	assertEqualInts(t, res, []int{10, 20})
}

func TestReorderer_ShutdownFlushContiguousOnly(t *testing.T) {
	res, cleanup := runReorderer[int](t, []completionEvent[int]{
		// only idx1 arrives; idx0 missing, so nothing should be emitted
		ev(1, 2, true),
	}, 4)
	defer cleanup()
	assertEmptyInts(t, res)
}

func TestReorderer_MultipleNoResultInARow(t *testing.T) {
	res, cleanup := runReorderer[int](t, []completionEvent[int]{
		ev(0, 0, false), // advance 0
		ev(1, 0, false), // advance 1
		ev(2, 3, true),  // should emit now
	}, 4)
	defer cleanup()
	assertEqualInts(t, res, []int{3})
}
