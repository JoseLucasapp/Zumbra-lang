package builtins

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"zumbra/object"

	"github.com/redis/go-redis/v9"
)

var legacyRedisMu sync.Mutex
var legacyRedis *object.RedisClient

type redisHandle struct {
	mu      sync.RWMutex
	client  *redis.Client
	open    bool
	address string
	timeout time.Duration
}

func newRedisHandle(address, password string, db, poolSize int, timeout time.Duration) (*redisHandle, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if poolSize <= 0 {
		poolSize = 10
	}
	client := redis.NewClient(&redis.Options{Addr: address, Password: password, DB: db, PoolSize: poolSize, DialTimeout: timeout, ReadTimeout: timeout, WriteTimeout: timeout})
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect Redis: %w", err)
	}
	return &redisHandle{client: client, open: true, address: address, timeout: timeout}, nil
}
func (r *redisHandle) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), r.timeout)
}
func (r *redisHandle) IsOpen() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.open && r.client != nil
}
func (r *redisHandle) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.open || r.client == nil {
		return nil
	}
	err := r.client.Close()
	if err == nil {
		r.client = nil
		r.open = false
	}
	return err
}
func (r *redisHandle) Ping() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return fmt.Errorf("Redis client is closed")
	}
	ctx, cancel := r.context()
	defer cancel()
	return r.client.Ping(ctx).Err()
}
func redisEncode(value object.Object) (string, error) {
	raw, err := json.Marshal(objectToGoValue(value))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
func redisDecode(value string) (object.Object, error) {
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return &object.String{Value: value}, nil
	}
	return goValueToObject(decoded), nil
}
func (r *redisHandle) Set(key string, value object.Object, ttl time.Duration) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return fmt.Errorf("Redis client is closed")
	}
	encoded, err := redisEncode(value)
	if err != nil {
		return err
	}
	ctx, cancel := r.context()
	defer cancel()
	return r.client.Set(ctx, key, encoded, ttl).Err()
}
func (r *redisHandle) Get(key string) (object.Object, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return nil, false, fmt.Errorf("Redis client is closed")
	}
	ctx, cancel := r.context()
	defer cancel()
	value, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return &object.Null{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	decoded, err := redisDecode(value)
	return decoded, true, err
}
func (r *redisHandle) Delete(keys []string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return 0, fmt.Errorf("Redis client is closed")
	}
	ctx, cancel := r.context()
	defer cancel()
	return r.client.Del(ctx, keys...).Result()
}
func (r *redisHandle) Exists(keys []string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return 0, fmt.Errorf("Redis client is closed")
	}
	ctx, cancel := r.context()
	defer cancel()
	return r.client.Exists(ctx, keys...).Result()
}
func (r *redisHandle) Expire(key string, ttl time.Duration) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return false, fmt.Errorf("Redis client is closed")
	}
	ctx, cancel := r.context()
	defer cancel()
	return r.client.Expire(ctx, key, ttl).Result()
}
func (r *redisHandle) TTL(key string) (time.Duration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return 0, fmt.Errorf("Redis client is closed")
	}
	ctx, cancel := r.context()
	defer cancel()
	return r.client.TTL(ctx, key).Result()
}
func (r *redisHandle) Increment(key string, amount int64) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return 0, fmt.Errorf("Redis client is closed")
	}
	ctx, cancel := r.context()
	defer cancel()
	return r.client.IncrBy(ctx, key, amount).Result()
}
func (r *redisHandle) RotateSession(oldKey, newKey string, value object.Object, ttl time.Duration) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return fmt.Errorf("Redis client is closed")
	}
	encoded, err := redisEncode(value)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	ctx, cancel := r.context()
	defer cancel()
	const script = `
if redis.call("EXISTS", KEYS[1]) == 0 then
  return 0
end
redis.call("SET", KEYS[2], ARGV[1], "PX", ARGV[2])
redis.call("DEL", KEYS[1])
return 1`
	result, err := r.client.Eval(ctx, script, []string{oldKey, newKey}, encoded, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("session not found")
	}
	return nil
}

