package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type worker struct{ id int }

func TestFixedPool_TableDriven(t *testing.T) {
	type args struct {
		capacity uint
	}
	type want struct {
		newCountMin int
		newCountMax int
		expectBlock bool // for cases we assert a blocked Get
	}

	tests := []struct {
		name string
		args args
		// setup can seed the pool state before exercising behavior
		setup func(t *testing.T, p *fixed, newCount *int32) (extra any)
		// run performs the actions under test and returns observed values
		run  func(t *testing.T, p *fixed, extra any, newCount *int32) (gotCreated int, gotVals []any)
		want want
	}{
		{
			name: "constructor: capacity>0 makes buffered channels",
			args: args{capacity: 3},
			setup: func(_ *testing.T, _ *fixed, _ *int32) any {
				// nothing
				return nil
			},
			run: func(t *testing.T, p *fixed, _ any, _ *int32) (int, []any) {
				// Indirectly verify by pushing up to capacity into available without blocking.
				for i := 0; i < cap(p.available); i++ {
					select {
					case p.available <- &worker{id: i}:
					case <-time.After(100 * time.Millisecond):
						t.Fatalf("available channel did not accept up to capacity elements")
					}
				}
				// Drain to restore pristine state.
				var drained int
				for i := 0; i < cap(p.available); i++ {
					select {
					case <-p.available:
						drained++
					default:
					}
				}
				if drained != cap(p.available) {
					t.Fatalf("drained %d, want %d", drained, cap(p.available))
				}
				return 0, nil
			},
			want: want{newCountMin: 0, newCountMax: 0},
		},
		{
			name: "Get creates up to capacity via newFn; then blocks until Put",
			args: args{capacity: 2},
			run: func(t *testing.T, p *fixed, _ any, newCount *int32) (int, []any) {
				// First two Get() calls should create two workers.
				w1 := p.Get().(*worker)
				w2 := p.Get().(*worker)
				if w1 == nil || w2 == nil || w1 == w2 {
					t.Fatalf("expected two distinct workers, got %v and %v", w1, w2)
				}

				// Third Get should block until a Put occurs.
				gotCh := make(chan any, 1)
				go func() { gotCh <- p.Get() }()

				select {
				case <-gotCh:
					t.Fatalf("third Get should block until Put; returned early")
				case <-time.After(100 * time.Millisecond):
					// still blocked as expected
				}

				// Now Put one worker back; the blocked Get should return that same worker.
				p.Put(w1)

				select {
				case got := <-gotCh:
					if got != w1 {
						t.Fatalf("expected blocked Get to receive reused worker w1; got %v", got)
					}
				case <-time.After(200 * time.Millisecond):
					t.Fatalf("blocked Get did not resume after Put")
				}

				created := int(atomic.LoadInt32(newCount))
				return created, []any{w1, w2}
			},
			want: want{newCountMin: 2, newCountMax: 2},
		},
		{
			name: "Get reuses worker from available even if capacity not yet reached",
			args: args{capacity: 3},
			setup: func(_ *testing.T, p *fixed, _ *int32) any {
				// Seed an externally provided worker into available BEFORE any creation.
				p.available <- &worker{id: 42}
				return nil
			},
			run: func(t *testing.T, p *fixed, _ any, newCount *int32) (int, []any) {
				got := p.Get()
				if w, ok := got.(*worker); !ok || w.id != 42 {
					t.Fatalf("expected to reuse seeded worker id=42, got %#v", got)
				}
				created := int(atomic.LoadInt32(newCount))
				if created != 0 {
					t.Fatalf("expected no new worker creation, newCount=%d", created)
				}
				return created, []any{got}
			},
			want: want{newCountMin: 0, newCountMax: 0},
		},
		{
			name: "Put then Get returns the same instance",
			args: args{capacity: 1},
			run: func(t *testing.T, p *fixed, _ any, _ *int32) (int, []any) {
				w := p.Get()
				p.Put(w)
				w2 := p.Get()
				if w2 != w {
					t.Fatalf("expected same instance after Put/Get; got %v vs %v", w, w2)
				}
				return 1, []any{w, w2}
			},
			want: want{newCountMin: 1, newCountMax: 1},
		},
		{
			name: "Concurrent Get/Put never creates more than capacity workers",
			args: args{capacity: 5},
			run: func(t *testing.T, p *fixed, _ any, newCount *int32) (int, []any) {
				const goroutines = 20
				var wg sync.WaitGroup
				wg.Add(goroutines)

				for i := 0; i < goroutines; i++ {
					go func() {
						defer wg.Done()
						w := p.Get()
						// simulate a tiny bit of work
						time.Sleep(5 * time.Millisecond)
						p.Put(w)
					}()
				}
				wg.Wait()
				created := int(atomic.LoadInt32(newCount))
				if created > int(p.capacity()) {
					t.Fatalf("created %d workers, exceeds capacity %d", created, p.capacity())
				}
				return created, nil
			},
			want: want{newCountMin: 1, newCountMax: 5},
		},
		{
			name: "capacity=0: Get blocks (documented edge-case)",
			args: args{capacity: 0},
			run: func(t *testing.T, p *fixed, _ any, newCount *int32) (int, []any) {
				done := make(chan struct{})
				go func() {
					_ = p.Get() // will block forever
					close(done)
				}()
				select {
				case <-done:
					t.Fatalf("Get unexpectedly returned with capacity 0 (should block)")
				case <-time.After(100 * time.Millisecond):
					// expected: still blocked
				}
				created := int(atomic.LoadInt32(newCount))
				if created != 0 {
					t.Fatalf("newFn should not be called when cap=0; got %d", created)
				}
				return created, nil
			},
			want: want{newCountMin: 0, newCountMax: 0, expectBlock: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var counter int32
			newFn := func() interface{} {
				id := int(atomic.AddInt32(&counter, 1))
				return &worker{id: id}
			}

			p := NewFixed(tt.args.capacity, newFn).(*fixed)

			if tt.setup != nil {
				tt.setup(t, p, &counter)
			}

			created, _ := tt.run(t, p, nil, &counter)

			if created < tt.want.newCountMin || created > tt.want.newCountMax {
				t.Fatalf("newFn calls = %d, want in [%d..%d]", created, tt.want.newCountMin, tt.want.newCountMax)
			}
		})
	}
}

// capacity helper (not exported by fixed) — derive from channel buffer.
func (p *fixed) capacity() uint {
	return uint(cap(p.len))
}
