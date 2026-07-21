package types

import (
	"fmt"
	"zumbra/ast"
)

type scope struct {
	parent *scope
	values map[string]*Type
}

func newScope(parent *scope) *scope {
	return &scope{
		parent: parent,
		values: make(map[string]*Type),
	}
}

func (s *scope) define(name string, t *Type) {
	s.values[name] = t
}

func (s *scope) assign(name string, t *Type) bool {
	for cur := s; cur != nil; cur = cur.parent {
		if _, ok := cur.values[name]; ok {
			cur.values[name] = t
			return true
		}
	}
	return false
}

func (s *scope) resolve(name string) (*Type, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if t, ok := cur.values[name]; ok {
			return t, true
		}
	}
	return nil, false
}

type Checker struct {
	global *scope
	scope  *scope
	errors []error
}

func NewChecker() *Checker {
	global := newScope(nil)
	c := &Checker{
		global: global,
		scope:  global,
		errors: []error{},
	}
	c.installBuiltins()
	return c
}

func (c *Checker) installBuiltins() {
	names := []string{
		"show",
		"input",
		"sizeOf",
		"toInt",
		"toFloat",
		"toString",
		"toBool",
		"first",
		"last",
		"u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64",
		"wrapAdd", "wrapSub", "wrapMul",
		"checkedAdd", "checkedSub", "checkedMul",
		"satAdd", "satSub", "satMul",
		"bytes", "arrayOf", "slice", "fill",
		"readBytes", "writeBytes",
		"readU16LE", "readU16BE", "readU32LE", "readU32BE", "readU64LE", "readU64BE",
		"writeU16LE", "writeU16BE", "writeU32LE", "writeU32BE", "writeU64LE", "writeU64BE",
		"copyBytes", "bytesEqual", "sha256",
	}

	for _, name := range names {
		if t, ok := builtinType(name); ok {
			c.global.define(name, t)
		}
	}
}

func (c *Checker) ResetForNextRun() {
	c.scope = c.global
	c.errors = nil
}

func (c *Checker) Check(program *ast.Program) []error {
	c.ResetForNextRun()

	for _, stmt := range program.Statements {
		c.checkStatement(stmt)
	}

	return c.errors
}

func (c *Checker) addError(err error) {
	if err != nil {
		c.errors = append(c.errors, err)
	}
}

func (c *Checker) pushScope() {
	c.scope = newScope(c.scope)
}

func (c *Checker) popScope() {
	if c.scope.parent != nil {
		c.scope = c.scope.parent
	}
}

func isIdentifierUnknownParam(expr ast.Expression) (*ast.Identifier, bool) {
	id, ok := expr.(*ast.Identifier)
	if !ok || id == nil {
		return nil, false
	}
	return id, true
}

func mergeTypes(current *Type, wanted *Type) *Type {
	if current == nil || current.Kind == Unknown {
		return wanted
	}
	if wanted == nil || wanted.Kind == Unknown {
		return current
	}
	if Same(current, wanted) {
		return current
	}
	return Simple(Unknown)
}

func unifyReturnTypes(left *Type, right *Type) *Type {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	if left.Kind == Unknown || right.Kind == Unknown {
		return Simple(Unknown)
	}
	if Same(left, right) {
		return left
	}
	return Simple(Unknown)
}

func (c *Checker) unifyTypesOrError(context string, left *Type, right *Type) *Type {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	if left.Kind == Unknown || right.Kind == Unknown {
		return Simple(Unknown)
	}
	if Same(left, right) {
		return left
	}

	c.addError(fmt.Errorf("%s has incompatible types: %s and %s", context, left.Kind, right.Kind))
	return Simple(Unknown)
}

func (c *Checker) constrainIdentifier(expr ast.Expression, wanted *Type) {
	id, ok := isIdentifierUnknownParam(expr)
	if !ok || wanted == nil {
		return
	}

	current, exists := c.scope.resolve(id.Value)
	if !exists || current == nil {
		return
	}

	if current.Kind == Unknown {
		_ = c.scope.assign(id.Value, wanted)
		return
	}

	merged := mergeTypes(current, wanted)
	_ = c.scope.assign(id.Value, merged)
}