func (r *redisHandle) PoolStats() map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.client == nil {
		return map[string]int64{}
	}
	stats := r.client.PoolStats()
	return map[string]int64{"hits": int64(stats.Hits), "misses": int64(stats.Misses), "timeouts": int64(stats.Timeouts), "totalConnections": int64(stats.TotalConns), "idleConnections": int64(stats.IdleConns), "staleConnections": int64(stats.StaleConns)}
}
func (r *redisHandle) Pipeline(commands []object.RedisCommand) ([]object.Object, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.open || r.client == nil {
		return nil, fmt.Errorf("Redis client is closed")
	}
	ctx, cancel := r.context()
	defer cancel()
	pipe := r.client.Pipeline()
	results := make([]redis.Cmder, 0, len(commands))
	for _, command := range commands {
		switch command.Name {
		case "set":
			if len(command.Args) < 2 || len(command.Args) > 3 {
				return nil, fmt.Errorf("redis pipeline set expects key, value and optional ttlMs")
			}
			key, ok := command.Args[0].(*object.String)
			if !ok {
				return nil, fmt.Errorf("redis pipeline set key must be string")
			}
			ttl := time.Duration(0)
			if len(command.Args) == 3 {
				ms, e := concurrencyInt(command.Args[2], "redis pipeline set")
				if e != nil {
					return nil, fmt.Errorf("%s", e.Message)
				}
				ttl = time.Duration(ms) * time.Millisecond
			}
			encoded, err := redisEncode(command.Args[1])
			if err != nil {
				return nil, err
			}
			results = append(results, pipe.Set(ctx, key.Value, encoded, ttl))
		case "get":
			if len(command.Args) != 1 {
				return nil, fmt.Errorf("redis pipeline get expects key")
			}
			key, ok := command.Args[0].(*object.String)
			if !ok {
				return nil, fmt.Errorf("redis pipeline get key must be string")
			}
			results = append(results, pipe.Get(ctx, key.Value))
		case "del":
			keys, err := redisStringArgs(command.Args, "redis pipeline del")
			if err != nil {
				return nil, err
			}
			results = append(results, pipe.Del(ctx, keys...))
		case "exists":
			keys, err := redisStringArgs(command.Args, "redis pipeline exists")
			if err != nil {
				return nil, err
			}
			results = append(results, pipe.Exists(ctx, keys...))
		case "expire":
			if len(command.Args) != 2 {
				return nil, fmt.Errorf("redis pipeline expire expects key and ttlMs")
			}
			key, ok := command.Args[0].(*object.String)
			if !ok {
				return nil, fmt.Errorf("redis pipeline expire key must be string")
			}
			ms, e := concurrencyInt(command.Args[1], "redis pipeline expire")
			if e != nil {
				return nil, fmt.Errorf("%s", e.Message)
			}
			results = append(results, pipe.Expire(ctx, key.Value, time.Duration(ms)*time.Millisecond))
		case "ttl":
			if len(command.Args) != 1 {
				return nil, fmt.Errorf("redis pipeline ttl expects key")
			}
			key, ok := command.Args[0].(*object.String)
			if !ok {
				return nil, fmt.Errorf("redis pipeline ttl key must be string")
			}
			results = append(results, pipe.TTL(ctx, key.Value))
		case "incr":
			if len(command.Args) != 2 {
				return nil, fmt.Errorf("redis pipeline incr expects key and amount")
			}
			key, ok := command.Args[0].(*object.String)
			if !ok {
				return nil, fmt.Errorf("redis pipeline incr key must be string")
			}
			amount, e := concurrencyInt(command.Args[1], "redis pipeline incr")
			if e != nil {
				return nil, fmt.Errorf("%s", e.Message)
			}
			results = append(results, pipe.IncrBy(ctx, key.Value, amount))
		default:
			return nil, fmt.Errorf("unsupported Redis pipeline command %q", command.Name)
		}
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}
	output := make([]object.Object, 0, len(results))
	for _, command := range results {
		switch current := command.(type) {
		case *redis.StringCmd:
			value, err := current.Result()
			if err == redis.Nil {
				output = append(output, &object.Null{})
				continue
			}
			if err != nil {
				return nil, err
			}
			decoded, err := redisDecode(value)
			if err != nil {
				return nil, err
			}
			output = append(output, decoded)
		case *redis.IntCmd:
			value, err := current.Result()
			if err != nil {
				return nil, err
			}
			output = append(output, &object.Integer{Value: value})
		case *redis.BoolCmd:
			value, err := current.Result()
			if err != nil {
				return nil, err
			}
			output = append(output, NewBoolean(value))
		case *redis.DurationCmd:
			value, err := current.Result()
			if err != nil {
				return nil, err
			}
			output = append(output, &object.Integer{Value: value.Milliseconds()})
		case *redis.StatusCmd:
			if err := current.Err(); err != nil {
				return nil, err
			}
			output = append(output, NewBoolean(true))
		default:
			output = append(output, &object.Null{})
		}
	}
	return output, nil
}

