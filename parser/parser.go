package parser

import (
	"fmt"
	"strconv"
	"strings"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/token"
)

var precedences = map[token.TokenType]int{
	token.OR:        OR,
	token.AND:       AND,
	token.BIT_OR:    BIT_OR,
	token.BIT_XOR:   BIT_XOR,
	token.BIT_AND:   BIT_AND,
	token.EQUAL:     EQUALS,
	token.NOT_EQUAL: EQUALS,
	token.LT:        LESSGREATER,
	token.GT:        LESSGREATER,
	token.LTE:       LESSGREATER,
	token.GTE:       LESSGREATER,
	token.SHIFT_L:   SHIFT,
	token.SHIFT_R:   SHIFT,
	token.DOTDOT:    RANGE,
	token.PLUS:      SUM,
	token.MINUS:     SUM,
	token.SLASH:     PRODUCT,
	token.ASTERISK:  PRODUCT,
	token.MODULE:    PRODUCT,
	token.POWER:     POWER,
	token.LPAREN:    CALL,
	token.LBRACKET:  INDEX,
	token.DOT:       INDEX,
}

const (
	_ int = iota
	LOWEST
	OR
	AND
	EQUALS
	LESSGREATER
	BIT_OR
	BIT_XOR
	BIT_AND
	SHIFT
	RANGE
	SUM
	PRODUCT
	POWER
	PREFIX
	CALL
	INDEX
)

type (
	prefixParseFct func() ast.Expression
	infixParseFct  func(ast.Expression) ast.Expression
)

type Parser struct {
	l *lexer.Lexer

	errors []string

	curToken   token.Token
	peekToken  token.Token
	peekToken2 token.Token

	prefixParseFcts map[token.TokenType]prefixParseFct
	infixParseFcts  map[token.TokenType]infixParseFct
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFcts = make(map[token.TokenType]prefixParseFct)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.BIT_NOT, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseDictLiteral)
	p.registerPrefix(token.FOR, p.parseForLoopExpression)
	p.registerPrefix(token.ASYNC, p.parseAsyncFunctionLiteral)
	p.registerPrefix(token.AWAIT, p.parseAwaitExpression)
	p.registerPrefix(token.SPAWN, p.parseSpawnExpression)
	p.registerPrefix(token.TRY, p.parseTryExpression)
	p.registerPrefix(token.MATCH, p.parseMatchExpression)
	p.registerPrefix(token.BREAK, p.parseBreakWithoutLoopContext)
	p.registerPrefix(token.CONTINUE, p.parseContinueWithoutLoopContext)

	p.infixParseFcts = make(map[token.TokenType]infixParseFct)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.MODULE, p.parseInfixExpression)
	p.registerInfix(token.POWER, p.parseInfixExpression)
	p.registerInfix(token.EQUAL, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQUAL, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LTE, p.parseInfixExpression)
	p.registerInfix(token.GTE, p.parseInfixExpression)
	p.registerInfix(token.AND, p.parseInfixExpression)
	p.registerInfix(token.OR, p.parseOrExpression)
	p.registerInfix(token.BIT_AND, p.parseInfixExpression)
	p.registerInfix(token.BIT_OR, p.parseInfixExpression)
	p.registerInfix(token.BIT_XOR, p.parseInfixExpression)
	p.registerInfix(token.SHIFT_L, p.parseInfixExpression)
	p.registerInfix(token.SHIFT_R, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.DOT, p.parseAttributeAccess)

	p.nextToken()
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.peekToken2
	p.peekToken2 = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.VAR:
		return p.parseVarStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.CONST:
		return p.parseConstStatement()
	case token.STRUCT:
		return p.parseStructStatement()
	case token.ENUM:
		return p.parseEnumStatement()
	case token.TYPE:
		return p.parseTypeAliasStatement()
	case token.PUB:
		return p.parsePublicStatement()
	case token.EXTERN:
		return p.parseExternBlockStatement(false)
	case token.UNSAFE:
		return p.parseUnsafeStatement()
	case token.FUNCTION:
		if p.peekTokenIs(token.IDENT) {
			return p.parseNamedFunctionStatement(false)
		}
		return p.parseExpressionStatement()
	case token.IDENT:
		if p.peekTokenIs(token.ASSIGN) {
			return p.parseAssignStatement()
		}
		fallthrough
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parsePublicStatement() ast.Statement {
	pubToken := p.curToken
	p.nextToken()
	switch p.curToken.Type {
	case token.VAR:
		stmt := p.parseVarStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case token.CONST:
		stmt := p.parseConstStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case token.STRUCT:
		stmt := p.parseStructStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case token.ENUM:
		stmt := p.parseEnumStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case token.TYPE:
		stmt := p.parseTypeAliasStatement()
		if stmt != nil {
			stmt.Public = true
		}
		return stmt
	case token.FUNCTION:
		return p.parseNamedFunctionStatementWithToken(true, pubToken)
	case token.EXTERN:
		return p.parseExternBlockStatement(true)
	default:
		p.errors = append(p.errors, fmt.Sprintf("pub expects var, const, fct, struct, enum, type or extern, got %s", p.tokenDebugString(p.curToken)))
		return nil
	}
}

