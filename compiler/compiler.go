package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"zumbra/ast"
	"zumbra/code"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
	"zumbra/token"
)

type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

type Compiler struct {
	constants           []object.Object
	previousInstruction EmittedInstruction
	symbolTable         *SymbolTable
	scopes              []CompilationScope
	scopeIndex          int
	importedFiles       map[string]bool
	currentDir          string
	errorTempCounter    int
}

func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	symbolTable := NewSymbolTable()

	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	cwd, _ := os.Getwd()
	return &Compiler{
		constants:     []object.Object{},
		symbolTable:   symbolTable,
		scopes:        []CompilationScope{mainScope},
		scopeIndex:    0,
		importedFiles: map[string]bool{},
		currentDir:    cwd,
	}
}

func NewWithStateAndDir(s *SymbolTable, constants []object.Object, baseDir string) *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	return &Compiler{
		constants:     constants,
		symbolTable:   s,
		scopes:        []CompilationScope{mainScope},
		scopeIndex:    0,
		importedFiles: map[string]bool{},
		currentDir:    baseDir,
	}
}

func (c *Compiler) Compile(node ast.Node) error {

	switch node := node.(type) {
	case *ast.Program:
		for _, statement := range node.Statements {
			err := c.Compile(statement)
			if err != nil {
				return err
			}
		}

	case *ast.ExpressionStatement:
		switch node.Expression.(type) {
		case *ast.ForEachArrayLoop,
			*ast.ForEachDotRange,
			*ast.ForEachMapLoop,
			*ast.ForLoop,
			*ast.ForEverLoop:
			return c.Compile(node.Expression)

		default:
			err := c.Compile(node.Expression)
			if err != nil {
				return err
			}
			c.emit(code.OpPop)
		}

	case *ast.InfixExpression:

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case "%":
			c.emit(code.OpMod)
		case ">":
			c.emit(code.OpGreaterThan)
		case "<":
			c.emit(code.OpLessThan)
		case ">=":
			c.emit(code.OpGreaterThanOrEqual)
		case "<=":
			c.emit(code.OpLessThanOrEqual)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		case "and":
			c.emit(code.OpAnd)
		case "or":
			c.emit(code.OpOr)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}
	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))

	case *ast.FloatLiteral:
		float := &object.Float{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(float))

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))

	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}

	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.removeLastPop()
		}

		jumpPos := c.emit(code.OpJump, 9999)

		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Alternative == nil {
			c.emit(code.OpNull)

		} else {
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
		}

		afterAlternativePos := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternativePos)

	case *ast.BlockStatement:
		for _, statement := range node.Statements {
			err := c.Compile(statement)
			if err != nil {
				return err
			}
		}

	case *ast.VarStatement:
		symbol := c.symbolTable.Define(node.Name.Value)
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return fmt.Errorf("undefined variable %s", node.Value)
		}

		c.loadSymbol(symbol)

	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpArray, len(node.Elements))

	case *ast.DictLiteral:
		keys := []ast.Expression{}

		for k := range node.Pairs {
			keys = append(keys, k)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})

		for _, k := range keys {
			err := c.Compile(k)

			if err != nil {
				return err
			}
			err = c.Compile(node.Pairs[k])
			if err != nil {
				return err
			}
		}

		c.emit(code.OpDict, len(node.Pairs)*2)

	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Index)
		if err != nil {
			return err
		}

		c.emit(code.OpIndex)

	case *ast.FunctionLiteral:
		c.enterScope()

		if node.Name != "" {
			c.symbolTable.DefineFunctionName(node.Name)
		}

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value)
		}

		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.replaceLastPopWithReturn()
		}
		if !c.lastInstructionIs(code.OpReturnValue) {
			c.emit(code.OpReturn)
		}

		freeSymbols := c.symbolTable.FreeSymbols
		numLocals := c.symbolTable.numDefinitions
		instructions := c.leaveScope()

		for _, s := range freeSymbols {
			c.loadSymbol(s)
		}

		compiledFn := &object.CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Parameters),
		}
		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))

	case *ast.ReturnStatement:
		if node.ReturnValue == nil {
			c.emit(code.OpNull)
			c.emit(code.OpReturnValue)
			return nil
		}

		err := c.Compile(node.ReturnValue)
		if err != nil {
			return err
		}

		c.emit(code.OpReturnValue)

	case *ast.AwaitExpression:
		return c.Compile(node.Value)

	case *ast.TryExpression:
		return c.Compile(node.Value)

	case *ast.ErrorHandlerExpression:
		if err := c.Compile(node.Left); err != nil {
			return err
		}

		c.emit(code.OpDup)
		c.emit(code.OpIsError)

		jumpNotErrorPos := c.emit(code.OpJumpNotTruthy, 9999)

		if node.ErrorIdent != nil {
			tempName := c.nextErrorTempName()
			tempSymbol := c.symbolTable.Define(tempName)

			c.emit(code.OpDup)

			if tempSymbol.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, tempSymbol.Index)
			} else {
				c.emit(code.OpSetLocal, tempSymbol.Index)
			}

			rewriteErrorIdentInNode(node.Handler, node.ErrorIdent.Value, tempName)
		}

		if err := c.Compile(node.Handler); err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.removeLastPop()
		}

		if !c.lastInstructionIs(code.OpReturnValue) &&
			!c.lastInstructionIs(code.OpReturn) &&
			!c.lastInstructionIs(code.OpConstant) &&
			!c.lastInstructionIs(code.OpNull) &&
			!c.lastInstructionIs(code.OpGetLocal) &&
			!c.lastInstructionIs(code.OpGetGlobal) &&
			!c.lastInstructionIs(code.OpGetBuiltin) &&
			!c.lastInstructionIs(code.OpCall) &&
			!c.lastInstructionIs(code.OpIndex) &&
			!c.lastInstructionIs(code.OpGetAttr) &&
			!c.lastInstructionIs(code.OpDict) &&
			!c.lastInstructionIs(code.OpArray) {
			c.emit(code.OpNull)
		}

		jumpEndPos := c.emit(code.OpJump, 9999)

		afterHandlerPos := len(c.currentInstructions())
		c.changeOperand(jumpNotErrorPos, afterHandlerPos)

		afterEndPos := len(c.currentInstructions())
		c.changeOperand(jumpEndPos, afterEndPos)

	case *ast.CallExpression:
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}

		for _, a := range node.Arguments {
			err := c.Compile(a)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpCall, len(node.Arguments))

	case *ast.WhileStatement:
		err := c.compileWhile(node)
		if err != nil {
			return err
		}
	case *ast.ForEachArrayLoop:
		return c.compileForEachArrayLoop(node)

	case *ast.ForEachDotRange:
		return c.compileForEachDotRange(node)

	case *ast.ForEachMapLoop:
		return c.compileForEachMapLoop(node)

	case *ast.ForLoop:
		return c.compileCStyleForLoop(node)

	case *ast.ForEverLoop:
		return c.compileForeverLoop(node)

	case *ast.AssignStatement:
		err := c.compileAssign(node)
		if err != nil {
			return err
		}

	case *ast.ImportStatement:
		return c.compileImport(node)

	case *ast.AttributeAccess:
		if err := c.Compile(node.Object); err != nil {
			return err
		}
		idx := c.addConstant(&object.String{Value: node.Property.Value})
		c.emit(code.OpConstant, idx)
		c.emit(code.OpGetAttr)

	}

	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	instruction := code.Make(op, operands...)
	pos := c.addInstruction(instruction)
	c.setLastInstruction(op, pos)
	return pos
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	prev := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Pos: pos}

	c.scopes[c.scopeIndex].previousInstruction = prev
	c.scopes[c.scopeIndex].lastInstruction = last
}