func (c *Checker) checkStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.VarStatement:
		var t *Type = Simple(Unknown)
		if s.Value != nil {
			t = c.inferExpression(s.Value)
		}
		if s.Name != nil {
			c.scope.define(s.Name.Value, t)
		}

	case *ast.AssignStatement:
		if s.Value != nil {
			valueType := c.inferExpression(s.Value)
			if s.Name != nil {
				if !c.scope.assign(s.Name.Value, valueType) {
					c.scope.define(s.Name.Value, valueType)
				}
			}
		}

	case *ast.IndexAssignStatement:
		c.checkIndexAssignment(s)

	case *ast.ExpressionStatement:
		if s.Expression != nil {
			c.inferExpression(s.Expression)
		}

	case *ast.ReturnStatement:
		if s.ReturnValue != nil {
			c.inferExpression(s.ReturnValue)
		}

	case *ast.WhileStatement:
		c.checkWhileStatement(s)
	}
}

func (c *Checker) checkIndexAssignment(stmt *ast.IndexAssignStatement) {
	if stmt == nil || stmt.Target == nil {
		return
	}

	containerType := c.inferExpression(stmt.Target.Left)
	indexType := c.inferExpression(stmt.Target.Index)
	valueType := c.inferExpression(stmt.Value)

	switch containerType.Kind {
	case Array:
		if indexType.Kind != Unknown && !IsInteger(indexType) {
			c.addError(fmt.Errorf("array index must be int, got %s", indexType.Kind))
		}
		if containerType.Elem != nil && containerType.Elem.Kind != Unknown && valueType.Kind != Unknown && !Same(containerType.Elem, valueType) {
			c.addError(fmt.Errorf("array element expects %s, got %s", containerType.Elem.Kind, valueType.Kind))
		}

	case ByteArray, TypedArray, Slice:
		if indexType.Kind != Unknown && !IsInteger(indexType) {
			c.addError(fmt.Errorf("collection index must be int, got %s", indexType.Kind))
		}
		if containerType.Elem != nil && containerType.Elem.Kind != Unknown && valueType.Kind != Unknown && !Same(containerType.Elem, valueType) {
			if !(IsInteger(containerType.Elem) && IsInteger(valueType)) {
				c.addError(fmt.Errorf("collection element expects %s, got %s", containerType.Elem.Kind, valueType.Kind))
			}
		}

	case Dict:
		if containerType.Key != nil && containerType.Key.Kind != Unknown && indexType.Kind != Unknown && !Same(containerType.Key, indexType) {
			c.addError(fmt.Errorf("dict key expects %s, got %s", containerType.Key.Kind, indexType.Kind))
		}
		if containerType.Value != nil && containerType.Value.Kind != Unknown && valueType.Kind != Unknown && !Same(containerType.Value, valueType) {
			c.addError(fmt.Errorf("dict value expects %s, got %s", containerType.Value.Kind, valueType.Kind))
		}

	case Unknown:
		return

	default:
		c.addError(fmt.Errorf("index assignment not supported for %s", containerType.Kind))
	}
}

func (c *Checker) checkWhileStatement(stmt *ast.WhileStatement) {
	if stmt == nil {
		return
	}

	if stmt.Condition != nil {
		condType := c.inferExpression(stmt.Condition)
		if condType.Kind != Unknown && condType.Kind != Bool {
			c.addError(fmt.Errorf("while condition must be bool, got %s", condType.Kind))
		}
	}

	if stmt.Body != nil {
		c.pushScope()
		for _, inner := range stmt.Body.Statements {
			c.checkStatement(inner)
		}
		c.popScope()
	}
}

func (c *Checker) inferSingleStatementReturnType(stmt ast.Statement) *Type {
	switch s := stmt.(type) {
	case *ast.ReturnStatement:
		if s.ReturnValue == nil {
			return Simple(Null)
		}
		return c.inferExpression(s.ReturnValue)

	case *ast.ExpressionStatement:
		if s.Expression == nil {
			return nil
		}

		if ifExpr, ok := s.Expression.(*ast.IfExpression); ok {
			return c.inferIfReturnType(ifExpr)
		}

		c.inferExpression(s.Expression)
		return nil

	case *ast.VarStatement, *ast.AssignStatement, *ast.WhileStatement:
		c.checkStatement(stmt)
		return nil
	}

	c.checkStatement(stmt)
	return nil
}