func (p *Parser) parseNamedFunctionStatement(public bool) *ast.VarStatement {
	return p.parseNamedFunctionStatementWithToken(public, p.curToken)
}

func (p *Parser) parseNamedFunctionStatementWithToken(public bool, declarationToken token.Token) *ast.VarStatement {
	functionToken := p.curToken
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	params := p.parseFunctionParameters()
	if params == nil {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	body := p.parseBlockStatement()
	fn := &ast.FunctionLiteral{Token: functionToken, Parameters: params, Body: body, Name: name.Value}
	varToken := token.Token{Type: token.VAR, Literal: "var", Pos: declarationToken.Pos}
	return &ast.VarStatement{Token: varToken, Public: public, Name: name, Value: fn}
}

func (p *Parser) parseUnsafeStatement() *ast.UnsafeStatement {
	stmt := &ast.UnsafeStatement{Token: p.curToken}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseExternBlockStatement(public bool) *ast.ExternBlockStatement {
	stmt := &ast.ExternBlockStatement{Token: p.curToken, Public: public, Functions: []*ast.ExternFunction{}}
	if !p.expectPeek(token.STRING) {
		return nil
	}
	stmt.ABI = p.curToken.Literal
	if stmt.ABI != "C" {
		p.errors = append(p.errors, fmt.Sprintf("unsupported extern ABI %q; only C is available", stmt.ABI))
	}
	if p.peekTokenIs(token.FROM) {
		p.nextToken()
		if !p.expectPeek(token.STRING) {
			return nil
		}
		stmt.Link = p.curToken.Literal
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.SEMICOLON) || p.curTokenIs(token.COMMA) {
			p.nextToken()
			continue
		}
		if !p.curTokenIs(token.FUNCTION) {
			p.errors = append(p.errors, fmt.Sprintf("extern block expects fct declarations, got %s", p.tokenDebugString(p.curToken)))
			return nil
		}
		fn := p.parseExternFunction()
		if fn == nil {
			return nil
		}
		stmt.Functions = append(stmt.Functions, fn)
		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExternFunction() *ast.ExternFunction {
	fn := &ast.ExternFunction{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	fn.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	fn.CName = fn.Name.Value
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	fn.Parameters = []*ast.ExternParam{}
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
	} else {
		p.nextToken()
		for {
			if !p.curTokenIs(token.IDENT) {
				p.errors = append(p.errors, fmt.Sprintf("extern parameter name must be an identifier, got %s", p.tokenDebugString(p.curToken)))
				return nil
			}
			param := &ast.ExternParam{Token: p.curToken, Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}
			if !p.expectPeek(token.COLON) {
				return nil
			}
			p.nextToken()
			param.Type = p.parseExternType()
			if param.Type == nil {
				return nil
			}
			fn.Parameters = append(fn.Parameters, param)
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
				p.nextToken()
				continue
			}
			if !p.expectPeek(token.RPAREN) {
				return nil
			}
			break
		}
	}
	if !p.expectPeek(token.ARROW) {
		return nil
	}
	p.nextToken()
	fn.ReturnType = p.parseExternType()
	if fn.ReturnType == nil {
		return nil
	}
	if p.peekTokenIs(token.AS) {
		p.nextToken()
		if !p.expectPeek(token.STRING) {
			return nil
		}
		fn.CName = p.curToken.Literal
	}
	return fn
}

func (p *Parser) parseExternType() *ast.ExternType {
	if !p.curTokenIs(token.IDENT) {
		p.errors = append(p.errors, fmt.Sprintf("expected extern type, got %s", p.tokenDebugString(p.curToken)))
		return nil
	}
	t := &ast.ExternType{Name: p.curToken.Literal}
	if t.Name != "callback" {
		return t
	}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	t.CallbackParams = []*ast.ExternType{}
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
	} else {
		p.nextToken()
		for {
			param := p.parseExternType()
			if param == nil {
				return nil
			}
			t.CallbackParams = append(t.CallbackParams, param)
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
				p.nextToken()
				continue
			}
			if !p.expectPeek(token.RPAREN) {
				return nil
			}
			break
		}
	}
	if !p.expectPeek(token.ARROW) {
		return nil
	}
	p.nextToken()
	t.CallbackReturn = p.parseExternType()
	return t
}