func redisClientArg(value object.Object, name string) (*object.RedisClient, *object.Error) {
	client, ok := value.(*object.RedisClient)
	if !ok || client.Runtime == nil {
		return nil, NewError("%s expects RedisClient, got %s", name, value.Type())
	}
	return client, nil
}
func redisStringArgs(args []object.Object, name string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%s expects at least one key", name)
	}
	keys := make([]string, len(args))
	for i, arg := range args {
		value, ok := arg.(*object.String)
		if !ok {
			return nil, fmt.Errorf("%s key %d must be string", name, i+1)
		}
		keys[i] = value.Value
	}
	return keys, nil
}
func redisCommands(value object.Object) ([]object.RedisCommand, *object.Error) {
	array, ok := value.(*object.Array)
	if !ok {
		return nil, NewError("redisPipeline commands must be an array")
	}
	commands := make([]object.RedisCommand, 0, len(array.Elements))
	for index, item := range array.Elements {
		dict, ok := item.(*object.Dict)
		if !ok {
			return nil, NewError("redisPipeline command %d must be a dictionary", index)
		}
		var command object.RedisCommand
		for _, pair := range dict.Pairs {
			key, ok := pair.Key.(*object.String)
			if !ok {
				continue
			}
			switch key.Value {
			case "name":
				name, ok := pair.Value.(*object.String)
				if !ok {
					return nil, NewError("redisPipeline command name must be string")
				}
				command.Name = name.Value
			case "args":
				args, ok := pair.Value.(*object.Array)
				if !ok {
					return nil, NewError("redisPipeline command args must be array")
				}
				command.Args = args.Elements
			}
		}
		if command.Name == "" {
			return nil, NewError("redisPipeline command %d requires name", index)
		}
		commands = append(commands, command)
	}
	return commands, nil
}

func RedisOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 || len(args) > 5 {
			return NewError("redisOpen expects host, port, optional password, database and pool size")
		}
		host, ok := args[0].(*object.String)
		if !ok {
			return NewError("redisOpen host must be string")
		}
		port, e := concurrencyInt(args[1], "redisOpen port")
		if e != nil {
			return e
		}
		if port < 1 || port > 65535 {
			return NewError("redisOpen port must be between 1 and 65535")
		}
		password := ""
		database := int64(0)
		poolSize := int64(10)
		if len(args) > 2 {
			value, ok := args[2].(*object.String)
			if !ok {
				return NewError("redisOpen password must be string")
			}
			password = value.Value
		}
		if len(args) > 3 {
			database, e = concurrencyInt(args[3], "redisOpen database")
			if e != nil {
				return e
			}
		}
		if len(args) > 4 {
			poolSize, e = concurrencyInt(args[4], "redisOpen pool size")
			if e != nil {
				return e
			}
		}
		address := net.JoinHostPort(host.Value, strconv.FormatInt(port, 10))
		runtime, err := newRedisHandle(address, password, int(database), int(poolSize), 5*time.Second)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.RedisClient{Runtime: runtime, Address: address}
	}}
}
func RedisPingBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("redisPing expects 1 argument")
		}
		client, e := redisClientArg(args[0], "redisPing")
		if e != nil {
			return e
		}
		if err := client.Runtime.Ping(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func RedisCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("redisClose expects 1 argument")
		}
		client, e := redisClientArg(args[0], "redisClose")
		if e != nil {
			return e
		}
		if err := client.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func RedisIsOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("redisIsOpen expects 1 argument")
		}
		client, e := redisClientArg(args[0], "redisIsOpen")
		if e != nil {
			return e
		}
		return NewBoolean(client.Runtime.IsOpen())
	}}
}
func RedisSetObjectBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 3 || len(args) > 4 {
			return NewError("redisSetClient expects client, key, value and optional ttlMs")
		}
		client, e := redisClientArg(args[0], "redisSetClient")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("redis key must be string")
		}
		ttl := time.Duration(0)
		if len(args) == 4 {
			ms, e := concurrencyInt(args[3], "redisSetClient")
			if e != nil {
				return e
			}
			ttl = time.Duration(ms) * time.Millisecond
		}
		if err := client.Runtime.Set(key.Value, args[2], ttl); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func RedisGetObjectBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("redisGetClient expects client and key")
		}
		client, e := redisClientArg(args[0], "redisGetClient")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("redis key must be string")
		}
		value, found, err := client.Runtime.Get(key.Value)
		if err != nil {
			return NewError("%s", err)
		}
		if !found {
			return &object.Null{}
		}
		return value
	}}
}
func RedisDelObjectBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return NewError("redisDelete expects client and keys")
		}
		client, e := redisClientArg(args[0], "redisDelete")
		if e != nil {
			return e
		}
		keys, err := redisStringArgs(args[1:], "redisDelete")
		if err != nil {
			return NewError("%s", err)
		}
		count, err := client.Runtime.Delete(keys)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Integer{Value: count}
	}}
}
func RedisExistsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return NewError("redisExists expects client and keys")
		}
		client, e := redisClientArg(args[0], "redisExists")
		if e != nil {
			return e
		}
		keys, err := redisStringArgs(args[1:], "redisExists")
		if err != nil {
			return NewError("%s", err)
		}
		count, err := client.Runtime.Exists(keys)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Integer{Value: count}
	}}
}
func RedisExpireBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("redisExpire expects client, key and ttlMs")
		}
		client, e := redisClientArg(args[0], "redisExpire")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("redis key must be string")
		}
		ms, e := concurrencyInt(args[2], "redisExpire")
		if e != nil {
			return e
		}
		changed, err := client.Runtime.Expire(key.Value, time.Duration(ms)*time.Millisecond)
		if err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(changed)
	}}
}
func RedisTTLBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("redisTTL expects client and key")
		}
		client, e := redisClientArg(args[0], "redisTTL")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("redis key must be string")
		}
		ttl, err := client.Runtime.TTL(key.Value)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Integer{Value: ttl.Milliseconds()}
	}}
}
func RedisIncrementBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("redisIncrement expects client, key and amount")
		}
		client, e := redisClientArg(args[0], "redisIncrement")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("redis key must be string")
		}
		amount, e := concurrencyInt(args[2], "redisIncrement")
		if e != nil {
			return e
		}
		value, err := client.Runtime.Increment(key.Value, amount)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Integer{Value: value}
	}}
}
func RedisPipelineBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("redisPipeline expects client and commands")
		}
		client, e := redisClientArg(args[0], "redisPipeline")
		if e != nil {
			return e
		}
		commands, e := redisCommands(args[1])
		if e != nil {
			return e
		}
		values, err := client.Runtime.Pipeline(commands)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Array{Elements: values}
	}}
}
func RedisPoolStatsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("redisPoolStats expects 1 argument")
		}
		client, e := redisClientArg(args[0], "redisPoolStats")
		if e != nil {
			return e
		}
		return postgresStatsObject(client.Runtime.PoolStats())
	}}
}