func (c *Checker) inferBlockReturnType(block *ast.BlockStatement) *Type {
	if block == nil {
		return Simple(Unknown)
	}

	var inferred *Type
	for _, stmt := range block.Statements {
		current := c.inferSingleStatementReturnType(stmt)
		if current == nil {
			continue
		}

		if inferred == nil {
			inferred = current
			continue
		}

		if inferred.Kind == Unknown || current.Kind == Unknown {
			inferred = Simple(Unknown)
			continue
		}

		if !Same(inferred, current) {
			c.addError(fmt.Errorf("function has conflicting return types: %s and %s", inferred.Kind, current.Kind))
			inferred = Simple(Unknown)
		}
	}

	if inferred == nil {
		return Simple(Null)
	}

	return inferred
}

func (c *Checker) inferHandlerBlockType(block *ast.BlockStatement) *Type {
	if block == nil {
		return Simple(Null)
	}
	return c.inferBlockReturnType(block)
}

func (c *Checker) checkBuiltinCall(name string, args []ast.Expression) *Type {
	switch name {
	case "show":
		if len(args) != 1 {
			c.addError(fmt.Errorf("show expects 1 argument, got %d", len(args)))
			return Simple(Null)
		}
		c.inferExpression(args[0])
		return Simple(Null)

	case "input":
		if len(args) > 1 {
			c.addError(fmt.Errorf("input expects 0 or 1 arguments, got %d", len(args)))
			return Simple(String)
		}
		if len(args) == 1 {
			c.inferExpression(args[0])
		}
		return Simple(String)

	case "sizeOf":
		if len(args) != 1 {
			c.addError(fmt.Errorf("sizeOf expects 1 argument, got %d", len(args)))
			return Simple(Int)
		}
		argType := c.inferExpression(args[0])
		if argType.Kind != Unknown && argType.Kind != Array && argType.Kind != ByteArray && argType.Kind != TypedArray && argType.Kind != Slice && argType.Kind != String && argType.Kind != Dict {
			c.addError(fmt.Errorf("sizeOf expects array, string or dict, got %s", argType.Kind))
		}
		return Simple(Int)

	case "toInt":
		if len(args) != 1 {
			c.addError(fmt.Errorf("toInt expects 1 argument, got %d", len(args)))
			return Simple(Int)
		}
		c.inferExpression(args[0])
		return Simple(Int)

	case "toFloat":
		if len(args) != 1 {
			c.addError(fmt.Errorf("toFloat expects 1 argument, got %d", len(args)))
			return Simple(Float)
		}
		c.inferExpression(args[0])
		return Simple(Float)

	case "toString":
		if len(args) != 1 {
			c.addError(fmt.Errorf("toString expects 1 argument, got %d", len(args)))
			return Simple(String)
		}
		c.inferExpression(args[0])
		return Simple(String)

	case "toBool":
		if len(args) != 1 {
			c.addError(fmt.Errorf("toBool expects 1 argument, got %d", len(args)))
			return Simple(Bool)
		}
		c.inferExpression(args[0])
		return Simple(Bool)

	case "u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64":
		if len(args) != 1 {
			c.addError(fmt.Errorf("%s expects 1 argument, got %d", name, len(args)))
			if kind, ok := FixedIntegerKind(name); ok {
				return Simple(kind)
			}
			return Simple(Unknown)
		}
		argType := c.inferExpression(args[0])
		if argType.Kind != Unknown && !IsInteger(argType) {
			c.addError(fmt.Errorf("%s expects an integer, got %s", name, argType.Kind))
		}
		kind, _ := FixedIntegerKind(name)
		return Simple(kind)

	case "wrapAdd", "wrapSub", "wrapMul",
		"checkedAdd", "checkedSub", "checkedMul",
		"satAdd", "satSub", "satMul":
		if len(args) != 2 {
			c.addError(fmt.Errorf("%s expects 2 arguments, got %d", name, len(args)))
			return Simple(Unknown)
		}
		left := c.inferExpression(args[0])
		right := c.inferExpression(args[1])
		result, err := fixedIntegerResultType(left, right)
		if err != nil {
			c.addError(fmt.Errorf("%s: %w", name, err))
			return Simple(Unknown)
		}
		return result

	case "bytes":
		if len(args) != 1 {
			c.addError(fmt.Errorf("bytes expects 1 argument, got %d", len(args)))
			return ByteArrayOf()
		}
		sizeType := c.inferExpression(args[0])
		if sizeType.Kind != Unknown && !IsInteger(sizeType) {
			c.addError(fmt.Errorf("bytes size must be integer, got %s", sizeType.Kind))
		}
		return ByteArrayOf()

	case "arrayOf":
		if len(args) != 2 {
			c.addError(fmt.Errorf("arrayOf expects 2 arguments, got %d", len(args)))
			return TypedArrayOf(Simple(Unknown))
		}
		typeArg := c.inferExpression(args[0])
		if typeArg.Kind != Unknown && typeArg.Kind != String {
			c.addError(fmt.Errorf("arrayOf type must be string, got %s", typeArg.Kind))
		}
		sizeType := c.inferExpression(args[1])
		if sizeType.Kind != Unknown && !IsInteger(sizeType) {
			c.addError(fmt.Errorf("arrayOf size must be integer, got %s", sizeType.Kind))
		}
		elem := Simple(Unknown)
		if literal, ok := args[0].(*ast.StringLiteral); ok {
			if kind, ok := FixedIntegerKind(literal.Value); ok {
				elem = Simple(kind)
			} else {
				c.addError(fmt.Errorf("arrayOf unsupported element type %q", literal.Value))
			}
		}
		return TypedArrayOf(elem)

	case "slice":
		if len(args) != 3 {
			c.addError(fmt.Errorf("slice expects 3 arguments, got %d", len(args)))
			return SliceOf(Simple(Unknown))
		}
		container := c.inferExpression(args[0])
		for _, arg := range args[1:] {
			indexType := c.inferExpression(arg)
			if indexType.Kind != Unknown && !IsInteger(indexType) {
				c.addError(fmt.Errorf("slice bounds must be integer, got %s", indexType.Kind))
			}
		}
		switch container.Kind {
		case Array, ByteArray, TypedArray, Slice:
			if container.Elem != nil {
				return SliceOf(container.Elem)
			}
			return SliceOf(Simple(Unknown))
		case Unknown:
			return SliceOf(Simple(Unknown))
		default:
			c.addError(fmt.Errorf("slice expects array-like value, got %s", container.Kind))
			return SliceOf(Simple(Unknown))
		}

	case "fill":
		if len(args) != 2 {
			c.addError(fmt.Errorf("fill expects 2 arguments, got %d", len(args)))
			return Simple(Unknown)
		}
		container := c.inferExpression(args[0])
		value := c.inferExpression(args[1])
		switch container.Kind {
		case Array, ByteArray, TypedArray, Slice:
			if container.Elem != nil && container.Elem.Kind != Unknown && value.Kind != Unknown && !Same(container.Elem, value) {
				if !(IsInteger(container.Elem) && IsInteger(value)) {
					c.addError(fmt.Errorf("fill expects %s value, got %s", container.Elem.Kind, value.Kind))
				}
			}
			return container
		case Unknown:
			return Simple(Unknown)
		default:
			c.addError(fmt.Errorf("fill expects array-like value, got %s", container.Kind))
			return Simple(Unknown)
		}

	case "readBytes":
		if len(args) != 1 {
			c.addError(fmt.Errorf("readBytes expects 1 argument, got %d", len(args)))
			return ByteArrayOf()
		}
		pathType := c.inferExpression(args[0])
		if pathType.Kind != Unknown && pathType.Kind != String {
			c.addError(fmt.Errorf("readBytes path must be string, got %s", pathType.Kind))
		}
		return ByteArrayOf()

	case "writeBytes":
		if len(args) != 2 {
			c.addError(fmt.Errorf("writeBytes expects 2 arguments, got %d", len(args)))
			return Simple(Int)
		}
		pathType := c.inferExpression(args[0])
		if pathType.Kind != Unknown && pathType.Kind != String {
			c.addError(fmt.Errorf("writeBytes path must be string, got %s", pathType.Kind))
		}
		bufferType := c.inferExpression(args[1])
		c.requireByteBuffer("writeBytes", bufferType)
		return Simple(Int)

	case "readU16LE", "readU16BE", "readU32LE", "readU32BE", "readU64LE", "readU64BE":
		if len(args) != 2 {
			c.addError(fmt.Errorf("%s expects 2 arguments, got %d", name, len(args)))
			return endianReadType(name)
		}
		c.requireByteBuffer(name, c.inferExpression(args[0]))
		c.requireIntegerArgument(name+" offset", c.inferExpression(args[1]))
		return endianReadType(name)

	case "writeU16LE", "writeU16BE", "writeU32LE", "writeU32BE", "writeU64LE", "writeU64BE":
		if len(args) != 3 {
			c.addError(fmt.Errorf("%s expects 3 arguments, got %d", name, len(args)))
			return Simple(Unknown)
		}
		bufferType := c.inferExpression(args[0])
		c.requireByteBuffer(name, bufferType)
		c.requireIntegerArgument(name+" offset", c.inferExpression(args[1]))
		c.requireIntegerArgument(name+" value", c.inferExpression(args[2]))
		return bufferType

	case "copyBytes":
		if len(args) != 5 {
			c.addError(fmt.Errorf("copyBytes expects 5 arguments, got %d", len(args)))
			return Simple(Unknown)
		}
		destination := c.inferExpression(args[0])
		c.requireByteBuffer("copyBytes destination", destination)
		c.requireIntegerArgument("copyBytes destination offset", c.inferExpression(args[1]))
		c.requireByteBuffer("copyBytes source", c.inferExpression(args[2]))
		c.requireIntegerArgument("copyBytes source offset", c.inferExpression(args[3]))
		c.requireIntegerArgument("copyBytes length", c.inferExpression(args[4]))
		return destination

	case "bytesEqual":
		if len(args) != 2 {
			c.addError(fmt.Errorf("bytesEqual expects 2 arguments, got %d", len(args)))
			return Simple(Bool)
		}
		c.requireByteBuffer("bytesEqual", c.inferExpression(args[0]))
		c.requireByteBuffer("bytesEqual", c.inferExpression(args[1]))
		return Simple(Bool)

	case "sha256":
		if len(args) != 1 {
			c.addError(fmt.Errorf("sha256 expects 1 argument, got %d", len(args)))
			return Simple(String)
		}
		c.requireByteBuffer("sha256", c.inferExpression(args[0]))
		return Simple(String)

	case "first", "last":
		if len(args) != 1 {
			c.addError(fmt.Errorf("%s expects 1 argument, got %d", name, len(args)))
			return Simple(Unknown)
		}
		argType := c.inferExpression(args[0])
		if argType.Kind == Array {
			if argType.Elem != nil {
				return argType.Elem
			}
			return Simple(Unknown)
		}
		if argType.Kind != Unknown {
			c.addError(fmt.Errorf("%s expects array, got %s", name, argType.Kind))
		}
		return Simple(Unknown)
	}

	for _, arg := range args {
		c.inferExpression(arg)
	}
	return Simple(Unknown)
}

