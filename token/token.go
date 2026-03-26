package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers
	IDENT  = "IDENT"
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"
	FOR    = "FOR"

	// Operators
	ASSIGN = "<<"

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

	// Logical
	OR       = "or"
	AND      = "and"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"
	WHERE    = "WHERE"
	IN       = "IN"
	DOTDOT   = "DOTDOT"

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