func (c *Compiler) addInstruction(ins []byte) int {
	pos := len(c.currentInstructions())
	updatedInstructions := append(c.currentInstructions(), ins...)

	c.scopes[c.scopeIndex].instructions = updatedInstructions

	return pos
}

type EmittedInstruction struct {
	Opcode code.Opcode
	Pos    int
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	previous := c.scopes[c.scopeIndex].previousInstruction

	old := c.currentInstructions()
	new := old[:last.Pos]

	c.scopes[c.scopeIndex].instructions = new
	c.scopes[c.scopeIndex].lastInstruction = previous
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	ins := c.currentInstructions()
	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(pos int, operand int) {
	op := code.Opcode(c.currentInstructions()[pos])
	newInstruction := code.Make(op, operand)
	c.replaceInstruction(pos, newInstruction)
}

func NewWithState(s *SymbolTable, constants []object.Object) *Compiler {
	c := New()
	c.symbolTable = s
	c.constants = constants
	return c
}

func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

func (c *Compiler) enterScope() {
	scope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.scopes[c.scopeIndex].instructions

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer

	return instructions
}

func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}

	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) replaceLastPopWithReturn() {
	lastPos := c.scopes[c.scopeIndex].lastInstruction.Pos
	c.replaceInstruction(lastPos, code.Make(code.OpReturnValue))

	c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(code.OpGetBuiltin, s.Index)
	case FreeScope:
		c.emit(code.OpGetFree, s.Index)
	case FunctionScope:
		c.emit(code.OpCurrentClosure)
	}
}

