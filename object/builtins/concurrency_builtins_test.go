package builtins

import (
	"testing"
	"time"
	"zumbra/object"
)

func TestChannelCloseUnblocksReceive(t *testing.T) {
	channel := object.NewChannel(0)
	done := make(chan object.Object, 1)
	go func() {
		value, open := channel.Receive()
		if open {
			done <- NewError("expected closed channel")
		} else {
			done <- value
		}
	}()
	time.Sleep(time.Millisecond)
	if !channel.Close() {
		t.Fatal("expected first close to succeed")
	}
	select {
	case value := <-done:
		if value.Type() != object.NULL_OBJ {
			t.Fatalf("unexpected receive value: %s", value.Inspect())
		}
	case <-time.After(time.Second):
		t.Fatal("receive remained blocked after close")
	}
}

func TestAtomicBuiltins(t *testing.T) {
	atomic := AtomicIntBuiltin().Fn(&object.Integer{Value: 10})
	result := AtomicAddBuiltin().Fn(atomic, &object.Integer{Value: 5})
	if result.(*object.Integer).Value != 15 {
		t.Fatalf("unexpected atomic result: %s", result.Inspect())
	}
	swapped := AtomicCompareSwapBuiltin().Fn(atomic, &object.Integer{Value: 15}, &object.Integer{Value: 20})
	if !swapped.(*object.Boolean).Value {
		t.Fatal("compare-and-swap should succeed")
	}
}

func TestWaitGroupRejectsNegativeCounter(t *testing.T) {
	group := WaitGroupBuiltin().Fn()
	result := WGDoneBuiltin().Fn(group)
	if result.Type() != object.ERROR_OBJ {
		t.Fatalf("expected negative counter error, got %s", result.Inspect())
	}
}

func TestSemaphoreRejectsInvalidCapacityAndUnmatchedRelease(t *testing.T) {
	invalid := SemaphoreBuiltin().Fn(&object.Integer{Value: 0})
	if invalid.Type() != object.ERROR_OBJ {
		t.Fatalf("expected capacity error, got %s", invalid.Inspect())
	}
	semaphore := SemaphoreBuiltin().Fn(&object.Integer{Value: 1})
	unmatched := ReleaseBuiltin().Fn(semaphore)
	if unmatched.Type() != object.ERROR_OBJ {
		t.Fatalf("expected unmatched release error, got %s", unmatched.Inspect())
	}
}
