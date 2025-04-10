package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF = "EOF"

	// Identifiers
	IDENT = "IDENT"
	INT = "INT"
	FLOAT = "FLOAT"

	// Operators
	ASSIGN = "<<"
	
	EQUAL= "=="
	NOT_EQUAL = "!="
	PLUS = "+"
	MINUS = "-"
	BANG = "!"
	ASTERISK = "*"
	SLASH = "/"
	MODULE = "%"
	LT = "<"
	GT = ">"
	POWER = "**"
	COMMENT = "c/"
	

	// Delimiters
	COMMA = ","
	SEMICOLON = ";"
	LPAREN = "("
	RPAREN = ")"
	LBRACE = "{"
	RBRACE = "}"

	// Keywords
	FUNCTION = "FUNCTION"
	VAR = "VAR"
	TRUE = "TRUE"
	FALSE = "FALSE"
	IF = "IF"
	ELSE = "ELSE"
	RETURN = "RETURN"
)

type Token struct {
	Type    TokenType
	Literal string
}

var keywords = map[string]TokenType{
	"fct":  FUNCTION,
	"var": VAR,
	"t": TRUE,
	"f": FALSE,
	"if": IF,
	"else": ELSE,
	"return": RETURN,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}