func (c *Compiler) compileWhile(stmt *ast.WhileStatement) error {
	loopStartPos := len(c.currentInstructions())

	if err := c.Compile(stmt.Condition); err != nil {
		return err
	}

	jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

	if err := c.Compile(stmt.Body); err != nil {
		return err
	}

	c.emit(code.OpJump, loopStartPos)

	afterLoopPos := len(c.currentInstructions())
	c.changeOperand(jumpNotTruthyPos, afterLoopPos)

	return nil
}

func (c *Compiler) compileAssign(stmt *ast.AssignStatement) error {
	if err := c.Compile(stmt.Value); err != nil {
		return err
	}

	symbol, ok := c.symbolTable.Resolve(stmt.Name.Value)
	if !ok {
		return fmt.Errorf("undefined variable %s", stmt.Name.Value)
	}

	switch symbol.Scope {
	case GlobalScope:
		c.emit(code.OpSetGlobal, symbol.Index)
	case LocalScope:
		c.emit(code.OpSetLocal, symbol.Index)
	default:
		return fmt.Errorf("unsupported assignment target scope: %s", symbol.Scope)
	}

	return nil
}

func (c *Compiler) compileImport(stmt *ast.ImportStatement) error {
	path := stmt.Path.Value

	if c.importedFiles == nil {
		c.importedFiles = make(map[string]bool)
	}

	importFullPath := filepath.Join(c.currentDir, path)
	importFullPath = filepath.Clean(importFullPath)

	if c.importedFiles[importFullPath] {
		return nil
	}

	c.importedFiles[importFullPath] = true

	content, err := os.ReadFile(importFullPath)
	if err != nil {
		return fmt.Errorf("could not read imported file: %s", path)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return fmt.Errorf("could not parse imported file: %s", path)
	}

	oldDir := c.currentDir
	c.currentDir = filepath.Dir(importFullPath)

	err = c.Compile(program)
	c.currentDir = oldDir

	return err
}

// for i in array where cond { block }
// =========================
// Loops "for"
// =========================

