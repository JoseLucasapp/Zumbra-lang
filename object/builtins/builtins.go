package builtins

import (
	"fmt"
	"zumbra/object"
)

var Builtins = []struct {
	Name    string
	Builtin *object.Builtin
}{
	{
		"sizeOf", SizeOfBuiltin(),
	},
	{
		"show", ShowBuiltin(),
	},
	{
		"input", InputBuiltin(),
	},
	{
		"first", ArrayFirstBuiltin(),
	},
	{
		"last", ArrayLastBuiltin(),
	},
	{
		"allButFirst", AllButFirstBuiltin(),
	},
	{
		"addToArrayStart", AddToArrayStartBuiltin(),
	},
	{
		"addToArrayEnd", AddToArrayEndBuiltin(),
	},
	{
		"removeFromArray", RemoveFromArrayBuiltin(),
	},
	{
		"max", MaxBuiltin(),
	},
	{
		"min", MinBuiltin(),
	},
	{
		"indexOf", IndexOfBuiltin(),
	},
	{
		"addToDict", AddToDictBuiltin(),
	},
	{
		"deleteFromDict", DeleteFromDictBuiltin(),
	},
	{
		"toString", ToStringParserBuiltin(),
	},
	{
		"toInt", ToIntParserBuiltin(),
	},
	{
		"toFloat", ToFloatParserBuiltin(),
	},
	{
		"toBool", ToBoolParserBuiltin(),
	},
	{
		"date", DateBuiltin(),
	},
}

func NewBoolean(value bool) *object.Boolean {
	return &object.Boolean{Value: value}
}
func NewFloat(value float64) *object.Float {
	return &object.Float{Value: value}
}
func NewString(value string) *object.String {
	return &object.String{Value: value}
}

func NewInteger(value int64) *object.Integer {
	return &object.Integer{Value: value}
}
func NewError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func GetBuiltinByName(name string) *object.Builtin {
	for _, builtin := range Builtins {
		if builtin.Name == name {
			return builtin.Builtin
		}
	}
	return nil
}
