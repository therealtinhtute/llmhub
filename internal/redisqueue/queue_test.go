package redisqueue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestEnqueueBroadcastsToUsageSubscribersAndSkipsQueue(t *testing.T) {
	withEnabledQueue(t, func() {
		first, unsubscribeFirst := SubscribeUsage()
		defer unsubscribeFirst()
		second, unsubscribeSecond := SubscribeUsage()
		defer unsubscribeSecond()

		Enqueue([]byte("usage-record"))

		requireUsageSubscriberPayload(t, first, "usage-record")
		requireUsageSubscriberPayload(t, second, "usage-record")

		if items := PopOldest(1); len(items) != 0 {
			t.Fatalf("PopOldest() items = %q, want empty after subscriber broadcast", items)
		}

		unsubscribeFirst()
		unsubscribeSecond()

		Enqueue([]byte("queued-record"))
		items := PopOldest(1)
		if len(items) != 1 || string(items[0]) != "queued-record" {
			t.Fatalf("PopOldest() items = %q, want queued record after unsubscribe", items)
		}
	})
}

func TestSetEnabledFalseClosesUsageSubscribers(t *testing.T) {
	withEnabledQueue(t, func() {
		subscriber, unsubscribe := SubscribeUsage()
		defer unsubscribe()

		SetEnabled(false)

		select {
		case _, ok := <-subscriber:
			if ok {
				t.Fatalf("subscriber channel remained open after SetEnabled(false)")
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for subscriber close")
		}
	})
}

func requireUsageSubscriberPayload(t *testing.T, subscriber <-chan []byte, want string) {
	t.Helper()

	select {
	case got, ok := <-subscriber:
		if !ok {
			t.Fatalf("subscriber closed before receiving %q", want)
		}
		if string(got) != want {
			t.Fatalf("subscriber payload = %q, want %q", string(got), want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for subscriber payload %q", want)
	}
}

type fakeBatchUsageStore struct {
	mu         sync.Mutex
	batches    int
	batchRows  int
	singleRows int
	popped     [][]byte
	failNext   bool
}

// counts returns the counters under the store mutex — the background flusher
// goroutine mutates them, so test reads must not touch the fields directly.
func (f *fakeBatchUsageStore) counts() (batches, batchRows, singleRows int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.batches, f.batchRows, f.singleRows
}

func (f *fakeBatchUsageStore) setFailNext() {
	f.mu.Lock()
	f.failNext = true
	f.mu.Unlock()
}

func (f *fakeBatchUsageStore) AppendUsage(context.Context, []byte, time.Time) error {
	f.mu.Lock()
	f.singleRows++
	f.mu.Unlock()
	return nil
}

func (f *fakeBatchUsageStore) AppendUsageBatch(_ context.Context, payloads [][]byte, _ []time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return errFlushTest
	}
	f.batches++
	f.batchRows += len(payloads)
	return nil
}

func (f *fakeBatchUsageStore) PopUsage(context.Context, int) ([][]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.popped, nil
}

func (f *fakeBatchUsageStore) PruneUsage(context.Context, time.Duration) error { return nil }

var errFlushTest = errors.New("flush failed")

func resetUsageBufferForTest() {
	usageBuffer.mu.Lock()
	usageBuffer.rows = nil
	usageBuffer.mu.Unlock()
}

func TestEnqueueUsageBuffersAndFlushBatches(t *testing.T) {
	withEnabledQueue(t, func() {
		resetUsageBufferForTest()
		store := &fakeBatchUsageStore{}
		SetUsageStore(store)
		defer func() { SetUsageStore(nil); resetUsageBufferForTest() }()

		EnqueueUsage([]byte(`{"id":1}`), time.Now())
		EnqueueUsage([]byte(`{"id":2}`), time.Now())
		EnqueueUsage([]byte(`{"id":3}`), time.Now())

		if store.batches != 0 || store.singleRows != 0 {
			t.Fatalf("durable writes before flush: batches=%d singles=%d, want 0", store.batches, store.singleRows)
		}

		FlushUsage(context.Background())
		if store.batches != 1 || store.batchRows != 3 {
			t.Fatalf("after flush: batches=%d rows=%d, want 1 batch of 3", store.batches, store.batchRows)
		}
	})
}

func TestEnqueueUsageCountTriggerFlushes(t *testing.T) {
	withEnabledQueue(t, func() {
		resetUsageBufferForTest()
		store := &fakeBatchUsageStore{}
		SetUsageStore(store)
		defer func() { SetUsageStore(nil); resetUsageBufferForTest() }()

		for i := 0; i < usageFlushBatchSize; i++ {
			EnqueueUsage([]byte(`{"id":1}`), time.Now())
		}
		// Flush is signaled to the background flusher, not run inline — poll
		// for it with a deadline so the request goroutine stays write-free.
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if _, rows, _ := store.counts(); rows == usageFlushBatchSize {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if _, rows, _ := store.counts(); rows != usageFlushBatchSize {
			t.Fatalf("flushed rows = %d, want %d after count trigger", rows, usageFlushBatchSize)
		}
	})
}

func TestUsageFlushErrorRequeuesRows(t *testing.T) {
	withEnabledQueue(t, func() {
		resetUsageBufferForTest()
		store := &fakeBatchUsageStore{failNext: true}
		SetUsageStore(store)
		defer func() { SetUsageStore(nil); resetUsageBufferForTest() }()

		EnqueueUsage([]byte(`{"id":1}`), time.Now())
		FlushUsage(context.Background())
		if b, _, _ := store.counts(); b != 0 {
			t.Fatalf("batches = %d, want 0 after failed flush", b)
		}
		FlushUsage(context.Background())
		if _, rows, _ := store.counts(); rows != 1 {
			t.Fatalf("retried rows = %d, want 1 requeued row persisted", rows)
		}
	})
}
