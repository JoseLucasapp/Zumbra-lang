package evaluator

import (
	"zumbra/object"
	"zumbra/object/builtins"
)

var builtinsList = make(map[string]*object.Builtin)

func init() {
	names := []string{
		"addToArrayStart", "addToArrayEnd", "addToDict", "allButFirst", "bhaskara", "capitalize", "date", "deleteFromDict", "dictKeys", "dictValues",
		"first", "get", "getFromDict", "html", "indexOf", "input", "json_parse", "last", "max", "min", "organize", "randomFloat", "randomInteger",
		"registerRoute", "removeFromArray", "removeWhiteSpaces", "replace", "sendEmail", "sendWhatsapp", "server", "serveFile", "serveStatic",
		"show", "sizeOf", "sum", "toBool", "toFloat", "toInt", "toLowercase", "toString", "toUppercase",
	}

	for _, name := range names {
		if builtin := builtins.GetBuiltinByName(name); builtin != nil {
			builtinsList[name] = builtin
		}
	}
}
