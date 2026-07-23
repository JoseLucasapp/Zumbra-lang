package object

import (
	"sync"
	"testing"
	"time"
)

func TestTaskCancellationUnblocksAwait(t *testing.T) {
	task := NewTask()
	result := make(chan Object, 1)
	go func() { result <- task.Await() }()
	if !task.Cancel() {
		t.Fatal("expected cancellation to change pending task")
	}
	select {
	case value := <-result:
		err, ok := value.(*Error)
		if !ok || err.Message != "task cancelled" {
			t.Fatalf("unexpected cancellation result: %T %v", value, value)
		}
	case <-time.After(time.Second):
		t.Fatal("await remained blocked after cancellation")
	}
	if task.Cancel() {
		t.Fatal("second cancellation must not change the task")
	}
}

func TestTaskTimeoutDoesNotCompleteTask(t *testing.T) {
	task := NewTask()
	value, completed := task.AwaitTimeout(time.Millisecond)
	if completed || value.Type() != NULL_OBJ {
		t.Fatalf("unexpected timeout result: completed=%t value=%s", completed, value.Inspect())
	}
	task.Complete(&Integer{Value: 42})
	if got := task.Await().(*Integer).Value; got != 42 {
		t.Fatalf("expected completed value 42, got %d", got)
	}
}

func TestClosedChannelDrainsBufferedValues(t *testing.T) {
	channel := NewChannel(2)
	if err := channel.Send(&Integer{Value: 7}); err != nil {
		t.Fatal(err)
	}
	if err := channel.Send(&Integer{Value: 8}); err != nil {
		t.Fatal(err)
	}
	channel.Close()
	first, firstOpen := channel.Receive()
	second, secondOpen := channel.Receive()
	end, endOpen := channel.Receive()
	if !firstOpen || !secondOpen || endOpen {
		t.Fatalf("unexpected open states: %t %t %t", firstOpen, secondOpen, endOpen)
	}
	if first.(*Integer).Value != 7 || second.(*Integer).Value != 8 || end.Type() != NULL_OBJ {
		t.Fatalf("unexpected values: %s %s %s", first.Inspect(), second.Inspect(), end.Inspect())
	}
}

func TestAtomicIntUnderConcurrentLoad(t *testing.T) {
	value := NewAtomicInt(0)
	var workers sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := 0; index < 1000; index++ {
				value.Value.Add(1)
			}
		}()
	}
	workers.Wait()
	if got := value.Value.Load(); got != 16000 {
		t.Fatalf("expected 16000, got %d", got)
	}
}
