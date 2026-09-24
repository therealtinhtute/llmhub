package redisqueue

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	defaultRetentionSeconds int64 = 60
	maxRetentionSeconds     int64 = 3600
	usageSubscriberBuffer         = 256
)

type queueItem struct {
	enqueuedAt time.Time
	payload    []byte
}

type queue struct {
	mu               sync.Mutex
	items            []queueItem
	head             int
	subscribers      map[uint64]chan []byte
	nextSubscriberID uint64
}

type UsageStore interface {
	AppendUsage(ctx context.Context, payload []byte, requestedAt time.Time) error
	PopUsage(ctx context.Context, count int) ([][]byte, error)
	PruneUsage(ctx context.Context, retention time.Duration) error
}

var (
	enabled          atomic.Bool
	retentionSeconds atomic.Int64
	global           queue
	usageStoreMu     sync.RWMutex
	usageStore       UsageStore
)

func init() {
	retentionSeconds.Store(defaultRetentionSeconds)
}

func SetEnabled(value bool) {
	enabled.Store(value)
	if !value {
		global.clear()
	}
}

func Enabled() bool {
	return enabled.Load()
}

func SetRetentionSeconds(value int) {
	normalized := int64(value)
	if normalized <= 0 {
		normalized = defaultRetentionSeconds
	} else if normalized > maxRetentionSeconds {
		normalized = maxRetentionSeconds
	}
	retentionSeconds.Store(normalized)
}

func SetUsageStore(store UsageStore) {
	usageStoreMu.Lock()
	usageStore = store
	usageStoreMu.Unlock()
	ensureUsageFlusher()
}

func Enqueue(payload []byte) {
	EnqueueUsage(payload, time.Now())
}

// Buffered usage persistence: EnqueueUsage never blocks the request path on a
// durable write. Rows flush when the buffer reaches usageFlushBatchSize or the
// usageFlushInterval ticker fires, whichever comes first. A crash loses at
// most the buffered window.
const (
	usageFlushBatchSize = 100
	usageFlushInterval  = 5 * time.Second
	usageBufferCap      = 10000
)

type usageRow struct {
	payload     []byte
	requestedAt time.Time
}

// usageBatcher is implemented by stores that can insert usage rows in one
// round-trip (PostgresStore). Other stores flush row-by-row via AppendUsage.
type usageBatcher interface {
	AppendUsageBatch(ctx context.Context, payloads [][]byte, requestedAts []time.Time) error
}

var usageBuffer = struct {
	mu      sync.Mutex
	rows    []usageRow
	started bool
}{}

func EnqueueUsage(payload []byte, requestedAt time.Time) {
	if !Enabled() {
		return
	}
	if len(payload) == 0 {
		return
	}
	if usageStoreSnapshot() != nil {
		bufferUsageRow(payload, requestedAt)
	}
	if global.publishToSubscribers(payload) {
		return
	}
	global.enqueue(payload)
}

func ensureUsageFlusher() {
	usageBuffer.mu.Lock()
	defer usageBuffer.mu.Unlock()
	if usageBuffer.started {
		return
	}
	usageBuffer.started = true
	go usageFlusherLoop()
}

// usageFlushSignal wakes the flusher when the buffer crosses the batch size;
// sends are non-blocking so the request path never waits on a durable write.
var usageFlushSignal = make(chan struct{}, 1)

func usageFlusherLoop() {
	ticker := time.NewTicker(usageFlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-usageFlushSignal:
		}
		if store := usageStoreSnapshot(); store != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			flushUsageBuffer(ctx, store)
			cancel()
		}
	}
}

func bufferUsageRow(payload []byte, requestedAt time.Time) {
	usageBuffer.mu.Lock()
	if len(usageBuffer.rows) >= usageBufferCap {
		// Shed the oldest rows rather than grow unbounded when the durable
		// store is down; usage stats tolerate the loss window.
		usageBuffer.rows = usageBuffer.rows[len(usageBuffer.rows)-usageBufferCap+1:]
	}
	usageBuffer.rows = append(usageBuffer.rows, usageRow{payload: payload, requestedAt: requestedAt})
	pending := len(usageBuffer.rows)
	usageBuffer.mu.Unlock()
	if pending >= usageFlushBatchSize {
		select {
		case usageFlushSignal <- struct{}{}:
		default:
		}
	}
}

// FlushUsage forces any buffered usage rows to the durable store. Called on
// service shutdown so the loss window does not stretch past process exit.
func FlushUsage(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	if store := usageStoreSnapshot(); store != nil {
		flushUsageBuffer(ctx, store)
	}
}

