package builtins

import (
	"os"
	"path/filepath"
	"strings"
	"zumbra/object"
)

func ServeFileBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 && len(args) != 2 {
				return NewError("serveFile expects 1 or 2 arguments")
			}

			pathObj, ok := args[0].(*object.String)
			if !ok {
				return NewError("first argument to serveFile must be STRING, got=%s", args[0].Type())
			}

			path := filepath.Clean(pathObj.Value)

			content, err := os.ReadFile(path)
			if err != nil {
				return NewError("failed to read file: %s", err)
			}

			html := string(content)

			if len(args) == 1 {
				return &object.String{Value: html}
			}

			dictObj, ok := args[1].(*object.Dict)
			if !ok {
				return NewError("second argument to serveFile must be DICT, got=%s", args[1].Type())
			}

			for _, pair := range dictObj.Pairs {
				key, ok1 := pair.Key.(*object.String)
				value, ok2 := pair.Value.(*object.String)

				if ok1 && ok2 {
					placeholder := "{{" + key.Value + "}}"
					html = strings.ReplaceAll(html, placeholder, value.Value)
				}
			}

			return &object.String{Value: html}
		},
	}
}
