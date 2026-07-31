package builtins

import (
	"time"
	"zumbra/object"
)

func concurrencyInt(value object.Object, name string) (int64, *object.Error) {
	switch v := value.(type) {
	case *object.Integer:
		return v.Value, nil
	case *object.FixedInteger:
		return v.SignedValue(), nil
	default:
		return 0, NewError("%s expects an integer, got %s", name, value.Type())
	}
}

func taskArg(args []object.Object, name string) (*object.Task, *object.Error) {
	if len(args) != 1 {
		return nil, NewError("%s expects 1 argument, got %d", name, len(args))
	}
	task, ok := args[0].(*object.Task)
	if !ok {
		return nil, NewError("%s expects Task, got %s", name, args[0].Type())
	}
	return task, nil
}

func JoinTaskBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		task, err := taskArg(args, "join")
		if err != nil {
			return err
		}
		return task.Await()
	}}
}
func CancelTaskBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		task, err := taskArg(args, "cancel")
		if err != nil {
			return err
		}
		return NewBoolean(task.Cancel())
	}}
}
func TaskDoneBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		task, err := taskArg(args, "taskDone")
		if err != nil {
			return err
		}
		return NewBoolean(task.Done())
	}}
}
func TaskCancelledBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		task, err := taskArg(args, "taskCancelled")
		if err != nil {
			return err
		}
		return NewBoolean(task.Cancelled())
	}}
}
func JoinTimeoutBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("joinTimeout expects 2 arguments, got %d", len(args))
		}
		task, ok := args[0].(*object.Task)
		if !ok {
			return NewError("joinTimeout expects Task, got %s", args[0].Type())
		}
		ms, err := concurrencyInt(args[1], "joinTimeout")
		if err != nil {
			return err
		}
		value, completed := task.AwaitTimeout(time.Duration(ms) * time.Millisecond)
		return &object.Array{Elements: []object.Object{value, NewBoolean(completed)}}
	}}
}
func SleepMsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sleepMs expects 1 argument, got %d", len(args))
		}
		ms, err := concurrencyInt(args[0], "sleepMs")
		if err != nil {
			return err
		}
		if ms < 0 {
			return NewError("sleepMs expects a non-negative duration")
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return &object.Null{}
	}}
}

func ChannelBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) > 1 {
			return NewError("channel expects 0 or 1 arguments, got %d", len(args))
		}
		capacity := int64(0)
		var err *object.Error
		if len(args) == 1 {
			capacity, err = concurrencyInt(args[0], "channel")
			if err != nil {
				return err
			}
		}
		if capacity < 0 {
			return NewError("channel capacity cannot be negative")
		}
		return object.NewChannel(int(capacity))
	}}
}
func SendBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("send expects 2 arguments, got %d", len(args))
		}
		ch, ok := args[0].(*object.Channel)
		if !ok {
			return NewError("send expects Channel, got %s", args[0].Type())
		}
		if err := ch.Send(args[1]); err != nil {
			return err
		}
		return &object.Null{}
	}}
}
func ReceiveBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("receive expects 1 argument, got %d", len(args))
		}
		ch, ok := args[0].(*object.Channel)
		if !ok {
			return NewError("receive expects Channel, got %s", args[0].Type())
		}
		value, _ := ch.Receive()
		return value
	}}
}
func ReceiveOkBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("receiveOk expects 1 argument, got %d", len(args))
		}
		ch, ok := args[0].(*object.Channel)
		if !ok {
			return NewError("receiveOk expects Channel, got %s", args[0].Type())
		}
		value, open := ch.Receive()
		return &object.Array{Elements: []object.Object{value, NewBoolean(open)}}
	}}
}
func ReceiveTimeoutBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("receiveTimeout expects 2 arguments, got %d", len(args))
		}
		ch, ok := args[0].(*object.Channel)
		if !ok {
			return NewError("receiveTimeout expects Channel, got %s", args[0].Type())
		}
		ms, err := concurrencyInt(args[1], "receiveTimeout")
		if err != nil {
			return err
		}
		value, open, received := ch.ReceiveTimeout(time.Duration(ms) * time.Millisecond)
		return &object.Array{Elements: []object.Object{value, NewBoolean(open), NewBoolean(received)}}
	}}
}
func CloseChannelBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("closeChannel expects 1 argument, got %d", len(args))
		}
		ch, ok := args[0].(*object.Channel)
		if !ok {
			return NewError("closeChannel expects Channel, got %s", args[0].Type())
		}
		return NewBoolean(ch.Close())
	}}
}
func ChannelClosedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("channelClosed expects 1 argument, got %d", len(args))
		}
		ch, ok := args[0].(*object.Channel)
		if !ok {
			return NewError("channelClosed expects Channel, got %s", args[0].Type())
		}
		return NewBoolean(ch.Closed())
	}}
}
func ChannelLenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("channelLen expects 1 argument, got %d", len(args))
		}
		ch, ok := args[0].(*object.Channel)
		if !ok {
			return NewError("channelLen expects Channel, got %s", args[0].Type())
		}
		return NewInteger(int64(ch.Len()))
	}}
}
func ChannelCapBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("channelCap expects 1 argument, got %d", len(args))
		}
		ch, ok := args[0].(*object.Channel)
		if !ok {
			return NewError("channelCap expects Channel, got %s", args[0].Type())
		}
		return NewInteger(int64(ch.Cap()))
	}}
}

func MutexBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("mutex expects no arguments")
		}
		return &object.Mutex{}
	}}
}
func LockBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("lock expects 1 argument")
		}
		switch m := args[0].(type) {
		case *object.Mutex:
			m.Value.Lock()
		case *object.RWMutex:
			m.Value.Lock()
		default:
			return NewError("lock expects Mutex or RWMutex")
		}
		return &object.Null{}
	}}
}
func UnlockBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("unlock expects 1 argument")
		}
		switch m := args[0].(type) {
		case *object.Mutex:
			m.Value.Unlock()
		case *object.RWMutex:
			m.Value.Unlock()
		default:
			return NewError("unlock expects Mutex or RWMutex")
		}
		return &object.Null{}
	}}
}
func RWMutexBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("rwMutex expects no arguments")
		}
		return &object.RWMutex{}
	}}
}
func RLockBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("rLock expects 1 argument")
		}
		m, ok := args[0].(*object.RWMutex)
		if !ok {
			return NewError("rLock expects RWMutex")
		}
		m.Value.RLock()
		return &object.Null{}
	}}
}
func RUnlockBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("rUnlock expects 1 argument")
		}
		m, ok := args[0].(*object.RWMutex)
		if !ok {
			return NewError("rUnlock expects RWMutex")
		}
		m.Value.RUnlock()
		return &object.Null{}
	}}
}

func WaitGroupBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("waitGroup expects no arguments")
		}
		return object.NewWaitGroup()
	}}
}
func WGAddBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("wgAdd expects 2 arguments")
		}
		w, ok := args[0].(*object.WaitGroup)
		if !ok {
			return NewError("wgAdd expects WaitGroup")
		}
		n, err := concurrencyInt(args[1], "wgAdd")
		if err != nil {
			return err
		}
		if err := w.Add(n); err != nil {
			return err
		}
		return &object.Null{}
	}}
}
func WGDoneBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("wgDone expects 1 argument")
		}
		w, ok := args[0].(*object.WaitGroup)
		if !ok {
			return NewError("wgDone expects WaitGroup")
		}
		if err := w.Done(); err != nil {
			return err
		}
		return &object.Null{}
	}}
}
func WGWaitBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("wgWait expects 1 argument")
		}
		w, ok := args[0].(*object.WaitGroup)
		if !ok {
			return NewError("wgWait expects WaitGroup")
		}
		w.Wait()
		return &object.Null{}
	}}
}

func SemaphoreBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("semaphore expects 1 argument")
		}
		n, err := concurrencyInt(args[0], "semaphore")
		if err != nil {
			return err
		}
		if n < 1 {
			return NewError("semaphore capacity must be positive")
		}
		return object.NewSemaphore(int(n))
	}}
}
func AcquireBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("acquire expects 1 argument")
		}
		s, ok := args[0].(*object.Semaphore)
		if !ok {
			return NewError("acquire expects Semaphore")
		}
		s.Tokens <- struct{}{}
		return &object.Null{}
	}}
}
func ReleaseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("release expects 1 argument")
		}
		s, ok := args[0].(*object.Semaphore)
		if !ok {
			return NewError("release expects Semaphore")
		}
		select {
		case <-s.Tokens:
			return &object.Null{}
		default:
			return NewError("release called without a matching acquire")
		}
	}}
}

func AtomicIntBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) > 1 {
			return NewError("atomicInt expects 0 or 1 arguments")
		}
		v := int64(0)
		var err *object.Error
		if len(args) == 1 {
			v, err = concurrencyInt(args[0], "atomicInt")
			if err != nil {
				return err
			}
		}
		return object.NewAtomicInt(v)
	}}
}
func AtomicLoadBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("atomicLoad expects 1 argument")
		}
		a, ok := args[0].(*object.AtomicInt)
		if !ok {
			return NewError("atomicLoad expects AtomicInt")
		}
		return NewInteger(a.Value.Load())
	}}
}
func AtomicStoreBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("atomicStore expects 2 arguments")
		}
		a, ok := args[0].(*object.AtomicInt)
		if !ok {
			return NewError("atomicStore expects AtomicInt")
		}
		v, err := concurrencyInt(args[1], "atomicStore")
		if err != nil {
			return err
		}
		a.Value.Store(v)
		return &object.Null{}
	}}
}
func AtomicAddBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("atomicAdd expects 2 arguments")
		}
		a, ok := args[0].(*object.AtomicInt)
		if !ok {
			return NewError("atomicAdd expects AtomicInt")
		}
		v, err := concurrencyInt(args[1], "atomicAdd")
		if err != nil {
			return err
		}
		return NewInteger(a.Value.Add(v))
	}}
}
func AtomicSwapBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("atomicSwap expects 2 arguments")
		}
		a, ok := args[0].(*object.AtomicInt)
		if !ok {
			return NewError("atomicSwap expects AtomicInt")
		}
		v, err := concurrencyInt(args[1], "atomicSwap")
		if err != nil {
			return err
		}
		return NewInteger(a.Value.Swap(v))
	}}
}
func AtomicCompareSwapBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("atomicCompareSwap expects 3 arguments")
		}
		a, ok := args[0].(*object.AtomicInt)
		if !ok {
			return NewError("atomicCompareSwap expects AtomicInt")
		}
		old, err := concurrencyInt(args[1], "atomicCompareSwap")
		if err != nil {
			return err
		}
		next, err := concurrencyInt(args[2], "atomicCompareSwap")
		if err != nil {
			return err
		}
		return NewBoolean(a.Value.CompareAndSwap(old, next))
	}}
}