func flushUsageBuffer(ctx context.Context, store UsageStore) {
	usageBuffer.mu.Lock()
	if len(usageBuffer.rows) == 0 {
		usageBuffer.mu.Unlock()
		return
	}
	rows := usageBuffer.rows
	usageBuffer.rows = nil
	usageBuffer.mu.Unlock()

	payloads := make([][]byte, len(rows))
	requestedAts := make([]time.Time, len(rows))
	for i, row := range rows {
		payloads[i] = row.payload
		requestedAts[i] = row.requestedAt
	}

	var err error
	if batcher, ok := store.(usageBatcher); ok {
		err = batcher.AppendUsageBatch(ctx, payloads, requestedAts)
	} else {
		for i := range rows {
			if err = store.AppendUsage(ctx, rows[i].payload, rows[i].requestedAt); err != nil {
				// Requeue the unwritten tail ahead of newer buffered rows.
				rows = rows[i:]
				break
			}
		}
	}
	if err != nil {
		log.WithError(err).Warn("redisqueue: usage flush failed; rows requeued")
		usageBuffer.mu.Lock()
		combined := append(rows, usageBuffer.rows...)
		if len(combined) > usageBufferCap {
			combined = combined[len(combined)-usageBufferCap:]
		}
		usageBuffer.rows = combined
		usageBuffer.mu.Unlock()
	}
}

func PopOldest(count int) [][]byte {
	if !Enabled() {
		return nil
	}
	if count <= 0 {
		return nil
	}
	if store := usageStoreSnapshot(); store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		items, err := store.PopUsage(ctx, count)
		if err == nil {
			return items
		}
	}
	return global.popOldest(count)
}

func PrunePersistentUsage() {
	store := usageStoreSnapshot()
	if store == nil {
		return
	}
	windowSeconds := retentionSeconds.Load()
	if windowSeconds <= 0 {
		windowSeconds = defaultRetentionSeconds
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = store.PruneUsage(ctx, time.Duration(windowSeconds)*time.Second)
}

func usageStoreSnapshot() UsageStore {
	usageStoreMu.RLock()
	defer usageStoreMu.RUnlock()
	return usageStore
}

func SubscribeUsage() (<-chan []byte, func()) {
	return global.subscribeUsage()
}

func (q *queue) clear() {
	q.mu.Lock()

	subscribers := make([]chan []byte, 0, len(q.subscribers))
	for _, subscriber := range q.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	q.items = nil
	q.head = 0
	q.subscribers = nil
	q.mu.Unlock()

	for _, subscriber := range subscribers {
		close(subscriber)
	}
}

func (q *queue) enqueue(payload []byte) {
	now := time.Now()

	q.mu.Lock()
	defer q.mu.Unlock()

	q.pruneLocked(now)
	q.items = append(q.items, queueItem{
		enqueuedAt: now,
		payload:    append([]byte(nil), payload...),
	})
	q.maybeCompactLocked()
}

func (q *queue) publishToSubscribers(payload []byte) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.subscribers) == 0 {
		return false
	}

	for id, subscriber := range q.subscribers {
		cloned := append([]byte(nil), payload...)
		select {
		case subscriber <- cloned:
		default:
			delete(q.subscribers, id)
			close(subscriber)
		}
	}

	return true
}

func (q *queue) subscribeUsage() (<-chan []byte, func()) {
	subscriber := make(chan []byte, usageSubscriberBuffer)

	q.mu.Lock()
	if q.subscribers == nil {
		q.subscribers = make(map[uint64]chan []byte)
	}
	q.nextSubscriberID++
	id := q.nextSubscriberID
	q.subscribers[id] = subscriber
	q.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			q.unsubscribeUsage(id)
		})
	}
	return subscriber, unsubscribe
}

func (q *queue) unsubscribeUsage(id uint64) {
	q.mu.Lock()
	subscriber, ok := q.subscribers[id]
	if ok {
		delete(q.subscribers, id)
	}
	q.mu.Unlock()

	if ok {
		close(subscriber)
	}
}

func (q *queue) popOldest(count int) [][]byte {
	now := time.Now()

	q.mu.Lock()
	defer q.mu.Unlock()

	q.pruneLocked(now)
	available := len(q.items) - q.head
	if available <= 0 {
		q.items = nil
		q.head = 0
		return nil
	}
	if count > available {
		count = available
	}

	out := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		item := q.items[q.head+i]
		out = append(out, item.payload)
	}
	q.head += count
	q.maybeCompactLocked()
	return out
}

func (q *queue) pruneLocked(now time.Time) {
	if q.head >= len(q.items) {
		q.items = nil
		q.head = 0
		return
	}

	windowSeconds := retentionSeconds.Load()
	if windowSeconds <= 0 {
		windowSeconds = defaultRetentionSeconds
	}
	cutoff := now.Add(-time.Duration(windowSeconds) * time.Second)
	for q.head < len(q.items) && q.items[q.head].enqueuedAt.Before(cutoff) {
		q.head++
	}
}

func (q *queue) maybeCompactLocked() {
	if q.head == 0 {
		return
	}
	if q.head >= len(q.items) {
		q.items = nil
		q.head = 0
		return
	}
	if q.head < 1024 && q.head*2 < len(q.items) {
		return
	}
	q.items = append([]queueItem(nil), q.items[q.head:]...)
	q.head = 0
}