func (c *Checker) inferDictLiteral(e *ast.DictLiteral) *Type {
	if len(e.Pairs) == 0 {
		return DictOf(Simple(Unknown), Simple(Unknown))
	}

	var keyType *Type
	var valueType *Type
	keyHomogeneous := true
	valueHomogeneous := true

	for keyExpr, valueExpr := range e.Pairs {
		currentKey := c.inferExpression(keyExpr)
		currentValue := c.inferExpression(valueExpr)

		if keyType == nil {
			keyType = currentKey
		} else if keyType.Kind == Unknown || currentKey.Kind == Unknown {
			keyType = Simple(Unknown)
		} else if !Same(keyType, currentKey) {
			keyHomogeneous = false
		}

		if valueType == nil {
			valueType = currentValue
		} else if valueType.Kind == Unknown || currentValue.Kind == Unknown {
			valueType = Simple(Unknown)
		} else if !Same(valueType, currentValue) {
			valueHomogeneous = false
		}
	}

	if !keyHomogeneous {
		c.addError(fmt.Errorf("dict literal has mixed key types"))
		keyType = Simple(Unknown)
	}

	if !valueHomogeneous {
		c.addError(fmt.Errorf("dict literal has mixed value types"))
		valueType = Simple(Unknown)
	}

	if keyType == nil {
		keyType = Simple(Unknown)
	}
	if valueType == nil {
		valueType = Simple(Unknown)
	}

	return DictOf(keyType, valueType)
}

