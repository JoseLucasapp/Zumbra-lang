package evaluator

import (
	"vaja/object"
)

var builtins = map[string]*object.Builtin{
	"sizeOf": object.GetBuiltinByName("sizeOf"),

	"first": object.GetBuiltinByName("first"),

	"last": object.GetBuiltinByName("last"),

	"allButFirst ": object.GetBuiltinByName("allButFirst"),

	"addToArray": object.GetBuiltinByName("addToArray"),

	"removeFromArray": object.GetBuiltinByName("removeFromArray"),

	"show":     object.GetBuiltinByName("show"),
	"middleOf": object.GetBuiltinByName("middleOf"),
}
