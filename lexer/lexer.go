package lexer

import (
	"zumbra/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte

	line int
	col  int
}

func New(input string) *Lexer {
	l := &Lexer{
		input: input,
		line:  1,
		col:   0,
	}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()

	pos := token.Position{
		Offset: l.position,
		Line:   l.line,
		Col:    l.col,
	}

	var tok token.Token

	switch l.ch {
	case '.':
		if l.peekChar() == '.' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.DOTDOT,
				Literal: string(ch) + string(l.ch),
			}
		} else {
			tok = token.Token{Pos: pos, Type: token.DOT, Literal: string(l.ch)}
		}

	case '<':
		if l.peekChar() == '<' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.ASSIGN,
				Literal: string(ch) + string(l.ch),
			}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.LTE,
				Literal: string(ch) + string(l.ch),
			}
		} else {
			tok = token.Token{Pos: pos, Type: token.LT, Literal: string(l.ch)}
		}

	case ';':
		tok = token.Token{Pos: pos, Type: token.SEMICOLON, Literal: string(l.ch)}

	case '(':
		tok = token.Token{Pos: pos, Type: token.LPAREN, Literal: string(l.ch)}

	case ')':
		tok = token.Token{Pos: pos, Type: token.RPAREN, Literal: string(l.ch)}

	case ',':
		tok = token.Token{Pos: pos, Type: token.COMMA, Literal: string(l.ch)}

	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.PLUSPLUS,
				Literal: string(ch) + string(l.ch),
			}
		} else {
			tok = token.Token{Pos: pos, Type: token.PLUS, Literal: string(l.ch)}
		}

	case '%':
		tok = token.Token{Pos: pos, Type: token.MODULE, Literal: string(l.ch)}

	case '{':
		tok = token.Token{Pos: pos, Type: token.LBRACE, Literal: string(l.ch)}

	case '}':
		tok = token.Token{Pos: pos, Type: token.RBRACE, Literal: string(l.ch)}

	case '-':
		if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.MINUSMINUS,
				Literal: string(ch) + string(l.ch),
			}
		} else {
			tok = token.Token{Pos: pos, Type: token.MINUS, Literal: string(l.ch)}
		}

	case '/':
		if l.peekChar() == '/' {
			l.readChar()
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			return l.NextToken()
		}
		tok = token.Token{Pos: pos, Type: token.SLASH, Literal: string(l.ch)}

	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.GTE,
				Literal: string(ch) + string(l.ch),
			}
		} else {
			tok = token.Token{Pos: pos, Type: token.GT, Literal: string(l.ch)}
		}

	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.EQUAL,
				Literal: string(ch) + string(l.ch),
			}
		} else {
			tok = token.Token{Pos: pos, Type: token.ILLEGAL, Literal: string(l.ch)}
		}

	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.NOT_EQUAL,
				Literal: string(ch) + string(l.ch),
			}
		} else {
			tok = token.Token{Pos: pos, Type: token.BANG, Literal: string(l.ch)}
		}

	case '*':
		if l.peekChar() == '*' {
			ch := l.ch
			l.readChar()
			tok = token.Token{
				Pos:     pos,
				Type:    token.POWER,
				Literal: string(ch) + string(l.ch),
			}
		} else {
			tok = token.Token{Pos: pos, Type: token.ASTERISK, Literal: string(l.ch)}
		}

	case '"':
		tok = token.Token{
			Pos:     pos,
			Type:    token.STRING,
			Literal: l.readString(),
		}
		l.readChar()
		return tok

	case '[':
		tok = token.Token{Pos: pos, Type: token.LBRACKET, Literal: string(l.ch)}

	case ']':
		tok = token.Token{Pos: pos, Type: token.RBRACKET, Literal: string(l.ch)}

	case ':':
		tok = token.Token{Pos: pos, Type: token.COLON, Literal: string(l.ch)}

	case 0:
		tok = token.Token{
			Pos:     pos,
			Type:    token.EOF,
			Literal: "",
		}

	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			return token.Token{
				Pos:     pos,
				Type:    token.LookupIdent(literal),
				Literal: literal,
			}
		}

		if isDigit(l.ch) {
			literal, tokenType := l.readNumber()
			return token.Token{
				Pos:     pos,
				Type:    tokenType,
				Literal: literal,
			}
		}

		tok = token.Token{Pos: pos, Type: token.ILLEGAL, Literal: string(l.ch)}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.position = l.readPosition
		l.ch = 0
		return
	}

	if l.ch == '\n' {
		l.line++
		l.col = 0
	}

	l.ch = l.input[l.readPosition]
	l.position = l.readPosition
	l.readPosition++
	l.col++
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isIdentChar(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() (string, token.TokenType) {
	position := l.position

	// Base-prefixed integers are intentionally simple: 0x for hexadecimal,
	// 0b for binary and 0o for octal. Invalid digits are left for the parser
	// to report with a precise integer-literal error.
	if l.ch == '0' {
		switch l.peekChar() {
		case 'x', 'X':
			l.readChar()
			l.readChar()
			for isNumberLiteralChar(l.ch) {
				l.readChar()
			}
			return l.input[position:l.position], token.INT
		case 'b', 'B':
			l.readChar()
			l.readChar()
			for isNumberLiteralChar(l.ch) {
				l.readChar()
			}
			return l.input[position:l.position], token.INT
		case 'o', 'O':
			l.readChar()
			l.readChar()
			for isNumberLiteralChar(l.ch) {
				l.readChar()
			}
			return l.input[position:l.position], token.INT
		}
	}

	for isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}

	tokenType := token.TokenType(token.INT)

	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = token.TokenType(token.FLOAT)
		l.readChar()

		for isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		}
	}

	if tokenType == token.INT {
		l.readFixedIntegerSuffix()
	}

	return l.input[position:l.position], tokenType
}

func (l *Lexer) readFixedIntegerSuffix() {
	remaining := l.input[l.position:]
	for _, suffix := range []string{"u16", "u32", "u64", "i16", "i32", "i64", "u8", "i8"} {
		if len(remaining) < len(suffix) || remaining[:len(suffix)] != suffix {
			continue
		}

		next := l.position + len(suffix)
		if next < len(l.input) && isIdentChar(l.input[next]) {
			return
		}

		for range suffix {
			l.readChar()
		}
		return
	}
}

func (l *Lexer) readString() string {
	position := l.position + 1

	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}

	return l.input[position:l.position]
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isNumberLiteralChar(ch byte) bool {
	return isIdentChar(ch)
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isIdentChar(ch byte) bool {
	return isLetter(ch) || isDigit(ch)
}
