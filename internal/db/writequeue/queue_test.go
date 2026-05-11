package writequeue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWriteQueue_SerializesConcurrentProducers spawns N goroutines that
// each submit a marker WriteOp; the consumer must execute them strictly
// one-at-a-time. The recorder mutex panics if a second op enters while
// another is running, which directly proves serialization.
func TestWriteQueue_SerializesConcurrentProducers(t *testing.T) {
	const producers = 16
	const opsPerProducer = 25

	var inFlight atomic.Int32
	var maxConcurrent atomic.Int32
	var executed atomic.Int64

	op := func(_ context.Context) error {
		now := inFlight.Add(1)
		// Track the high-water mark so the test message reads cleanly
		// when serialization breaks.
		for {
			prev := maxConcurrent.Load()
			if now <= prev || maxConcurrent.CompareAndSwap(prev, now) {
				break
			}
		}
		time.Sleep(200 * time.Microsecond)
		inFlight.Add(-1)
		executed.Add(1)
		return nil
	}

	q := New(Config{Capacity: producers * opsPerProducer})
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var wg sync.WaitGroup
	for p := 0; p < producers; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerProducer; i++ {
				if err := q.Submit(context.Background(), op); err != nil {
					t.Errorf("Submit: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	q.Stop(5 * time.Second)

	if got := executed.Load(); got != producers*opsPerProducer {
		t.Errorf("executed = %d, want %d", got, producers*opsPerProducer)
	}
	if peak := maxConcurrent.Load(); peak > 1 {
		t.Errorf("max concurrent in-flight ops = %d, want 1 (single-writer invariant)", peak)
	}
}

// TestWriteQueue_BoundedBackpressure fills the queue to capacity then
// asserts the next Submit returns ErrQueueFull without blocking. The
// consumer is started AFTER the fill so it cannot drain during the fill
// loop (the channel buffer is what we are exercising).
func TestWriteQueue_BoundedBackpressure(t *testing.T) {
	const capacity = 4
	q := New(Config{Capacity: capacity})
	// Build a queue that has not been started — Submit should return
	// ErrWriterUnavailable for a clean before-and-after comparison.
	if err := q.Submit(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrWriterUnavailable) {
		t.Fatalf("pre-start Submit error = %v, want ErrWriterUnavailable", err)
	}

	// Start, then immediately block the consumer with a permanent op so
	// the channel buffer is the only thing absorbing producer submits.
	started := make(chan struct{})
	blockingDone := make(chan struct{})
	blocker := func(ctx context.Context) error {
		close(started)
		<-blockingDone
		return nil
	}
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		close(blockingDone)
		q.Stop(5 * time.Second)
	}()
	if err := q.Submit(context.Background(), blocker); err != nil {
		t.Fatalf("Submit blocker: %v", err)
	}
	// Wait for the consumer to pick the blocker off the channel so the
	// channel buffer is purely producer-visible.
	<-started

	noop := func(context.Context) error { return nil }
	for i := 0; i < capacity; i++ {
		if err := q.Submit(context.Background(), noop); err != nil {
			t.Fatalf("Submit %d: %v", i, err)
		}
	}
	err := q.Submit(context.Background(), noop)
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("overflow Submit error = %v, want ErrQueueFull", err)
	}

	stats := q.Stats()
	if stats.Rejected == 0 {
		t.Errorf("Stats.Rejected = 0, want > 0 after overflow")
	}
}

// TestWriteQueue_TimeoutReturnsError exercises SubmitWithTimeout: a
// permanently-blocked consumer plus a full buffer means the timeout
// branch wins.
func TestWriteQueue_TimeoutReturnsError(t *testing.T) {
	const capacity = 2
	q := New(Config{Capacity: capacity})

	started := make(chan struct{})
	blockingDone := make(chan struct{})
	defer close(blockingDone)
	blocker := func(context.Context) error {
		close(started)
		<-blockingDone
		return nil
	}
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer q.Stop(time.Second)

	if err := q.Submit(context.Background(), blocker); err != nil {
		t.Fatalf("Submit blocker: %v", err)
	}
	// Wait until the consumer has actually pulled the blocker off the
	// channel — only then is the buffer purely producer-visible.
	<-started
	noop := func(context.Context) error { return nil }
	for i := 0; i < capacity; i++ {
		if err := q.Submit(context.Background(), noop); err != nil {
			t.Fatalf("fill Submit %d: %v", i, err)
		}
	}

	err := q.SubmitWithTimeout(context.Background(), noop, 50*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("SubmitWithTimeout error = %v, want ErrTimeout", err)
	}
}

// TestWriteQueue_BurstHandlesGracefully (review-2026-05-11 MED critique):
// submit 2x capacity in a tight loop and assert ~half succeed, the rest
// return ErrQueueFull cleanly — no panic, no lost-work surprises.
func TestWriteQueue_BurstHandlesGracefully(t *testing.T) {
	const capacity = 32
	q := New(Config{Capacity: capacity})

	started := make(chan struct{})
	blockingDone := make(chan struct{})
	defer close(blockingDone)
	blocker := func(context.Context) error {
		close(started)
		<-blockingDone
		return nil
	}
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer q.Stop(time.Second)
	if err := q.Submit(context.Background(), blocker); err != nil {
		t.Fatalf("Submit blocker: %v", err)
	}
	// Wait until the consumer has dequeued the blocker so the buffer
	// is entirely available to the burst loop below.
	<-started

	const burst = 2 * capacity
	var success, full int
	for i := 0; i < burst; i++ {
		err := q.Submit(context.Background(), func(context.Context) error { return nil })
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrQueueFull):
			full++
		default:
			t.Fatalf("unexpected Submit error: %v", err)
		}
	}

	if success != capacity {
		t.Errorf("success count = %d, want exactly %d (burst capacity)", success, capacity)
	}
	if full != burst-capacity {
		t.Errorf("full count = %d, want %d (rejected overflow)", full, burst-capacity)
	}
}