func (p *Parser) parseConstStatement() *ast.ConstStatement {
	stmt := &ast.ConstStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseTypeAliasStatement() *ast.TypeAliasStatement {
	stmt := &ast.TypeAliasStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	targetToken := p.curToken
	targetName := p.parseQualifiedTypeName()
	stmt.Target = &ast.Identifier{Token: targetToken, Value: targetName}
	stmt.Target.Token.Literal = targetName
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseStructStatement() *ast.StructStatement {
	stmt := &ast.StructStatement{Token: p.curToken, Fields: []*ast.StructField{}, Methods: []*ast.StructMethod{}}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.SEMICOLON, token.COMMA:
			p.nextToken()
			continue
		case token.FUNCTION:
			method := p.parseStructMethod()
			if method == nil {
				return nil
			}
			stmt.Methods = append(stmt.Methods, method)
		case token.ASYNC:
			if !p.expectPeek(token.FUNCTION) {
				return nil
			}
			method := p.parseStructMethod()
			if method == nil {
				return nil
			}
			method.Function.Async = true
			stmt.Methods = append(stmt.Methods, method)
		default:
			if !p.curTokenIs(token.IDENT) {
				p.errors = append(p.errors, fmt.Sprintf("expected struct field or method, got %s", p.tokenDebugString(p.curToken)))
				return nil
			}
			field := &ast.StructField{Token: p.curToken, Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}
			if p.peekTokenIs(token.COLON) {
				p.nextToken()
				if !p.expectPeek(token.IDENT) {
					return nil
				}
				field.TypeName = p.parseQualifiedTypeName()
			}
			stmt.Fields = append(stmt.Fields, field)
			if p.peekTokenIs(token.SEMICOLON) || p.peekTokenIs(token.COMMA) {
				p.nextToken()
			}
		}
		p.nextToken()
	}
	stmt.RBraceToken = p.curToken
	return stmt
}

func (p *Parser) parseQualifiedTypeName() string {
	name := p.curToken.Literal
	if p.peekTokenIs(token.DOT) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return name
		}
		name += "." + p.curToken.Literal
	}
	return name
}

func (p *Parser) parseStructMethod() *ast.StructMethod {
	method := &ast.StructMethod{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	method.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	params := p.parseFunctionParameters()
	if params == nil {
		return nil
	}
	selfToken := token.Token{Type: token.IDENT, Literal: "self", Pos: method.Token.Pos}
	params = append([]*ast.Identifier{{Token: selfToken, Value: "self"}}, params...)
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	body := p.parseBlockStatement()
	method.Function = &ast.FunctionLiteral{Token: method.Token, Parameters: params, Body: body, Name: method.Name.Value}
	return method
}

func (p *Parser) parseEnumStatement() *ast.EnumStatement {
	stmt := &ast.EnumStatement{Token: p.curToken, Members: []*ast.Identifier{}}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.SEMICOLON) || p.curTokenIs(token.COMMA) {
			p.nextToken()
			continue
		}
		if !p.curTokenIs(token.IDENT) {
			p.errors = append(p.errors, fmt.Sprintf("expected enum member, got %s", p.tokenDebugString(p.curToken)))
			return nil
		}
		stmt.Members = append(stmt.Members, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		if p.peekTokenIs(token.SEMICOLON) || p.peekTokenIs(token.COMMA) {
			p.nextToken()
		}
		p.nextToken()
	}
	stmt.RBraceToken = p.curToken
	return stmt
}

func (p *Parser) parseMatchExpression() ast.Expression {
	expr := &ast.MatchExpression{Token: p.curToken, Cases: []*ast.MatchCase{}}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken()
	expr.Value = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.SEMICOLON) {
			p.nextToken()
			continue
		}
		switch p.curToken.Type {
		case token.CASE:
			caseToken := p.curToken
			p.nextToken()
			pattern := p.parseExpression(LOWEST)
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			body := p.parseBlockStatement()
			expr.Cases = append(expr.Cases, &ast.MatchCase{Token: caseToken, Pattern: pattern, Body: body})
			p.nextToken()
		case token.ELSE:
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			expr.Default = p.parseBlockStatement()
			p.nextToken()
		default:
			p.errors = append(p.errors, fmt.Sprintf("expected case or else in match, got %s", p.tokenDebugString(p.curToken)))
			return nil
		}
	}
	expr.RBraceToken = p.curToken
	return expr
}

