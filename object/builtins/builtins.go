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
		"addToArrayStart", AddToArrayStartBuiltin(),
	},
	{
		"addToArrayEnd", AddToArrayEndBuiltin(),
	},
	{
		"addToDict", AddToDictBuiltin(),
	},
	{
		"allButFirst", AllButFirstBuiltin(),
	},
	{
		"bhaskara", BhaskaraBuiltin(),
	},
	{
		"capitalize", CapitalizeBuiltin(),
	},
	{
		"date", DateBuiltin(),
	},
	{
		"deleteFromDict", DeleteFromDictBuiltin(),
	},
	{
		"dictKeys", DictKeysBuiltin(),
	},
	{
		"dictValues", DictValuesBuiltin(),
	},
	{
		"dotenvLoad", loadEnvBuiltin(),
	},
	{
		"dotenvGet", getEnvBuiltin(),
	},
	{
		"first", ArrayFirstBuiltin(),
	},
	{
		"get", GetBuiltin(),
	},
	{
		"getFromDict", GetFromDictBuiltin(),
	},
	{
		"hashCode", HashCodeBuiltin(),
	},
	{
		"html", HtmlHandlerBuiltin(),
	},
	{
		"indexOf", IndexOfBuiltin(),
	},
	{
		"input", InputBuiltin(),
	},
	{
		"jsonParse", JsonParse(),
	},
	{
		"jwtCreateToken", createTokenBuiltin(),
	},
	{
		"jwtVerifyToken", verifyTokenBuiltin(),
	},
	{
		"last", ArrayLastBuiltin(),
	},
	{
		"max", MaxBuiltin(),
	},
	{
		"min", MinBuiltin(),
	},
	{
		"mysqlConnection", MySqlConnectionBuiltin(),
	},
	{
		"mysqlCreateTable", mysqlCreateTableBuiltin(),
	},
	{
		"mysqlDeleteFromTable", mysqlDeleteFromTableBuiltin(),
	},
	{
		"mysqlDropTable", mysqlDeleteTableBuiltin(),
	},
	{
		"mysqlGetFromTable", mysqlGetFromTableBuiltin(),
	},
	{
		"mysqlInsertIntoTable", mysqlInsertIntoTableBuiltin(),
	},
	{
		"mysqlShowTables", mysqlShowTablesBuiltin(),
	},
	{
		"mysqlShowTableColumns", mysqlShowTableColumnsBuiltin(),
	},
	{
		"mysqlUpdateIntoTable", mysqlUpdateIntoTableBuiltin(),
	},
	{
		"organize", OrganizeBuiltins(),
	},
	{
		"randomFloat", GenerateRandomFloatBuiltin(),
	},
	{
		"randomInteger", GenerateRandomIntegerBuiltin(),
	},
	{
		"registerRoute", RegisterRoutesBuiltin(),
	},
	{
		"removeFromArray", RemoveFromArrayBuiltin(),
	},

	{
		"removeWhiteSpaces", RemoveWhiteSpacesBuiltin(),
	},
	{
		"replace", ReplaceBuiltin(),
	},
	{
		"sendEmail", SendEmailBuiltin(),
	},
	{
		"sendWhatsapp", SendWhatsappBuiltin(),
	},
	{
		"server", CreateServerBuiltin(),
	},
	{
		"serveFile", ServeFileBuiltin(),
	},
	{
		"serveStatic", ServerStaticBuiltin(),
	},
	{
		"show", ShowBuiltin(),
	},
	{
		"sizeOf", SizeOfBuiltin(),
	},
	{
		"sum", SumBuiltin(),
	},
	{
		"toBool", ToBoolParserBuiltin(),
	},
	{
		"toFloat", ToFloatParserBuiltin(),
	},
	{
		"toInt", ToIntParserBuiltin(),
	},
	{
		"toLowercase", LowercaseBuiltin(),
	},
	{
		"toString", ToStringParserBuiltin(),
	},
	{
		"toUppercase", UppercaseBuiltin(),
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