// TestWriteQueue_StopDrainsRemaining asserts that pending ops in the
// channel run to completion when Stop is called.
func TestWriteQueue_StopDrainsRemaining(t *testing.T) {
	q := New(Config{Capacity: 8})
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var executed atomic.Int32
	op := func(context.Context) error {
		time.Sleep(5 * time.Millisecond)
		executed.Add(1)
		return nil
	}
	for i := 0; i < 8; i++ {
		if err := q.Submit(context.Background(), op); err != nil {
			t.Fatalf("Submit %d: %v", i, err)
		}
	}

	q.Stop(5 * time.Second)
	if got := executed.Load(); got != 8 {
		t.Errorf("executed after Stop = %d, want 8 (drain on shutdown)", got)
	}
	if state := q.Stats().State; state != StateStopped {
		t.Errorf("state after Stop = %s, want %s", state, StateStopped)
	}
}

// TestWriteQueue_PostStopReturnsUnavailable verifies the lifecycle
// invariant: once stopped, Submit must not accept new work — the writer
// is gone and the producer's canonical NDJSON has already won.
func TestWriteQueue_PostStopReturnsUnavailable(t *testing.T) {
	q := New(Config{Capacity: 4})
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	q.Stop(time.Second)
	err := q.Submit(context.Background(), func(context.Context) error { return nil })
	if !errors.Is(err, ErrWriterUnavailable) {
		t.Errorf("post-Stop Submit error = %v, want ErrWriterUnavailable", err)
	}
}

// TestWriteQueue_OpErrorIsObservable asserts that an op returning an
// error is surfaced via Stats.Errors and the OnError callback. This
// validates the diagnostic surface dashboard/collector-status reads.
func TestWriteQueue_OpErrorIsObservable(t *testing.T) {
	var captured atomic.Value
	q := New(Config{Capacity: 2, OnError: func(err error) {
		captured.Store(err.Error())
	}})
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	wantErr := errors.New("op failed")
	if err := q.Submit(context.Background(), func(context.Context) error { return wantErr }); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if q.Stats().Errors > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	q.Stop(time.Second)

	if got := q.Stats().Errors; got != 1 {
		t.Errorf("Stats.Errors = %d, want 1", got)
	}
	if got, _ := captured.Load().(string); got != wantErr.Error() {
		t.Errorf("OnError captured = %q, want %q", got, wantErr.Error())
	}
}

// TestWriteQueue_StatsTracksDepth verifies that depth + counters move
// correctly through Submit → consume → drain. The collector-status
// endpoint depends on these counters; this test locks in their semantics.
func TestWriteQueue_StatsTracksDepth(t *testing.T) {
	q := New(Config{Capacity: 4})

	if got := q.Stats().State; got != StateInit {
		t.Errorf("init state = %s, want %s", got, StateInit)
	}

	blockingDone := make(chan struct{})
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer q.Stop(time.Second)

	if got := q.Stats().State; got != StateRunning {
		t.Errorf("post-Start state = %s, want %s", got, StateRunning)
	}

	started := make(chan struct{})
	if err := q.Submit(context.Background(), func(context.Context) error {
		close(started)
		<-blockingDone
		return nil
	}); err != nil {
		t.Fatalf("Submit blocker: %v", err)
	}
	// The blocker is on the consumer goroutine, not the channel buffer.
	// Wait for the consumer to pick it up so we can observe a clean
	// "channel empty, one op executing" snapshot.
	<-started

	noop := func(context.Context) error { return nil }
	for i := 0; i < 3; i++ {
		if err := q.Submit(context.Background(), noop); err != nil {
			t.Fatalf("Submit %d: %v", i, err)
		}
	}
	stats := q.Stats()
	if stats.Depth != 3 {
		t.Errorf("Depth = %d, want 3 (after queueing 3 behind a blocked consumer)", stats.Depth)
	}
	if stats.Enqueued != 4 {
		t.Errorf("Enqueued = %d, want 4", stats.Enqueued)
	}
	if stats.Capacity != 4 {
		t.Errorf("Capacity = %d, want 4", stats.Capacity)
	}

	close(blockingDone)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if q.Stats().Dequeued == 4 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	stats = q.Stats()
	if stats.Dequeued != 4 {
		t.Errorf("Dequeued = %d, want 4 after drain", stats.Dequeued)
	}
	if stats.Depth != 0 {
		t.Errorf("Depth = %d, want 0 after drain", stats.Depth)
	}
}

// TestWriteQueue_ContextCancelledRejects checks that a cancelled
// producer context returns the context error without enqueuing.
func TestWriteQueue_ContextCancelledRejects(t *testing.T) {
	q := New(Config{Capacity: 4})
	if err := q.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer q.Stop(time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := q.Submit(ctx, func(context.Context) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Submit with cancelled ctx error = %v, want context.Canceled", err)
	}
	if err := q.SubmitWithTimeout(ctx, func(context.Context) error { return nil }, time.Second); !errors.Is(err, context.Canceled) {
		t.Errorf("SubmitWithTimeout with cancelled ctx error = %v, want context.Canceled", err)
	}
}

// TestWriteQueue_DefaultCapacity makes sure New(Config{}) lands on a
// non-zero capacity. This protects callers who forget to set it.
func TestWriteQueue_DefaultCapacity(t *testing.T) {
	q := New(Config{})
	if got := q.Capacity(); got != DefaultCapacity {
		t.Errorf("default Capacity = %d, want %d", got, DefaultCapacity)
	}
}
