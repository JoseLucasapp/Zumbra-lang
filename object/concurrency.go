package object

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Task is the runtime representation of a concurrently executing Zumbra call.
// Cancellation is cooperative: it marks the task and unblocks waiters, but it
// cannot forcibly stop arbitrary user code.
type Task struct {
	done      chan struct{}
	once      sync.Once
	mu        sync.RWMutex
	result    Object
	err       *Error
	cancelled atomic.Bool
}

func NewTask() *Task             { return &Task{done: make(chan struct{})} }
func (t *Task) Type() ObjectType { return TASK_OBJ }
func (t *Task) Inspect() string {
	if t == nil {
		return "Task<nil>"
	}
	if t.cancelled.Load() {
		return "Task<cancelled>"
	}
	select {
	case <-t.done:
		return "Task<done>"
	default:
		return "Task<pending>"
	}
}
func (t *Task) Complete(result Object) {
	t.once.Do(func() {
		t.mu.Lock()
		if err, ok := result.(*Error); ok {
			t.err = err
		} else {
			t.result = result
		}
		t.mu.Unlock()
		close(t.done)
	})
}
func (t *Task) Cancel() bool {
	if t == nil {
		return false
	}
	changed := t.cancelled.CompareAndSwap(false, true)
	if changed {
		t.once.Do(func() {
			t.mu.Lock()
			t.err = &Error{Message: "task cancelled"}
			t.mu.Unlock()
			close(t.done)
		})
	}
	return changed
}
func (t *Task) Done() bool {
	if t == nil {
		return true
	}
	select {
	case <-t.done:
		return true
	default:
		return false
	}
}
func (t *Task) Cancelled() bool { return t != nil && t.cancelled.Load() }
func (t *Task) Await() Object {
	if t == nil {
		return &Error{Message: "cannot await a nil task"}
	}
	<-t.done
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.err != nil {
		return t.err
	}
	if t.result == nil {
		return &Null{}
	}
	return t.result
}
func (t *Task) AwaitTimeout(timeout time.Duration) (Object, bool) {
	if t == nil {
		return &Error{Message: "cannot await a nil task"}, true
	}
	if timeout < 0 {
		return t.Await(), true
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-t.done:
		return t.Await(), true
	case <-timer.C:
		return &Null{}, false
	}
}

// Channel wraps a Go channel and uses a separate closed signal. The data
// channel is never closed directly, which lets closeChannel safely unblock
// blocked senders without racing with a send panic.
type Channel struct {
	values chan Object
	done   chan struct{}
	once   sync.Once
	closed atomic.Bool
}

func NewChannel(capacity int) *Channel {
	if capacity < 0 {
		capacity = 0
	}
	return &Channel{values: make(chan Object, capacity), done: make(chan struct{})}
}
func (c *Channel) Type() ObjectType { return CHANNEL_OBJ }
func (c *Channel) Inspect() string {
	if c == nil {
		return "Channel<nil>"
	}
	return fmt.Sprintf("Channel<len=%d cap=%d closed=%t>", len(c.values), cap(c.values), c.closed.Load())
}
func (c *Channel) Send(value Object) *Error {
	if c == nil {
		return &Error{Message: "cannot send to nil channel"}
	}
	select {
	case <-c.done:
		return &Error{Message: "cannot send to closed channel"}
	default:
	}
	select {
	case c.values <- value:
		return nil
	case <-c.done:
		return &Error{Message: "cannot send to closed channel"}
	}
}
func (c *Channel) Receive() (Object, bool) {
	if c == nil {
		return &Null{}, false
	}
	// Drain values already buffered before reporting closure.
	select {
	case value := <-c.values:
		if value == nil {
			value = &Null{}
		}
		return value, true
	default:
	}
	select {
	case value := <-c.values:
		if value == nil {
			value = &Null{}
		}
		return value, true
	case <-c.done:
		select {
		case value := <-c.values:
			if value == nil {
				value = &Null{}
			}
			return value, true
		default:
			return &Null{}, false
		}
	}
}
func (c *Channel) ReceiveTimeout(timeout time.Duration) (Object, bool, bool) {
	if timeout < 0 {
		value, open := c.Receive()
		return value, open, true
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case value := <-c.values:
		if value == nil {
			value = &Null{}
		}
		return value, true, true
	case <-c.done:
		select {
		case value := <-c.values:
			if value == nil {
				value = &Null{}
			}
			return value, true, true
		default:
			return &Null{}, false, true
		}
	case <-timer.C:
		return &Null{}, !c.closed.Load(), false
	}
}
func (c *Channel) Close() bool {
	if c == nil {
		return false
	}
	changed := false
	c.once.Do(func() { changed = true; c.closed.Store(true); close(c.done) })
	return changed
}
func (c *Channel) Closed() bool { return c == nil || c.closed.Load() }
func (c *Channel) Len() int {
	if c == nil {
		return 0
	}
	return len(c.values)
}
func (c *Channel) Cap() int {
	if c == nil {
		return 0
	}
	return cap(c.values)
}

// Mutex is intentionally non-reentrant, matching Go and pthread mutexes.
type Mutex struct{ Value sync.Mutex }

func (m *Mutex) Type() ObjectType { return MUTEX_OBJ }
func (m *Mutex) Inspect() string  { return "Mutex" }

type RWMutex struct{ Value sync.RWMutex }

func (m *RWMutex) Type() ObjectType { return RW_MUTEX_OBJ }
func (m *RWMutex) Inspect() string  { return "RWMutex" }

type WaitGroup struct {
	mu    sync.Mutex
	cond  *sync.Cond
	count int64
}

func NewWaitGroup() *WaitGroup {
	w := &WaitGroup{}
	w.cond = sync.NewCond(&w.mu)
	return w
}
func (w *WaitGroup) ensureCond() {
	if w.cond == nil {
		w.cond = sync.NewCond(&w.mu)
	}
}
func (w *WaitGroup) Add(delta int64) *Error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ensureCond()
	if w.count+delta < 0 {
		return &Error{Message: "negative WaitGroup counter"}
	}
	w.count += delta
	if w.count == 0 {
		w.cond.Broadcast()
	}
	return nil
}
func (w *WaitGroup) Done() *Error { return w.Add(-1) }
func (w *WaitGroup) Wait() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ensureCond()
	for w.count != 0 {
		w.cond.Wait()
	}
}
func (w *WaitGroup) Type() ObjectType { return WAIT_GROUP_OBJ }
func (w *WaitGroup) Inspect() string  { return "WaitGroup" }

type Semaphore struct{ Tokens chan struct{} }

func NewSemaphore(size int) *Semaphore {
	return &Semaphore{Tokens: make(chan struct{}, size)}
}
func (s *Semaphore) Type() ObjectType { return SEMAPHORE_OBJ }
func (s *Semaphore) Inspect() string {
	return fmt.Sprintf("Semaphore<%d/%d>", len(s.Tokens), cap(s.Tokens))
}

type AtomicInt struct{ Value atomic.Int64 }

func NewAtomicInt(value int64) *AtomicInt { a := &AtomicInt{}; a.Value.Store(value); return a }
func (a *AtomicInt) Type() ObjectType     { return ATOMIC_INT_OBJ }
func (a *AtomicInt) Inspect() string      { return fmt.Sprintf("AtomicInt<%d>", a.Value.Load()) }