// Legacy API delegates to the object client.
func RedisConnectionBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		value := RedisOpenBuiltin().Fn(args...)
		if value.Type() == object.ERROR_OBJ {
			return value
		}
		legacyRedisMu.Lock()
		if legacyRedis != nil {
			_ = legacyRedis.Runtime.Close()
		}
		legacyRedis = value.(*object.RedisClient)
		legacyRedisMu.Unlock()
		return &object.Null{}
	}}
}
func RedisSetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 || len(args) > 3 {
			return NewError("redisSet expects key, value and optional ttlMs")
		}
		legacyRedisMu.Lock()
		client := legacyRedis
		legacyRedisMu.Unlock()
		if client == nil {
			return NewError("redis is not connected; use redisConnection or redisOpen")
		}
		forward := []object.Object{client, args[0], args[1]}
		if len(args) == 3 {
			forward = append(forward, args[2])
		}
		return RedisSetObjectBuiltin().Fn(forward...)
	}}
}
func RedisGetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("redisGet expects key")
		}
		legacyRedisMu.Lock()
		client := legacyRedis
		legacyRedisMu.Unlock()
		if client == nil {
			return NewError("redis is not connected; use redisConnection or redisOpen")
		}
		return RedisGetObjectBuiltin().Fn(client, args[0])
	}}
}
func RedisDelBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return NewError("redisDel expects keys")
		}
		legacyRedisMu.Lock()
		client := legacyRedis
		legacyRedisMu.Unlock()
		if client == nil {
			return NewError("redis is not connected; use redisConnection or redisOpen")
		}
		return RedisDelObjectBuiltin().Fn(append([]object.Object{client}, args...)...)
	}}
}

func prependRedisReceiver(receiver object.Object, builtin *object.Builtin) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return builtin.Fn(append([]object.Object{receiver}, args...)...)
	}}
}
func RedisClientMethod(client *object.RedisClient, name string) object.Object {
	switch name {
	case "ping":
		return prependRedisReceiver(client, RedisPingBuiltin())
	case "set":
		return prependRedisReceiver(client, RedisSetObjectBuiltin())
	case "get":
		return prependRedisReceiver(client, RedisGetObjectBuiltin())
	case "delete":
		return prependRedisReceiver(client, RedisDelObjectBuiltin())
	case "exists":
		return prependRedisReceiver(client, RedisExistsBuiltin())
	case "expire":
		return prependRedisReceiver(client, RedisExpireBuiltin())
	case "ttl":
		return prependRedisReceiver(client, RedisTTLBuiltin())
	case "increment":
		return prependRedisReceiver(client, RedisIncrementBuiltin())
	case "pipeline":
		return prependRedisReceiver(client, RedisPipelineBuiltin())
	case "poolStats":
		return prependRedisReceiver(client, RedisPoolStatsBuiltin())
	case "close":
		return prependRedisReceiver(client, RedisCloseBuiltin())
	case "isOpen":
		return prependRedisReceiver(client, RedisIsOpenBuiltin())
	default:
		return nil
	}
}
