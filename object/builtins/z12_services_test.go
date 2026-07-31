package builtins

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"zumbra/object"
)

func requireZ12ServiceValue(t *testing.T, value object.Object) object.Object {
	t.Helper()
	if errObj, ok := value.(*object.Error); ok {
		t.Fatalf("Z12 service builtin failed: %s", errObj.Message)
	}
	return value
}

func TestZ12TypedConfigurationRedactionMetricsTracingAndRateLimiting(t *testing.T) {
	config := requireZ12ServiceValue(t, ConfigFromBuiltin().Fn(z11TestDict(map[string]object.Object{
		"port":     NewString("8080"),
		"ratio":    NewString("1.5"),
		"debug":    NewString("true"),
		"password": NewString("secret"),
	}))).(*object.Config)
	if got := requireZ12ServiceValue(t, ConfigIntBuiltin().Fn(config, NewString("port"), NewInteger(0))).(*object.Integer).Value; got != 8080 {
		t.Fatalf("port=%d", got)
	}
	if got := requireZ12ServiceValue(t, ConfigFloatBuiltin().Fn(config, NewString("ratio"), NewFloat(0))).(*object.Float).Value; got != 1.5 {
		t.Fatalf("ratio=%v", got)
	}
	if !requireZ12ServiceValue(t, ConfigBoolBuiltin().Fn(config, NewString("debug"), NewBoolean(false))).(*object.Boolean).Value {
		t.Fatal("debug was not converted")
	}
	requireZ12ServiceValue(t, ConfigSecretBuiltin().Fn(config, NewString("password")))
	redacted := requireZ12ServiceValue(t, ConfigRedactedBuiltin().Fn(config)).(*object.Dict)
	if got := sqliteTestDictValue(t, redacted, "password").(*object.String).Value; got != "[REDACTED]" {
		t.Fatalf("password=%q", got)
	}

	registry := requireZ12ServiceValue(t, MetricsBuiltin().Fn()).(*object.MetricsRegistry)
	labels := z11TestDict(map[string]object.Object{"route": NewString("/health")})
	requireZ12ServiceValue(t, MetricsCounterBuiltin().Fn(registry, NewString("requests"), NewInteger(2), labels))
	requireZ12ServiceValue(t, MetricsGaugeBuiltin().Fn(registry, NewString("workers"), NewInteger(3), z11TestDict(nil)))
	requireZ12ServiceValue(t, MetricsHistogramBuiltin().Fn(registry, NewString("latency_ms"), NewFloat(12.5), labels))
	snapshot := requireZ12ServiceValue(t, MetricsSnapshotBuiltin().Fn(registry)).(*object.Dict)
	counters := sqliteTestDictValue(t, snapshot, "counters").(*object.Dict)
	if got := sqliteTestDictValue(t, counters, "requests{route=/health}").(*object.Float).Value; got != 2 {
		t.Fatalf("counter=%v", got)
	}

	span := requireZ12ServiceValue(t, TraceStartBuiltin().Fn(NewString("request"), labels)).(*object.TraceSpan)
	requireZ12ServiceValue(t, TraceEventBuiltin().Fn(span, NewString("handled"), z11TestDict(map[string]object.Object{"status": NewInteger(200)})))
	if !requireZ12ServiceValue(t, TraceActiveBuiltin().Fn(span)).(*object.Boolean).Value {
		t.Fatal("span unexpectedly inactive")
	}
	finished := requireZ12ServiceValue(t, TraceFinishBuiltin().Fn(span, NewString("ok"))).(*object.Dict)
	if got := sqliteTestDictValue(t, finished, "status").(*object.String).Value; got != "ok" {
		t.Fatalf("trace status=%q", got)
	}

	limiter := requireZ12ServiceValue(t, RateLimiterBuiltin().Fn(NewInteger(2), NewInteger(1000))).(*object.RateLimiter)
	for index, expected := range []bool{true, true, false} {
		result := requireZ12ServiceValue(t, RateAllowBuiltin().Fn(limiter, NewString("client"))).(*object.Dict)
		if got := sqliteTestDictValue(t, result, "allowed").(*object.Boolean).Value; got != expected {
			t.Fatalf("allow %d=%t", index, got)
		}
	}
	requireZ12ServiceValue(t, RateResetBuiltin().Fn(limiter, NewString("client")))
	reset := requireZ12ServiceValue(t, RateAllowBuiltin().Fn(limiter, NewString("client"))).(*object.Dict)
	if !sqliteTestDictValue(t, reset, "allowed").(*object.Boolean).Value {
		t.Fatal("rate limiter did not reset")
	}
}

