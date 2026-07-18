package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers / literals
	IDENT  = "IDENT"
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"

	// Keywords
	FUNCTION = "FUNCTION"
	VAR      = "VAR"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	WHILE    = "WHILE"
	IMPORT   = "IMPORT"
	ASYNC    = "ASYNC"
	AWAIT    = "AWAIT"
	TRY      = "TRY"
	FOR      = "FOR"
	IN       = "IN"
	WHERE    = "WHERE"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"

	// Operators
	ASSIGN     = "<<"
	EQUAL      = "=="
	NOT_EQUAL  = "!="
	PLUS       = "+"
	MINUS      = "-"
	BANG       = "!"
	ASTERISK   = "*"
	SLASH      = "/"
	MODULE     = "%"
	LT         = "<"
	GT         = ">"
	LTE        = "<="
	GTE        = ">="
	POWER      = "**"
	PLUSPLUS   = "++"
	MINUSMINUS = "--"
	DOT        = "."
	DOTDOT     = ".."

	// Logical
	OR  = "or"
	AND = "and"

	// Bitwise. Word operators keep Zumbra readable and avoid conflicting
	// with the existing << assignment syntax during the rebuild.
	BIT_AND = "band"
	BIT_OR  = "bor"
	BIT_XOR = "bxor"
	BIT_NOT = "bnot"
	SHIFT_L = "shl"
	SHIFT_R = "shr"

	// Delimiters
	COMMA     = ","
	COLON     = ":"
	SEMICOLON = ";"
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"
)

type Token struct {
	Pos     Position
	Type    TokenType
	Literal string
}

var keywords = map[string]TokenType{
	"fct":      FUNCTION,
	"var":      VAR,
	"true":     TRUE,
	"false":    FALSE,
	"if":       IF,
	"else":     ELSE,
	"return":   RETURN,
	"while":    WHILE,
	"import":   IMPORT,
	"async":    ASYNC,
	"await":    AWAIT,
	"try":      TRY,
	"and":      AND,
	"or":       OR,
	"band":     BIT_AND,
	"bor":      BIT_OR,
	"bxor":     BIT_XOR,
	"bnot":     BIT_NOT,
	"shl":      SHIFT_L,
	"shr":      SHIFT_R,
	"for":      FOR,
	"in":       IN,
	"where":    WHERE,
	"break":    BREAK,
	"continue": CONTINUE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
