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

	extras := []string{
		"date", "hashCode",
	}

	http := []string{
		"get", "html", "registerRoute", "server", "serveFile", "serveStatic",
	}

	ioUtils := []string{
		"input", "show",
	}

	jwt := []string{
		"jwtCreateToken", "jwtVerifyToken",
	}

	messages := []string{
		"sendEmail", "sendWhatsapp",
	}

	mysql := []string{
		"mysqlConnection", "mysqlCreateTable", "mysqlDeleteFromTable", "mysqlDropTable", "mysqlGetFromTable", "mysqlInsertIntoTable", "mysqlShowTables", "mysqlShowTableColumns", "mysqlUpdateIntoTable",
	}

	numbersUtils := []string{
		"bhaskara", "randomFloat", "randomInteger",
	}

	parsers := []string{
		"json_parse", "toBool", "toFloat", "toInt", "toString",
	}

	stringUtils := []string{
		"capitalize", "removeWhiteSpaces", "replace", "toLowercase", "toUppercase",
	}

	allBuiltins := append(arrays, dicts...)
	allBuiltins = append(allBuiltins, http...)
	allBuiltins = append(allBuiltins, parsers...)
	allBuiltins = append(allBuiltins, stringUtils...)
	allBuiltins = append(allBuiltins, numbersUtils...)
	allBuiltins = append(allBuiltins, ioUtils...)
	allBuiltins = append(allBuiltins, messages...)
	allBuiltins = append(allBuiltins, extras...)
	allBuiltins = append(allBuiltins, mysql...)
	allBuiltins = append(allBuiltins, jwt...)

	for _, name := range allBuiltins {
		if builtin := builtins.GetBuiltinByName(name); builtin != nil {
			builtinsList[name] = builtin
		}
	}
}