func TestZ12SQLiteSessionStoreCreateRotateTouchAndDelete(t *testing.T) {
	store := requireZ12ServiceValue(t, SessionSQLiteBuiltin().Fn(NewString(":memory:"))).(*object.SessionStore)
	id := requireZ12ServiceValue(t, SessionCreateBuiltin().Fn(store, z11TestDict(map[string]object.Object{"user": NewString("Lucas")}), NewInteger(60000))).(*object.String).Value
	current := requireZ12ServiceValue(t, SessionGetBuiltin().Fn(store, NewString(id))).(*object.Dict)
	if got := sqliteTestDictValue(t, current, "user").(*object.String).Value; got != "Lucas" {
		t.Fatalf("user=%q", got)
	}
	requireZ12ServiceValue(t, SessionSetBuiltin().Fn(store, NewString(id), z11TestDict(map[string]object.Object{"user": NewString("Zumbra")}), NewInteger(60000)))
	if !requireZ12ServiceValue(t, SessionTouchBuiltin().Fn(store, NewString(id), NewInteger(60000))).(*object.Boolean).Value {
		t.Fatal("session touch failed")
	}
	rotated := requireZ12ServiceValue(t, SessionRotateBuiltin().Fn(store, NewString(id), NewInteger(60000))).(*object.String).Value
	if rotated == id {
		t.Fatal("session ID was not rotated")
	}
	if SessionGetBuiltin().Fn(store, NewString(id)).Type() != object.NULL_OBJ {
		t.Fatal("old session remained available")
	}
	requireZ12ServiceValue(t, SessionDeleteBuiltin().Fn(store, NewString(rotated)))
	if SessionGetBuiltin().Fn(store, NewString(rotated)).Type() != object.NULL_OBJ {
		t.Fatal("deleted session remained available")
	}
	requireZ12ServiceValue(t, SessionCloseBuiltin().Fn(store))
}

type memoryRedisEntry struct {
	value  object.Object
	expiry time.Time
}

type memoryRedisRuntime struct {
	mu     sync.Mutex
	open   bool
	values map[string]memoryRedisEntry
}

func newMemoryRedisRuntime() *memoryRedisRuntime {
	return &memoryRedisRuntime{open: true, values: map[string]memoryRedisEntry{}}
}
func (r *memoryRedisRuntime) Ping() error {
	if !r.open {
		return fmt.Errorf("closed")
	}
	return nil
}
func (r *memoryRedisRuntime) Set(key string, value object.Object, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	expiry := time.Time{}
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}
	r.values[key] = memoryRedisEntry{value: value, expiry: expiry}
	return nil
}
func (r *memoryRedisRuntime) Get(key string) (object.Object, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.values[key]
	if !ok {
		return nil, false, nil
	}
	if !entry.expiry.IsZero() && time.Now().After(entry.expiry) {
		delete(r.values, key)
		return nil, false, nil
	}
	return entry.value, true, nil
}
func (r *memoryRedisRuntime) Delete(keys []string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	for _, key := range keys {
		if _, ok := r.values[key]; ok {
			delete(r.values, key)
			count++
		}
	}
	return count, nil
}
func (r *memoryRedisRuntime) Exists(keys []string) (int64, error) {
	var count int64
	for _, key := range keys {
		_, ok, _ := r.Get(key)
		if ok {
			count++
		}
	}
	return count, nil
}
func (r *memoryRedisRuntime) Expire(key string, ttl time.Duration) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.values[key]
	if !ok {
		return false, nil
	}
	entry.expiry = time.Now().Add(ttl)
	r.values[key] = entry
	return true, nil
}
func (r *memoryRedisRuntime) TTL(key string) (time.Duration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.values[key]
	if !ok {
		return -2 * time.Second, nil
	}
	if entry.expiry.IsZero() {
		return -1, nil
	}
	return time.Until(entry.expiry), nil
}
func (r *memoryRedisRuntime) Increment(key string, amount int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var current int64
	if entry, ok := r.values[key]; ok {
		if integer, ok := entry.value.(*object.Integer); ok {
			current = integer.Value
		}
	}
	current += amount
	r.values[key] = memoryRedisEntry{value: NewInteger(current)}
	return current, nil
}
func (r *memoryRedisRuntime) Pipeline(commands []object.RedisCommand) ([]object.Object, error) {
	return nil, nil
}
func (r *memoryRedisRuntime) Close() error                { r.open = false; return nil }
func (r *memoryRedisRuntime) IsOpen() bool                { return r.open }
func (r *memoryRedisRuntime) PoolStats() map[string]int64 { return map[string]int64{"total": 1} }
func (r *memoryRedisRuntime) RotateSession(oldKey, newKey string, value object.Object, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, oldKey)
	r.values[newKey] = memoryRedisEntry{value: value, expiry: time.Now().Add(ttl)}
	return nil
}

func TestZ12RedisSessionStoreUsesPortableRuntime(t *testing.T) {
	runtime := newMemoryRedisRuntime()
	client := &object.RedisClient{Runtime: runtime, Address: "memory"}
	store := requireZ12ServiceValue(t, SessionRedisBuiltin().Fn(client, NewString("test:session:"))).(*object.SessionStore)
	id := requireZ12ServiceValue(t, SessionCreateBuiltin().Fn(store, z11TestDict(map[string]object.Object{"user": NewString("Lucas")}), NewInteger(60000))).(*object.String).Value
	rotated := requireZ12ServiceValue(t, SessionRotateBuiltin().Fn(store, NewString(id), NewInteger(60000))).(*object.String).Value
	if rotated == id {
		t.Fatal("Redis session was not rotated")
	}
	value := requireZ12ServiceValue(t, SessionGetBuiltin().Fn(store, NewString(rotated))).(*object.Dict)
	if got := sqliteTestDictValue(t, value, "user").(*object.String).Value; got != "Lucas" {
		t.Fatalf("user=%q", got)
	}
	requireZ12ServiceValue(t, SessionCloseBuiltin().Fn(store))
}
