package mailx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrQueueStopped is returned when attempting to enqueue onto a stopped queue.
	ErrQueueStopped = errors.New("mailx: queue is stopped")
	// ErrQueueFull is returned when the queue buffer is completely filled.
	ErrQueueFull = errors.New("mailx: queue buffer is full")
)

// Queue handles asynchronous message delivery.
type Queue interface {
	// Enqueue adds a message to the delivery pipeline.
	Enqueue(ctx context.Context, msg Message) error
	// Start boots up background delivery workers.
	Start(ctx context.Context) error
	// Stop halts intake and flushes remaining messages within the context deadline.
	Stop(ctx context.Context) error
}

// MemoryQueueConfig holds tuning parameters for MemoryQueue.
type MemoryQueueConfig struct {
	BufferSize int
	Workers    int
	Logger     *slog.Logger
}

// MemoryQueue provides in-memory buffered channel delivery workers.
type MemoryQueue struct {
	driver  Driver
	cfg     MemoryQueueConfig
	ch      chan Message
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	running atomic.Bool
}

// NewMemoryQueue constructs a MemoryQueue backed by the given Driver.
func NewMemoryQueue(driver Driver, cfg MemoryQueueConfig) *MemoryQueue {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 256
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 2
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &MemoryQueue{
		driver: driver,
		cfg:    cfg,
		ch:     make(chan Message, cfg.BufferSize),
	}
}

// Start begins worker goroutines.
func (q *MemoryQueue) Start(ctx context.Context) error {
	if q.running.Swap(true) {
		return nil // already running
	}

	q.ctx, q.cancel = context.WithCancel(ctx)

	for i := 0; i < q.cfg.Workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	return nil
}

func (q *MemoryQueue) worker(id int) {
	defer q.wg.Done()

	for {
		select {
		case msg, ok := <-q.ch:
			if !ok {
				return
			}
			q.dispatch(msg)
		case <-q.ctx.Done():
			// Process remaining messages in buffer before terminating
			for {
				select {
				case msg, ok := <-q.ch:
					if !ok {
						return
					}
					q.dispatch(msg)
				default:
					return
				}
			}
		}
	}
}

func (q *MemoryQueue) dispatch(msg Message) {
	defer func() {
		if r := recover(); r != nil {
			q.cfg.Logger.Error("mailx: worker recovered from panic", "panic", r)
		}
	}()

	sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := q.driver.Send(sendCtx, msg); err != nil {
		q.cfg.Logger.Error("mailx: background delivery failed",
			"subject", msg.Subject,
			"to", msg.To,
			"err", err,
		)
	}
}

// Enqueue queues a message for asynchronous transmission.
func (q *MemoryQueue) Enqueue(ctx context.Context, msg Message) error {
	if !q.running.Load() {
		return ErrQueueStopped
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- msg:
		return nil
	default:
		return ErrQueueFull
	}
}

// Stop closes the message channel and waits for pending jobs to finish.
func (q *MemoryQueue) Stop(ctx context.Context) error {
	if !q.running.Swap(false) {
		return nil
	}

	close(q.ch)
	if q.cancel != nil {
		q.cancel()
	}

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("mailx: queue shutdown timeout: %w", ctx.Err())
	}
}