func (p *Parser) parseImportStatement() *ast.ImportStatement {
	stmt := &ast.ImportStatement{Token: p.curToken}

	if !p.expectPeek(token.STRING) {
		return nil
	}

	stmt.Path = &ast.StringLiteral{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}

	if p.peekTokenIs(token.AS) {
		p.nextToken()
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseAssignStatement() *ast.AssignStatement {
	stmt := &ast.AssignStatement{Token: p.peekToken}

	name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken()
	p.nextToken()

	stmt.Name = name
	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseVarStatement() *ast.VarStatement {
	stmt := &ast.VarStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if fl, ok := stmt.Value.(*ast.FunctionLiteral); ok {
		fl.Name = stmt.Name.Value
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}

	p.peekError(t)
	return false
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf(
		"expected next token to be %s, got %s at line %d, col %d",
		t,
		p.tokenDebugString(p.peekToken),
		p.peekToken.Pos.Line,
		p.peekToken.Pos.Col,
	)
	p.errors = append(p.errors, msg)
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	if p.curTokenIs(token.SEMICOLON) {
		return stmt
	}

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseAsyncFunctionLiteral() ast.Expression {
	if !p.expectPeek(token.FUNCTION) {
		return nil
	}

	lit := p.parseFunctionLiteral()
	fn, ok := lit.(*ast.FunctionLiteral)
	if !ok {
		return lit
	}

	fn.Async = true
	return fn
}

func (p *Parser) parseAwaitExpression() ast.Expression {
	expr := &ast.AwaitExpression{Token: p.curToken}
	p.nextToken()
	expr.Value = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseSpawnExpression() ast.Expression {
	expr := &ast.SpawnExpression{Token: p.curToken}
	p.nextToken()
	expr.Value = p.parseExpression(PREFIX)
	if _, ok := expr.Value.(*ast.CallExpression); !ok {
		p.errors = append(p.errors, "spawn expects a function call, for example: spawn work()")
	}
	return expr
}

func (p *Parser) parseTryExpression() ast.Expression {
	expr := &ast.TryExpression{Token: p.curToken}
	p.nextToken()
	expr.Value = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseOrExpression(left ast.Expression) ast.Expression {
	if p.peekTokenIs(token.LBRACE) {
		expr := &ast.ErrorHandlerExpression{
			Token: p.curToken,
			Left:  left,
		}

		p.nextToken()
		expr.Handler = p.parseBlockStatement()
		return expr
	}

	if p.peekTokenIs(token.IDENT) && p.peekSecondTokenIs(token.LBRACE) {
		expr := &ast.ErrorHandlerExpression{
			Token: p.curToken,
			Left:  left,
		}

		p.nextToken()
		expr.ErrorIdent = &ast.Identifier{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expr.Handler = p.parseBlockStatement()
		return expr
	}

	return p.parseInfixExpression(left)
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fct prefixParseFct) {
	p.prefixParseFcts[tokenType] = fct
}

func (p *Parser) registerInfix(tokenType token.TokenType, fct infixParseFct) {
	p.infixParseFcts[tokenType] = fct
}

func (p *Parser) parseExpressionStatement() ast.Statement {
	startToken := p.curToken
	expression := p.parseExpression(LOWEST)

	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken()
		assignToken := p.curToken
		p.nextToken()
		value := p.parseExpression(LOWEST)

		var stmt ast.Statement
		switch target := expression.(type) {
		case *ast.IndexExpression:
			stmt = &ast.IndexAssignStatement{Token: assignToken, Target: target, Value: value}
		case *ast.AttributeAccess:
			stmt = &ast.AttributeAssignStatement{Token: assignToken, Target: target, Value: value}
		default:
			p.errors = append(p.errors, fmt.Sprintf("invalid assignment target %T: only identifiers, indexed values and fields can be assigned", expression))
			return &ast.ExpressionStatement{Token: startToken, Expression: expression}
		}

		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
		return stmt
	}

	stmt := &ast.ExpressionStatement{Token: startToken, Expression: expression}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFcts[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFctError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFcts[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	normalized := strings.ReplaceAll(p.curToken.Literal, "_", "")
	number, fixedType, bits, signed := splitFixedIntegerSuffix(normalized)

	if fixedType == "" {
		value, err := strconv.ParseInt(number, 0, 64)
		if err != nil {
			p.addIntegerLiteralError(p.curToken.Literal)
			return nil
		}
		lit.Value = value
		return lit
	}

	lit.FixedType = fixedType
	if signed {
		value, err := strconv.ParseInt(number, 0, bits)
		if err != nil {
			p.addIntegerLiteralError(p.curToken.Literal)
			return nil
		}
		lit.RawValue = uint64(value)
		return lit
	}

	value, err := strconv.ParseUint(number, 0, bits)
	if err != nil {
		p.addIntegerLiteralError(p.curToken.Literal)
		return nil
	}
	lit.RawValue = value
	return lit
}

func splitFixedIntegerSuffix(literal string) (number string, fixedType string, bits int, signed bool) {
	types := []struct {
		suffix string
		bits   int
		signed bool
	}{
		{"u16", 16, false}, {"u32", 32, false}, {"u64", 64, false},
		{"i16", 16, true}, {"i32", 32, true}, {"i64", 64, true},
		{"u8", 8, false}, {"i8", 8, true},
	}

	for _, candidate := range types {
		if strings.HasSuffix(literal, candidate.suffix) {
			return strings.TrimSuffix(literal, candidate.suffix), candidate.suffix, candidate.bits, candidate.signed
		}
	}

	return literal, "", 64, true
}

func (p *Parser) addIntegerLiteralError(literal string) {
	p.errors = append(p.errors, fmt.Sprintf("could not parse %q as integer", literal))
}

func (p *Parser) noPrefixParseFctError(t token.TokenType) {
	msg := fmt.Sprintf(
		"no prefix parse function for %s at line %d, col %d",
		p.tokenDebugString(p.curToken),
		p.curToken.Pos.Line,
		p.curToken.Pos.Col,
	)
	p.errors = append(p.errors, msg)
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)
	return expression
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()

	if expression.Token.Type == token.POWER {
		expression.Right = p.parseExpression(precedence - 1)
	} else {
		expression.Right = p.parseExpression(precedence)
	}

	return expression
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	block.RBraceToken = p.curToken
	return block
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()
	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	return array
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}
	list = append(list, first)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()

		exp := p.parseExpression(LOWEST)
		if exp == nil {
			return nil
		}
		list = append(list, exp)
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseDictLiteral() ast.Expression {
	dict := &ast.DictLiteral{Token: p.curToken}
	dict.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()

		key := p.parseExpression(LOWEST)

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		dict.Pairs[key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return dict
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	float := &ast.FloatLiteral{Token: p.curToken}

	normalized := strings.ReplaceAll(p.curToken.Literal, "_", "")
	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as float", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	float.Value = value
	return float
}

func (p *Parser) parseAttributeAccess(left ast.Expression) ast.Expression {
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	property := &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}

	return &ast.AttributeAccess{
		Object:   left,
		Property: property,
	}
}

func (p *Parser) parseForLoopExpression() ast.Expression {
	curToken := p.curToken

	if p.peekTokenIs(token.LBRACE) {
		return p.parseForEverLoopExpression(curToken)
	}

	if p.peekTokenIs(token.LPAREN) {
		return p.parseCForLoopExpression(curToken)
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	variable := p.curToken.Literal

	if p.peekTokenIs(token.COMMA) {
		return p.parseForEachMapExpression(curToken, variable)
	}

	return p.parseForEachArrayOrRangeExpression(curToken, variable)
}

func (p *Parser) parseForEverLoopExpression(curToken token.Token) ast.Expression {
	p.registerPrefix(token.BREAK, p.parseBreakExpression)
	p.registerPrefix(token.CONTINUE, p.parseContinueExpression)

	loop := &ast.ForEverLoop{Token: curToken}

	p.expectPeek(token.LBRACE)
	loop.Block = p.parseBlockStatement()

	p.registerPrefix(token.BREAK, p.parseBreakWithoutLoopContext)
	p.registerPrefix(token.CONTINUE, p.parseContinueWithoutLoopContext)

	return loop
}

func (p *Parser) parseCForLoopExpression(curToken token.Token) ast.Expression {
	var result ast.Expression

	p.registerPrefix(token.BREAK, p.parseBreakExpression)
	p.registerPrefix(token.CONTINUE, p.parseContinueExpression)

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	var init ast.Expression
	var cond ast.Expression
	var update ast.Expression

	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) {
		init = p.parseExpression(LOWEST)
		p.nextToken()
	}

	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) {
		cond = p.parseExpression(LOWEST)
		p.nextToken()
	}

	p.nextToken()
	if !p.curTokenIs(token.RPAREN) {
		update = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	if init == nil && cond == nil && update == nil {
		loop := &ast.ForEverLoop{Token: curToken}
		loop.Block = p.parseBlockStatement()
		result = loop
	} else {
		loop := &ast.ForLoop{Token: curToken, Init: init, Cond: cond, Update: update}
		loop.Block = p.parseBlockStatement()
		result = loop
	}

	p.registerPrefix(token.BREAK, p.parseBreakWithoutLoopContext)
	p.registerPrefix(token.CONTINUE, p.parseContinueWithoutLoopContext)

	return result
}

func (p *Parser) parseForEachMapExpression(curToken token.Token, variable string) ast.Expression {
	p.registerPrefix(token.BREAK, p.parseBreakExpression)
	p.registerPrefix(token.CONTINUE, p.parseContinueExpression)

	loop := &ast.ForEachMapLoop{Token: curToken}
	loop.Key = variable

	if !p.expectPeek(token.COMMA) {
		return nil
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	loop.Value = p.curToken.Literal

	if !p.expectPeek(token.IN) {
		return nil
	}

	p.nextToken()
	loop.X = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.WHERE) {
		p.nextToken()
		p.nextToken()
		loop.Cond = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	loop.Block = p.parseBlockStatement()

	p.registerPrefix(token.BREAK, p.parseBreakWithoutLoopContext)
	p.registerPrefix(token.CONTINUE, p.parseContinueWithoutLoopContext)

	return loop
}

func (p *Parser) parseForEachArrayOrRangeExpression(curToken token.Token, variable string) ast.Expression {
	p.registerPrefix(token.BREAK, p.parseBreakExpression)
	p.registerPrefix(token.CONTINUE, p.parseContinueExpression)

	if !p.expectPeek(token.IN) {
		return nil
	}

	p.nextToken()
	aValue1 := p.parseExpression(LOWEST)

	var aValue2 ast.Expression
	isRange := false

	if p.peekTokenIs(token.DOTDOT) {
		isRange = true
		p.nextToken()
		p.nextToken()
		aValue2 = p.parseExpression(RANGE)
	}

	var aCond ast.Expression
	if p.peekTokenIs(token.WHERE) {
		p.nextToken()
		p.nextToken()
		aCond = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	aBlock := p.parseBlockStatement()

	var result ast.Expression
	if !isRange {
		result = &ast.ForEachArrayLoop{
			Token: curToken,
			Var:   variable,
			Value: aValue1,
			Cond:  aCond,
			Block: aBlock,
		}
	} else {
		result = &ast.ForEachDotRange{
			Token:    curToken,
			Var:      variable,
			StartIdx: aValue1,
			EndIdx:   aValue2,
			Cond:     aCond,
			Block:    aBlock,
		}
	}

	p.registerPrefix(token.BREAK, p.parseBreakWithoutLoopContext)
	p.registerPrefix(token.CONTINUE, p.parseContinueWithoutLoopContext)

	return result
}

func (p *Parser) parseBreakWithoutLoopContext() ast.Expression {
	msg := fmt.Sprintf("Syntax Error:%v- 'break' outside of loop context", p.curToken.Pos)
	p.errors = append(p.errors, msg)

	return p.parseBreakExpression()
}

func (p *Parser) parseBreakExpression() ast.Expression {
	return &ast.BreakExpression{Token: p.curToken}
}

func (p *Parser) parseContinueWithoutLoopContext() ast.Expression {
	msg := fmt.Sprintf("Syntax Error:%v- 'continue' outside of loop context", p.curToken.Pos)
	p.errors = append(p.errors, msg)

	return p.parseContinueExpression()
}

func (p *Parser) parseContinueExpression() ast.Expression {
	return &ast.ContinueExpression{Token: p.curToken}
}

func (p *Parser) peekSecondTokenIs(t token.TokenType) bool {
	return p.peekToken2.Type == t
}

func (p *Parser) tokenDebugString(tok token.Token) string {
	if tok.Literal == "" {
		return string(tok.Type)
	}
	return fmt.Sprintf("%s(%q)", tok.Type, tok.Literal)
}
