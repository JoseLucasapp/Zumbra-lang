package evaluator

import (
	"zumbra/object"
	"zumbra/object/builtins"
)

var builtinsList = make(map[string]*object.Builtin)

func init() {
	names := []string{
		"sizeOf", "first", "last", "allButFirst",
		"addToArrayStart", "addToArrayEnd", "removeFromArray",
		"show", "middleOf", "input", "max", "min",
		"indexOf", "addToDict", "deleteFromDict",
		"toString", "toInt", "toFloat", "toBool", "date", "organize", "toUppercase", "toLowercase", "capitalize", "removeWhiteSpaces", "sum", "bhaskara", "getFromDict",
	}

	for _, name := range names {
		if builtin := builtins.GetBuiltinByName(name); builtin != nil {
			builtinsList[name] = builtin
		}
	}
}