func (c *Checker) inferIfReturnType(e *ast.IfExpression) *Type {
	if e == nil {
		return nil
	}

	var consequenceType *Type
	var alternativeType *Type

	if e.Consequence != nil {
		c.pushScope()
		consequenceType = c.inferBlockReturnType(e.Consequence)
		c.popScope()
	}

	if e.Alternative != nil {
		c.pushScope()
		alternativeType = c.inferBlockReturnType(e.Alternative)
		c.popScope()
	}

	if consequenceType == nil && alternativeType == nil {
		return nil
	}
	if consequenceType == nil {
		return alternativeType
	}
	if alternativeType == nil {
		return consequenceType
	}

	unified := unifyReturnTypes(consequenceType, alternativeType)
	if unified.Kind == Unknown && consequenceType.Kind != Unknown && alternativeType.Kind != Unknown && !Same(consequenceType, alternativeType) {
		c.addError(fmt.Errorf("function has conflicting return types: %s and %s", consequenceType.Kind, alternativeType.Kind))
	}

	return unified
}

func fixedIntegerResultType(left, right *Type) (*Type, error) {
	if left == nil || right == nil {
		return Simple(Unknown), nil
	}
	if left.Kind == Unknown || right.Kind == Unknown {
		return Simple(Unknown), nil
	}
	if !IsInteger(left) || !IsInteger(right) {
		return nil, fmt.Errorf("fixed integer operations require integer operands, got %s and %s", left.Kind, right.Kind)
	}
	if IsFixedInteger(left) && IsFixedInteger(right) && left.Kind != right.Kind {
		return nil, fmt.Errorf("fixed integer types must match: %s and %s", left.Kind, right.Kind)
	}
	if IsFixedInteger(left) {
		return left, nil
	}
	if IsFixedInteger(right) {
		return right, nil
	}
	return Simple(Int), nil
}

