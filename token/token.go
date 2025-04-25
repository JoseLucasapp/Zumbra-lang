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
)

type Token struct {
	Type    TokenType
	Literal string
}

var keywords = map[string]TokenType{
	"fct":    FUNCTION,
	"var":    VAR,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
	"while":  WHILE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
