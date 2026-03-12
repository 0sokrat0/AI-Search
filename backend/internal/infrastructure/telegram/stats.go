package telegram

import (
	"sync"
	"sync/atomic"
	"time"
)

type RuntimeStatsSnapshot struct {
	QueueCapacity      int    `json:"queueCapacity"`
	WorkerCount        int    `json:"workerCount"`
	WorkerBatchSize    int    `json:"workerBatchSize"`
	WorkerFlushMS      int64  `json:"workerFlushMs"`
	EnqueuedMessages   int64  `json:"enqueuedMessages"`
	DroppedMessages    int64  `json:"droppedMessages"`
	ProcessedMessages  int64  `json:"processedMessages"`
	ProcessedBatches   int64  `json:"processedBatches"`
	HandlerErrors      int64  `json:"handlerErrors"`
	LastBatchProcessed string `json:"lastBatchProcessed,omitempty"`
	LastErrorAt        string `json:"lastErrorAt,omitempty"`
	LastError          string `json:"lastError,omitempty"`
}

var runtimeStats struct {
	queueCapacity   atomic.Int64
	workerCount     atomic.Int64
	workerBatchSize atomic.Int64
	workerFlushMS   atomic.Int64

	enqueuedMessages  atomic.Int64
	droppedMessages   atomic.Int64
	processedMessages atomic.Int64
	processedBatches  atomic.Int64
	handlerErrors     atomic.Int64

	lastBatchUnix atomic.Int64
	lastErrorUnix atomic.Int64

	mu        sync.RWMutex
	lastError string
}

func RegisterRuntimeConfig(queueCapacity, workerCount, workerBatchSize int, workerFlush time.Duration) {
	runtimeStats.queueCapacity.Store(int64(queueCapacity))
	runtimeStats.workerCount.Store(int64(workerCount))
	runtimeStats.workerBatchSize.Store(int64(workerBatchSize))
	runtimeStats.workerFlushMS.Store(workerFlush.Milliseconds())
}

func MarkEnqueued() {
	runtimeStats.enqueuedMessages.Add(1)
}

func MarkDropped() {
	runtimeStats.droppedMessages.Add(1)
}

func MarkBatchProcessed(batchSize int) {
	runtimeStats.processedMessages.Add(int64(batchSize))
	runtimeStats.processedBatches.Add(1)
	runtimeStats.lastBatchUnix.Store(time.Now().UTC().Unix())
}

func MarkHandlerError(err error) {
	runtimeStats.handlerErrors.Add(1)
	runtimeStats.lastErrorUnix.Store(time.Now().UTC().Unix())
	if err == nil {
		return
	}
	runtimeStats.mu.Lock()
	runtimeStats.lastError = err.Error()
	runtimeStats.mu.Unlock()
}

func RuntimeStats() RuntimeStatsSnapshot {
	out := RuntimeStatsSnapshot{
		QueueCapacity:     int(runtimeStats.queueCapacity.Load()),
		WorkerCount:       int(runtimeStats.workerCount.Load()),
		WorkerBatchSize:   int(runtimeStats.workerBatchSize.Load()),
		WorkerFlushMS:     runtimeStats.workerFlushMS.Load(),
		EnqueuedMessages:  runtimeStats.enqueuedMessages.Load(),
		DroppedMessages:   runtimeStats.droppedMessages.Load(),
		ProcessedMessages: runtimeStats.processedMessages.Load(),
		ProcessedBatches:  runtimeStats.processedBatches.Load(),
		HandlerErrors:     runtimeStats.handlerErrors.Load(),
	}

	if ts := runtimeStats.lastBatchUnix.Load(); ts > 0 {
		out.LastBatchProcessed = time.Unix(ts, 0).UTC().Format(time.RFC3339)
	}
	if ts := runtimeStats.lastErrorUnix.Load(); ts > 0 {
		out.LastErrorAt = time.Unix(ts, 0).UTC().Format(time.RFC3339)
	}

	runtimeStats.mu.RLock()
	out.LastError = runtimeStats.lastError
	runtimeStats.mu.RUnlock()
	return out
}
