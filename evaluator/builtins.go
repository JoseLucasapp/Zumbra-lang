package evaluator

import (
	"zumbra/object"
)

var builtins = map[string]*object.Builtin{
	"sizeOf":          object.GetBuiltinByName("sizeOf"),
	"first":           object.GetBuiltinByName("first"),
	"last":            object.GetBuiltinByName("last"),
	"allButFirst ":    object.GetBuiltinByName("allButFirst"),
	"addToArray":      object.GetBuiltinByName("addToArray"),
	"removeFromArray": object.GetBuiltinByName("removeFromArray"),
	"show":            object.GetBuiltinByName("show"),
	"middleOf":        object.GetBuiltinByName("middleOf"),
	"input":           object.GetBuiltinByName("input"),
	"max":             object.GetBuiltinByName("max"),
	"min":             object.GetBuiltinByName("min"),
	"indexOf":         object.GetBuiltinByName("indexOf"),
	"addToDict":       object.GetBuiltinByName("addToDict"),
	"deleteFromDict":  object.GetBuiltinByName("deleteFromDict"),
}