func (c *Checker) inferExpression(exp ast.Expression) *Type {
	switch e := exp.(type) {
	case nil:
		return Simple(Unknown)

	case *ast.IntegerLiteral:
		if e.FixedType != "" {
			if kind, ok := FixedIntegerKind(e.FixedType); ok {
				return Simple(kind)
			}
		}
		return Simple(Int)

	case *ast.FloatLiteral:
		return Simple(Float)

	case *ast.StringLiteral:
		return Simple(String)

	case *ast.Boolean:
		return Simple(Bool)

	case *ast.Identifier:
		if t, ok := c.scope.resolve(e.Value); ok {
			return t
		}
		if t, ok := builtinType(e.Value); ok {
			return t
		}
		return Simple(Unknown)

	case *ast.ArrayLiteral:
		if len(e.Elements) == 0 {
			return ArrayOf(Simple(Unknown))
		}

		first := c.inferExpression(e.Elements[0])
		homogeneous := true

		for i := 1; i < len(e.Elements); i++ {
			next := c.inferExpression(e.Elements[i])

			if first.Kind == Unknown || next.Kind == Unknown {
				first = Simple(Unknown)
				continue
			}

			if !Same(first, next) {
				homogeneous = false
			}
		}

		if !homogeneous {
			c.addError(fmt.Errorf("array literal has mixed element types"))
			return ArrayOf(Simple(Unknown))
		}

		return ArrayOf(first)

	case *ast.DictLiteral:
		return c.inferDictLiteral(e)

	case *ast.PrefixExpression:
		if e.Right == nil {
			return Simple(Unknown)
		}
		right := c.inferExpression(e.Right)
		if e.Operator == "bnot" || e.Operator == "-" {
			if right.Kind == Unknown {
				c.constrainIdentifier(e.Right, Simple(Int))
				return Simple(Int)
			}
			if !IsInteger(right) {
				c.addError(fmt.Errorf("%s expects int, got %s", e.Operator, right.Kind))
				return Simple(Unknown)
			}
		}
		return right

	case *ast.InfixExpression:
		left := c.inferExpression(e.Left)
		right := c.inferExpression(e.Right)

		switch e.Operator {
		case "+", "-", "*", "/", "%", "**":
			if IsFixedInteger(left) || IsFixedInteger(right) {
				result, err := fixedIntegerResultType(left, right)
				if err != nil {
					c.addError(fmt.Errorf("invalid operands for %s: %w", e.Operator, err))
					return Simple(Unknown)
				}
				return result
			}

			if e.Operator == "+" {
				if left.Kind == String && right.Kind == Unknown {
					c.constrainIdentifier(e.Right, Simple(String))
					right = c.inferExpression(e.Right)
				}
				if right.Kind == String && left.Kind == Unknown {
					c.constrainIdentifier(e.Left, Simple(String))
					left = c.inferExpression(e.Left)
				}
			}

			if IsNumeric(left) && right.Kind == Unknown {
				c.constrainIdentifier(e.Right, left)
				right = c.inferExpression(e.Right)
			}
			if IsNumeric(right) && left.Kind == Unknown {
				c.constrainIdentifier(e.Left, right)
				left = c.inferExpression(e.Left)
			}

			if left.Kind == Unknown || right.Kind == Unknown {
				return Simple(Unknown)
			}

			if IsNumeric(left) && IsNumeric(right) {
				if left.Kind == Float || right.Kind == Float {
					return Simple(Float)
				}
				return Simple(Int)
			}

			if e.Operator == "+" && left.Kind == String && right.Kind == String {
				return Simple(String)
			}

			c.addError(fmt.Errorf("invalid operands for %s: %s and %s", e.Operator, left.Kind, right.Kind))
			return Simple(Unknown)

		case "band", "bor", "bxor", "shl", "shr":
			if left.Kind == Unknown {
				c.constrainIdentifier(e.Left, Simple(Int))
				left = c.inferExpression(e.Left)
			}
			if right.Kind == Unknown {
				c.constrainIdentifier(e.Right, Simple(Int))
				right = c.inferExpression(e.Right)
			}

			if left.Kind == Unknown || right.Kind == Unknown {
				return Simple(Int)
			}
			if !IsInteger(left) || !IsInteger(right) {
				c.addError(fmt.Errorf("%s expects int operands, got %s and %s", e.Operator, left.Kind, right.Kind))
				return Simple(Unknown)
			}
			if e.Operator == "shl" || e.Operator == "shr" {
				return left
			}
			if IsFixedInteger(left) || IsFixedInteger(right) {
				result, err := fixedIntegerResultType(left, right)
				if err != nil {
					c.addError(fmt.Errorf("%s: %w", e.Operator, err))
					return Simple(Unknown)
				}
				return result
			}
			return Simple(Int)

		case "==", "!=", "<", ">", "<=", ">=":
			if IsFixedInteger(left) || IsFixedInteger(right) {
				if _, err := fixedIntegerResultType(left, right); err != nil {
					c.addError(fmt.Errorf("cannot compare: %w", err))
				}
				return Simple(Bool)
			}

			if left.Kind == Unknown && right.Kind != Unknown {
				c.constrainIdentifier(e.Left, right)
				left = c.inferExpression(e.Left)
			}
			if right.Kind == Unknown && left.Kind != Unknown {
				c.constrainIdentifier(e.Right, left)
				right = c.inferExpression(e.Right)
			}

			if left.Kind == Unknown || right.Kind == Unknown {
				return Simple(Bool)
			}

			if left.Kind == right.Kind {
				return Simple(Bool)
			}

			if IsNumeric(left) && IsNumeric(right) {
				return Simple(Bool)
			}

			c.addError(fmt.Errorf("cannot compare %s with %s", left.Kind, right.Kind))
			return Simple(Bool)
		}

		return Simple(Unknown)

	case *ast.FunctionLiteral:
		c.pushScope()

		params := make([]*Type, 0, len(e.Parameters))
		for _, p := range e.Parameters {
			if p == nil {
				continue
			}
			pt := Simple(Unknown)
			c.scope.define(p.Value, pt)
			params = append(params, pt)
		}

		ret := Simple(Null)
		if e.Body != nil {
			ret = c.inferBlockReturnType(e.Body)
		}

		refinedParams := make([]*Type, 0, len(e.Parameters))
		for _, p := range e.Parameters {
			if p == nil {
				continue
			}
			if pt, ok := c.scope.resolve(p.Value); ok {
				refinedParams = append(refinedParams, pt)
			} else {
				refinedParams = append(refinedParams, Simple(Unknown))
			}
		}

		c.popScope()
		return FuncOf(refinedParams, ret)

	case *ast.CallExpression:
		if ident, ok := e.Function.(*ast.Identifier); ok {
			if _, builtin := builtinType(ident.Value); builtin {
				return c.checkBuiltinCall(ident.Value, e.Arguments)
			}
		}

		fnType := Simple(Unknown)
		if e.Function != nil {
			fnType = c.inferExpression(e.Function)
		}

		argTypes := make([]*Type, 0, len(e.Arguments))
		for _, arg := range e.Arguments {
			argTypes = append(argTypes, c.inferExpression(arg))
		}

		if fnType.Kind == Func {
			if len(fnType.Params) != len(e.Arguments) {
				c.addError(fmt.Errorf("function expects %d arguments, got %d", len(fnType.Params), len(e.Arguments)))
			} else {
				for i := range fnType.Params {
					paramType := fnType.Params[i]
					argType := argTypes[i]

					if paramType == nil || argType == nil {
						continue
					}
					if paramType.Kind == Unknown || argType.Kind == Unknown {
						continue
					}
					if !Same(paramType, argType) {
						c.addError(fmt.Errorf("argument %d expects %s, got %s", i+1, paramType.Kind, argType.Kind))
					}
				}
			}

			if fnType.Return != nil {
				return fnType.Return
			}
			return Simple(Unknown)
		}

		return Simple(Unknown)

	case *ast.IndexExpression:
		left := c.inferExpression(e.Left)
		index := c.inferExpression(e.Index)

		if left.Kind == Array || left.Kind == ByteArray || left.Kind == TypedArray || left.Kind == Slice {
			if index.Kind != Unknown && !IsInteger(index) {
				c.addError(fmt.Errorf("array index must be int, got %s", index.Kind))
			}
			if left.Elem != nil {
				return left.Elem
			}
			return Simple(Unknown)
		}

		if left.Kind == Dict {
			if left.Key != nil && index.Kind != Unknown && left.Key.Kind != Unknown && !Same(left.Key, index) {
				c.addError(fmt.Errorf("dict key expects %s, got %s", left.Key.Kind, index.Kind))
			}
			if left.Value != nil {
				return left.Value
			}
			return Simple(Unknown)
		}

		if index.Kind != Unknown && !IsInteger(index) {
			c.addError(fmt.Errorf("array index must be int, got %s", index.Kind))
		}
		return Simple(Unknown)

	case *ast.IfExpression:
		if e.Condition != nil {
			condType := c.inferExpression(e.Condition)
			if condType.Kind != Unknown && condType.Kind != Bool {
				c.addError(fmt.Errorf("if condition must be bool, got %s", condType.Kind))
			}
		}

		if ret := c.inferIfReturnType(e); ret != nil {
			return ret
		}

		c.pushScope()
		if e.Consequence != nil {
			for _, stmt := range e.Consequence.Statements {
				c.checkStatement(stmt)
			}
		}
		c.popScope()

		if e.Alternative != nil {
			c.pushScope()
			for _, stmt := range e.Alternative.Statements {
				c.checkStatement(stmt)
			}
			c.popScope()
		}

		return Simple(Unknown)

	case *ast.AwaitExpression:
		if e.Value != nil {
			return c.inferExpression(e.Value)
		}
		return Simple(Unknown)

	case *ast.TryExpression:
		if e.Value != nil {
			return c.inferExpression(e.Value)
		}
		return Simple(Unknown)

	case *ast.ErrorHandlerExpression:
		leftType := Simple(Unknown)
		if e.Left != nil {
			leftType = c.inferExpression(e.Left)
		}

		handlerType := Simple(Null)
		c.pushScope()
		if e.ErrorIdent != nil {
			c.scope.define(e.ErrorIdent.Value, Simple(Unknown))
		}
		if e.Handler != nil {
			handlerType = c.inferHandlerBlockType(e.Handler)
		}
		c.popScope()

		return c.unifyTypesOrError("or handler", leftType, handlerType)

	case *ast.AttributeAccess:
		return Simple(Unknown)
	}

	return Simple(Unknown)
}

func (c *Checker) requireIntegerArgument(label string, value *Type) {
	if value != nil && value.Kind != Unknown && !IsInteger(value) {
		c.addError(fmt.Errorf("%s must be integer, got %s", label, value.Kind))
	}
}

func (c *Checker) requireByteBuffer(label string, value *Type) {
	if value == nil || value.Kind == Unknown {
		return
	}
	switch value.Kind {
	case ByteArray:
		return
	case TypedArray, Slice:
		if value.Elem != nil && (value.Elem.Kind == U8 || value.Elem.Kind == I8 || value.Elem.Kind == Unknown) {
			return
		}
	}
	c.addError(fmt.Errorf("%s expects byte-compatible buffer, got %s", label, value.Kind))
}

func endianReadType(name string) *Type {
	switch name {
	case "readU16LE", "readU16BE":
		return Simple(U16)
	case "readU32LE", "readU32BE":
		return Simple(U32)
	case "readU64LE", "readU64BE":
		return Simple(U64)
	default:
		return Simple(Unknown)
	}
}
