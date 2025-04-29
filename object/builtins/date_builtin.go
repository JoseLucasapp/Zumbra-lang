package builtins

import (
	"time"
	"zumbra/object"
)

func DateBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return NewError("date() does not take arguments, got=%d", len(args))
			}
			return &object.Date{
				FullDate: time.Now(),
				Hour:     time.Now().Hour(),
				Minute:   time.Now().Minute(),
				Day:      time.Now().Day(),
				Second:   time.Now().Second(),
				Month:    time.Now().Month(),
				Year:     time.Now().Year()}
		},
	}
}
