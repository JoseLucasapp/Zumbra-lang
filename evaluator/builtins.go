package evaluator

import (
	"zumbra/object"
	"zumbra/object/builtins"
)

var builtinsList = make(map[string]*object.Builtin)

func init() {
	arrays := []string{
		"addToArrayStart", "addToArrayEnd", "allButFirst", "first", "indexOf", "last", "max", "min", "organize", "removeFromArray", "sizeOf", "sum",
	}

	dicts := []string{
		"addToDict", "deleteFromDict", "dictKeys", "dictValues", "getFromDict",
	}

	http := []string{
		"get", "html", "registerRoute", "server", "serveFile", "serveStatic",
	}

	parsers := []string{
		"json_parse", "toBool", "toFloat", "toInt", "toString",
	}

	stringUtils := []string{
		"capitalize", "removeWhiteSpaces", "replace", "toLowercase", "toUppercase",
	}

	numbersUtils := []string{
		"bhaskara", "randomFloat", "randomInteger",
	}

	ioUtils := []string{
		"input", "show",
	}

	messages := []string{
		"sendEmail", "sendWhatsapp",
	}

	extras := []string{
		"date",
	}

	allBuiltins := append(arrays, dicts...)
	allBuiltins = append(allBuiltins, http...)
	allBuiltins = append(allBuiltins, parsers...)
	allBuiltins = append(allBuiltins, stringUtils...)
	allBuiltins = append(allBuiltins, numbersUtils...)
	allBuiltins = append(allBuiltins, ioUtils...)
	allBuiltins = append(allBuiltins, messages...)
	allBuiltins = append(allBuiltins, extras...)

	for _, name := range allBuiltins {
		if builtin := builtins.GetBuiltinByName(name); builtin != nil {
			builtinsList[name] = builtin
		}
	}
}
