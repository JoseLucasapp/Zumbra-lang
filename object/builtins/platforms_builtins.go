package builtins

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"zumbra/object"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var postgresConnection *sql.DB
var redisClient *redis.Client
var supabaseURL string
var supabaseKey string

func PostgresConnectionBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments, postgresConnection(connectionString). got=%d, want=1", len(args))
			}

			connStr, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument to `postgresConnection` must be STRING, got %s", args[0].Type())
			}

			db, err := sql.Open("postgres", connStr.Value)
			if err != nil {
				return NewError("failed to open postgres connection. got %s", err)
			}

			if err := db.Ping(); err != nil {
				return NewError("failed to ping postgres. got %s", err)
			}

			postgresConnection = db
			return &object.Null{}
		},
	}
}

func PostgresExecBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments, postgresExec(query). got=%d, want=1", len(args))
			}
			if postgresConnection == nil {
				return NewError("postgres is not connected. Use postgresConnection(...) first.")
			}

			query, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument to `postgresExec` must be STRING, got %s", args[0].Type())
			}

			_, err := postgresConnection.Exec(query.Value)
			if err != nil {
				return NewError("failed to exec postgres query. got %s", err)
			}

			return &object.Null{}
		},
	}
}

func PostgresQueryBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments, postgresQuery(query). got=%d, want=1", len(args))
			}
			if postgresConnection == nil {
				return NewError("postgres is not connected. Use postgresConnection(...) first.")
			}

			query, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument to `postgresQuery` must be STRING, got %s", args[0].Type())
			}

			rows, err := postgresConnection.Query(query.Value)
			if err != nil {
				return NewError("failed to query postgres. got %s", err)
			}
			defer rows.Close()

			cols, err := rows.Columns()
			if err != nil {
				return NewError("failed to get postgres columns. got %s", err)
			}

			var result []object.Object

			for rows.Next() {
				values := make([]interface{}, len(cols))
				ptrs := make([]interface{}, len(cols))
				for i := range values {
					ptrs[i] = &values[i]
				}

				if err := rows.Scan(ptrs...); err != nil {
					return NewError("failed to scan postgres row. got %s", err)
				}

				pairs := map[object.DictKey]object.DictPair{}
				for i, col := range cols {
					key := &object.String{Value: col}
					val := goValueToObject(values[i])
					pairs[key.DictKey()] = object.DictPair{Key: key, Value: val}
				}

				result = append(result, &object.Dict{Pairs: pairs})
			}

			return &object.Array{Elements: result}
		},
	}
}

func RedisConnectionBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments, redisConnection(addr, password, db). got=%d, want=3", len(args))
			}

			addr, ok1 := args[0].(*object.String)
			password, ok2 := args[1].(*object.String)
			dbNum, ok3 := args[2].(*object.Integer)

			if !ok1 || !ok2 || !ok3 {
				return NewError("redisConnection(addr, password, db) expects STRING, STRING, INTEGER")
			}

			redisClient = redis.NewClient(&redis.Options{
				Addr:     addr.Value,
				Password: password.Value,
				DB:       int(dbNum.Value),
			})

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := redisClient.Ping(ctx).Err(); err != nil {
				return NewError("failed to connect to redis. got %s", err)
			}

			return &object.Null{}
		},
	}
}

func RedisSetBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, redisSet(key, value). got=%d, want=2", len(args))
			}
			if redisClient == nil {
				return NewError("redis is not connected. Use redisConnection(...) first.")
			}

			key, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument 1 to `redisSet` must be STRING")
			}

			value := objectToPlainString(args[1])

			ctx := context.Background()
			if err := redisClient.Set(ctx, key.Value, value, 0).Err(); err != nil {
				return NewError("failed to set redis key. got %s", err)
			}

			return &object.Null{}
		},
	}
}

func RedisGetBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments, redisGet(key). got=%d, want=1", len(args))
			}
			if redisClient == nil {
				return NewError("redis is not connected. Use redisConnection(...) first.")
			}

			key, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument 1 to `redisGet` must be STRING")
			}

			ctx := context.Background()
			value, err := redisClient.Get(ctx, key.Value).Result()
			if err == redis.Nil {
				return &object.Null{}
			}
			if err != nil {
				return NewError("failed to get redis key. got %s", err)
			}

			return &object.String{Value: value}
		},
	}
}

func RedisDelBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments, redisDel(key). got=%d, want=1", len(args))
			}
			if redisClient == nil {
				return NewError("redis is not connected. Use redisConnection(...) first.")
			}

			key, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument 1 to `redisDel` must be STRING")
			}

			ctx := context.Background()
			if err := redisClient.Del(ctx, key.Value).Err(); err != nil {
				return NewError("failed to delete redis key. got %s", err)
			}

			return &object.Null{}
		},
	}
}

func SupabaseConnectionBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseConnection(url, key). got=%d, want=2", len(args))
			}

			urlObj, ok1 := args[0].(*object.String)
			keyObj, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseConnection(url, key) expects STRING, STRING")
			}

			supabaseURL = strings.TrimRight(urlObj.Value, "/")
			supabaseKey = keyObj.Value
			return &object.Null{}
		},
	}
}

func SupabaseSelectBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseSelect(table, selectQuery). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok1 := args[0].(*object.String)
			selectQuery, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("supabaseSelect(table, selectQuery) expects STRING, STRING")
			}

			url := fmt.Sprintf("%s/rest/v1/%s?select=%s", supabaseURL, table.Value, selectQuery.Value)

			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to call supabase. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			var decoded interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				return NewError("failed to decode supabase response. got %s", err)
			}

			return goValueToObject(decoded)
		},
	}
}

func SupabaseInsertBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments, supabaseInsert(table, payload). got=%d, want=2", len(args))
			}
			if supabaseURL == "" || supabaseKey == "" {
				return NewError("supabase is not connected. Use supabaseConnection(...) first.")
			}

			table, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument 1 to `supabaseInsert` must be STRING")
			}

			payloadBytes, err := json.Marshal(objectToGoValue(args[1]))
			if err != nil {
				return NewError("failed to encode supabase payload. got %s", err)
			}

			url := fmt.Sprintf("%s/rest/v1/%s", supabaseURL, table.Value)
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
			req.Header.Set("apikey", supabaseKey)
			req.Header.Set("Authorization", "Bearer "+supabaseKey)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Prefer", "return=representation")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return NewError("failed to insert into supabase. got %s", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			var decoded interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				return NewError("failed to decode supabase insert response. got %s", err)
			}

			return goValueToObject(decoded)
		},
	}
}

func objectToPlainString(obj object.Object) string {
	switch v := obj.(type) {
	case *object.String:
		return v.Value
	default:
		return v.Inspect()
	}
}
