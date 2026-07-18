package builtins

import (
	"fmt"
	"zumbra/numeric"
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

	// files
	{
		"createCsv", CreateCsvBuiltin(),
	},
	{
		"createDoc", CreateDocBuiltin(),
	},
	{
		"createFile", CreateFileBuiltin(),
	},
	{
		"createPdf", CreatePdfBuiltin(),
	},
	{
		"createTxt", CreateTxtBuiltin(),
	},

	// rest
	{
		"restDelete", RestDeleteBuiltin(),
	},
	{
		"restGet", RestGetBuiltin(),
	},
	{
		"restPatch", RestPatchBuiltin(),
	},
	{
		"restPost", RestPostBuiltin(),
	},
	{
		"restPut", RestPutBuiltin(),
	},

	// utils
	{
		"switchCase", SwitchCaseBuiltin(),
	},

	{
		"postgresConnection", PostgresConnectionBuiltin(),
	},
	{
		"postgresExec", PostgresExecBuiltin(),
	},
	{
		"postgresQuery", PostgresQueryBuiltin(),
	},
	{
		"redisConnection", RedisConnectionBuiltin(),
	},
	{
		"redisSet", RedisSetBuiltin(),
	},
	{
		"redisGet", RedisGetBuiltin(),
	},
	{
		"redisDel", RedisDelBuiltin(),
	},
	{
		"supabaseConnection", SupabaseConnectionBuiltin(),
	},
	{
		"supabaseSelect", SupabaseSelectBuiltin(),
	},
	{
		"supabaseInsert", SupabaseInsertBuiltin(),
	},
	{
		"supabaseQuery", SupabaseQueryBuiltin(),
	},
	{
		"supabaseUpdate", SupabaseUpdateBuiltin(),
	},
	{
		"supabaseDelete", SupabaseDeleteBuiltin(),
	},
	{
		"supabaseUpsert", SupabaseUpsertBuiltin(),
	},
	{
		"supabaseRpc", SupabaseRpcBuiltin(),
	},
	{
		"supabaseCount", SupabaseCountBuiltin(),
	},
	{
		"supabaseSingle", SupabaseSingleBuiltin(),
	},
	{
		"supabaseStorageUpload", SupabaseStorageUploadBuiltin(),
	},
	{
		"supabaseStorageDelete", SupabaseStorageDeleteBuiltin(),
	},
	{
		"supabaseStoragePublicUrl", SupabaseStoragePublicUrlBuiltin(),
	},
	{
		"supabaseStorageSignedUrl", SupabaseStorageSignedUrlBuiltin(),
	},
	{
		"supabaseStorageDownload", SupabaseStorageDownloadBuiltin(),
	},
	{
		"supabaseAuthSignUp", SupabaseAuthSignUpBuiltin(),
	},
	{
		"supabaseAuthSignIn", SupabaseAuthSignInBuiltin(),
	},
	{"error", &ErrorBuiltin},
	{"panic", &PanicBuiltin},

	// fixed-width integers
	{
		"u8", FixedIntegerConversionBuiltin(object.FixedU8),
	},
	{
		"u16", FixedIntegerConversionBuiltin(object.FixedU16),
	},
	{
		"u32", FixedIntegerConversionBuiltin(object.FixedU32),
	},
	{
		"u64", FixedIntegerConversionBuiltin(object.FixedU64),
	},
	{
		"i8", FixedIntegerConversionBuiltin(object.FixedI8),
	},
	{
		"i16", FixedIntegerConversionBuiltin(object.FixedI16),
	},
	{
		"i32", FixedIntegerConversionBuiltin(object.FixedI32),
	},
	{
		"i64", FixedIntegerConversionBuiltin(object.FixedI64),
	},
	{
		"wrapAdd", FixedArithmeticBuiltin(numeric.Wrapping, "+", "wrapAdd"),
	},
	{
		"wrapSub", FixedArithmeticBuiltin(numeric.Wrapping, "-", "wrapSub"),
	},
	{
		"wrapMul", FixedArithmeticBuiltin(numeric.Wrapping, "*", "wrapMul"),
	},
	{
		"checkedAdd", FixedArithmeticBuiltin(numeric.Checked, "+", "checkedAdd"),
	},
	{
		"checkedSub", FixedArithmeticBuiltin(numeric.Checked, "-", "checkedSub"),
	},
	{
		"checkedMul", FixedArithmeticBuiltin(numeric.Checked, "*", "checkedMul"),
	},
	{
		"satAdd", FixedArithmeticBuiltin(numeric.Saturating, "+", "satAdd"),
	},
	{
		"satSub", FixedArithmeticBuiltin(numeric.Saturating, "-", "satSub"),
	},
	{
		"satMul", FixedArithmeticBuiltin(numeric.Saturating, "*", "satMul"),
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