// for i in array where cond { block }
func (c *Compiler) compileForEachArrayLoop(node *ast.ForEachArrayLoop) error {
	dummyTok := token.Token{}

	arrName := "__z_for_arr_" + node.Var
	idxName := "__z_for_idx_" + node.Var

	arrVarStmt := &ast.VarStatement{
		Token: dummyTok,
		Name: &ast.Identifier{
			Token: dummyTok,
			Value: arrName,
		},
		Value: node.Value,
	}

	idxVarStmt := &ast.VarStatement{
		Token: dummyTok,
		Name: &ast.Identifier{
			Token: dummyTok,
			Value: idxName,
		},
		Value: &ast.IntegerLiteral{
			Token: dummyTok,
			Value: 0,
		},
	}

	iterVarStmt := &ast.VarStatement{
		Token: dummyTok,
		Name: &ast.Identifier{
			Token: dummyTok,
			Value: node.Var,
		},
		Value: &ast.IntegerLiteral{
			Token: dummyTok,
			Value: 0,
		},
	}

	cond := &ast.InfixExpression{
		Token: dummyTok,
		Left: &ast.Identifier{
			Token: dummyTok,
			Value: idxName,
		},
		Operator: "<",
		Right: &ast.CallExpression{
			Token: dummyTok,
			Function: &ast.Identifier{
				Token: dummyTok,
				Value: "sizeOf",
			},
			Arguments: []ast.Expression{
				&ast.Identifier{
					Token: dummyTok,
					Value: arrName,
				},
			},
		},
	}

	assignIter := &ast.AssignStatement{
		Token: dummyTok,
		Name: &ast.Identifier{
			Token: dummyTok,
			Value: node.Var,
		},
		Value: &ast.IndexExpression{
			Token: dummyTok,
			Left: &ast.Identifier{
				Token: dummyTok,
				Value: arrName,
			},
			Index: &ast.Identifier{
				Token: dummyTok,
				Value: idxName,
			},
		},
	}

	inc := &ast.AssignStatement{
		Token: dummyTok,
		Name: &ast.Identifier{
			Token: dummyTok,
			Value: idxName,
		},
		Value: &ast.InfixExpression{
			Token: dummyTok,
			Left: &ast.Identifier{
				Token: dummyTok,
				Value: idxName,
			},
			Operator: "+",
			Right: &ast.IntegerLiteral{
				Token: dummyTok,
				Value: 1,
			},
		},
	}

	var loopBodyStatements []ast.Statement
	loopBodyStatements = append(loopBodyStatements, assignIter)

	if node.Cond != nil {
		ifExpr := &ast.IfExpression{
			Token:       dummyTok,
			Condition:   node.Cond,
			Consequence: node.Block,
			Alternative: nil,
		}

		loopBodyStatements = append(loopBodyStatements, &ast.ExpressionStatement{
			Token:      dummyTok,
			Expression: ifExpr,
		})
	} else {
		loopBodyStatements = append(loopBodyStatements, node.Block.Statements...)
	}

	loopBodyStatements = append(loopBodyStatements, inc)

	whileStmt := &ast.WhileStatement{
		Token:     dummyTok,
		Condition: cond,
		Body: &ast.BlockStatement{
			Token:      dummyTok,
			Statements: loopBodyStatements,
		},
	}

	if err := c.Compile(arrVarStmt); err != nil {
		return err
	}
	if err := c.Compile(idxVarStmt); err != nil {
		return err
	}
	if err := c.Compile(iterVarStmt); err != nil {
		return err
	}
	if err := c.Compile(whileStmt); err != nil {
		return err
	}

	return nil
}

func (c *Compiler) compileForEachDotRange(node *ast.ForEachDotRange) error {
	if err := c.Compile(node.StartIdx); err != nil {
		return err
	}
	if err := c.Compile(node.EndIdx); err != nil {
		return err
	}

	c.emit(code.OpNull)
	return nil
}

func (c *Compiler) compileForEachMapLoop(node *ast.ForEachMapLoop) error {
	if err := c.Compile(node.X); err != nil {
		return err
	}

	c.emit(code.OpNull)
	return nil
}

func (c *Compiler) compileCStyleForLoop(node *ast.ForLoop) error {
	if node.Init != nil {
		if err := c.Compile(node.Init); err != nil {
			return err
		}
	}
	if node.Cond != nil {
		if err := c.Compile(node.Cond); err != nil {
			return err
		}
	}
	if node.Update != nil {
		if err := c.Compile(node.Update); err != nil {
			return err
		}
	}

	c.emit(code.OpNull)
	return nil
}

