package builtins

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"zumbra/object"
)

var EnvVars = map[string]string{}

func loadEnvBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			path, ok := args[0].(*object.String)

			if !ok {
				return NewError("argument to `env` must be STRING, got %s", args[0].Type())
			}

			file, err := os.Open(path.Value)
			if err != nil {
				fmt.Println(fmt.Sprintf("failed to open file: %s", err))
				return nil
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					EnvVars[key] = value
				}
			}

			if err := scanner.Err(); err != nil {
				return NewError("failed to read file: %s", err)
			}

			return nil
		},
	}
}

func getEnvBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			key, ok := args[0].(*object.String)

			if !ok {
				return NewError("argument to `getEnv` must be STRING, got %s", args[0].Type())
			}

			value, ok := EnvVars[key.Value]

			if !ok {
				return nil
			}

			return &object.String{Value: value}
		},
	}
}