func (c *Compiler) compileForeverLoop(node *ast.ForEverLoop) error {
	if err := c.Compile(node.Block); err != nil {
		return err
	}

	c.emit(code.OpNull)
	return nil
}

func (c *Compiler) nextErrorTempName() string {
	name := fmt.Sprintf("__z_or_err_%d", c.errorTempCounter)
	c.errorTempCounter++
	return name
}

func rewriteErrorIdentInNode(node ast.Node, from, to string) {
	switch n := node.(type) {
	case *ast.Identifier:
		if n.Value == from {
			n.Value = to
		}

	case *ast.Program:
		for _, stmt := range n.Statements {
			rewriteErrorIdentInNode(stmt, from, to)
		}

	case *ast.ExpressionStatement:
		if n.Expression != nil {
			rewriteErrorIdentInNode(n.Expression, from, to)
		}

	case *ast.BlockStatement:
		for _, stmt := range n.Statements {
			rewriteErrorIdentInNode(stmt, from, to)
		}

	case *ast.VarStatement:
		if n.Value != nil {
			rewriteErrorIdentInNode(n.Value, from, to)
		}

	case *ast.AssignStatement:
		if n.Value != nil {
			rewriteErrorIdentInNode(n.Value, from, to)
		}

	case *ast.ReturnStatement:
		if n.ReturnValue != nil {
			rewriteErrorIdentInNode(n.ReturnValue, from, to)
		}

	case *ast.IfExpression:
		if n.Condition != nil {
			rewriteErrorIdentInNode(n.Condition, from, to)
		}
		if n.Consequence != nil {
			rewriteErrorIdentInNode(n.Consequence, from, to)
		}
		if n.Alternative != nil {
			rewriteErrorIdentInNode(n.Alternative, from, to)
		}

	case *ast.PrefixExpression:
		if n.Right != nil {
			rewriteErrorIdentInNode(n.Right, from, to)
		}

	case *ast.InfixExpression:
		if n.Left != nil {
			rewriteErrorIdentInNode(n.Left, from, to)
		}
		if n.Right != nil {
			rewriteErrorIdentInNode(n.Right, from, to)
		}

	case *ast.CallExpression:
		if n.Function != nil {
			rewriteErrorIdentInNode(n.Function, from, to)
		}
		for _, arg := range n.Arguments {
			rewriteErrorIdentInNode(arg, from, to)
		}

	case *ast.ArrayLiteral:
		for _, el := range n.Elements {
			rewriteErrorIdentInNode(el, from, to)
		}

	case *ast.DictLiteral:
		for k, v := range n.Pairs {
			rewriteErrorIdentInNode(k, from, to)
			rewriteErrorIdentInNode(v, from, to)
		}

	case *ast.IndexExpression:
		if n.Left != nil {
			rewriteErrorIdentInNode(n.Left, from, to)
		}
		if n.Index != nil {
			rewriteErrorIdentInNode(n.Index, from, to)
		}

	case *ast.AttributeAccess:
		if n.Object != nil {
			rewriteErrorIdentInNode(n.Object, from, to)
		}

	case *ast.FunctionLiteral:
		if n.Body != nil {
			rewriteErrorIdentInNode(n.Body, from, to)
		}

	case *ast.AwaitExpression:
		if n.Value != nil {
			rewriteErrorIdentInNode(n.Value, from, to)
		}

	case *ast.TryExpression:
		if n.Value != nil {
			rewriteErrorIdentInNode(n.Value, from, to)
		}

	case *ast.ErrorHandlerExpression:
		if n.Left != nil {
			rewriteErrorIdentInNode(n.Left, from, to)
		}
		if n.Handler != nil {
			rewriteErrorIdentInNode(n.Handler, from, to)
		}
	}
}
