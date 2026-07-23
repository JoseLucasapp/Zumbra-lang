package types

import (
	"fmt"
	"zumbra/ast"
)

type scope struct {
	parent    *scope
	values    map[string]*Type
	mutable   map[string]bool
	functions map[string]functionBinding
	origins   map[string]ast.Node
}

type functionBinding struct {
	literal          *ast.FunctionLiteral
	declarationScope *scope
}

type methodBinding struct {
	literal          *ast.FunctionLiteral
	declarationScope *scope
	owner            *Type
	name             string
}

func newScope(parent *scope) *scope {
	return &scope{
		parent:    parent,
		values:    make(map[string]*Type),
		mutable:   make(map[string]bool),
		functions: make(map[string]functionBinding),
		origins:   make(map[string]ast.Node),
	}
}

func (s *scope) define(name string, t *Type) {
	s.values[name] = t
	s.mutable[name] = true
}

func (s *scope) defineWithOrigin(name string, t *Type, origin ast.Node) {
	s.define(name, t)
	if origin != nil {
		s.origins[name] = origin
	}
}

func (s *scope) defineImmutable(name string, t *Type) {
	s.values[name] = t
	s.mutable[name] = false
}

func (s *scope) defineImmutableWithOrigin(name string, t *Type, origin ast.Node) {
	s.defineImmutable(name, t)
	if origin != nil {
		s.origins[name] = origin
	}
}

func (s *scope) canAssign(name string) bool {
	for cur := s; cur != nil; cur = cur.parent {
		if _, ok := cur.values[name]; ok {
			return cur.mutable[name]
		}
	}
	return false
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

func (s *scope) resolveOrigin(name string) (ast.Node, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if origin, ok := cur.origins[name]; ok {
			return origin, true
		}
	}
	return nil, false
}

func (s *scope) bindFunction(name string, literal *ast.FunctionLiteral) {
	if s == nil || name == "" || literal == nil {
		return
	}
	s.functions[name] = functionBinding{literal: literal, declarationScope: s}
}

func (s *scope) assignFunction(name string, literal *ast.FunctionLiteral) {
	if s == nil || name == "" || literal == nil {
		return
	}
	for cur := s; cur != nil; cur = cur.parent {
		if _, exists := cur.values[name]; exists {
			cur.functions[name] = functionBinding{literal: literal, declarationScope: cur}
			return
		}
	}
	s.bindFunction(name, literal)
}

func (s *scope) unbindFunction(name string) {
	for cur := s; cur != nil; cur = cur.parent {
		if _, exists := cur.values[name]; exists {
			delete(cur.functions, name)
			return
		}
	}
}

func (s *scope) resolveFunction(name string) (functionBinding, bool) {
	for cur := s; cur != nil; cur = cur.parent {
		if value, ok := cur.functions[name]; ok {
			return value, true
		}
	}
	return functionBinding{}, false
}

type Checker struct {
	global      *scope
	scope       *scope
	errors      []error
	aliases     map[string]*Type
	structs     map[string]*Type
	enums       map[string]*Type
	externals   map[string]*Type
	unsafeDepth int
	nodeTypes   map[ast.Node]*Type
	contextual  map[*ast.FunctionLiteral]*Type
	methods     map[string]methodBinding
}

func NewChecker() *Checker {
	global := newScope(nil)
	c := &Checker{
		global:     global,
		scope:      global,
		errors:     []error{},
		aliases:    map[string]*Type{},
		structs:    map[string]*Type{},
		enums:      map[string]*Type{},
		externals:  map[string]*Type{},
		nodeTypes:  map[ast.Node]*Type{},
		contextual: map[*ast.FunctionLiteral]*Type{},
		methods:    map[string]methodBinding{},
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
	c.aliases = map[string]*Type{}
	c.structs = map[string]*Type{}
	c.enums = map[string]*Type{}
	c.externals = map[string]*Type{}
	c.unsafeDepth = 0
	c.nodeTypes = map[ast.Node]*Type{}
	c.contextual = map[*ast.FunctionLiteral]*Type{}
	c.methods = map[string]methodBinding{}
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

// refineType fills unknown portions of current with information from wanted.
// It deliberately remains monomorphic: once a concrete call specializes a
// function or collection, a later incompatible use is rejected instead of
// silently creating a second hidden specialization.
func refineType(current *Type, wanted *Type) (*Type, bool) {
	if current == nil {
		return Clone(wanted), true
	}
	if wanted == nil {
		return Clone(current), true
	}
	if current.Kind == Unknown {
		return Clone(wanted), true
	}
	if wanted.Kind == Unknown {
		return Clone(current), true
	}
	if current.Kind != wanted.Kind {
		return Clone(current), false
	}

	result := Clone(current)
	switch current.Kind {
	case Array, ByteArray, TypedArray, Slice, Task, Channel:
		refined, ok := refineType(current.Elem, wanted.Elem)
		if !ok {
			return result, false
		}
		result.Elem = refined
		return result, true

	case Dict:
		key, keyOK := refineType(current.Key, wanted.Key)
		value, valueOK := refineType(current.Value, wanted.Value)
		if !keyOK || !valueOK {
			return result, false
		}
		result.Key = key
		result.Value = value
		return result, true

	case Func:
		if current.Async != wanted.Async || len(current.Params) != len(wanted.Params) {
			return result, false
		}
		result.Params = make([]*Type, len(current.Params))
		for index := range current.Params {
			parameter, ok := refineType(current.Params[index], wanted.Params[index])
			if !ok {
				return Clone(current), false
			}
			result.Params[index] = parameter
		}
		returned, ok := refineType(current.Return, wanted.Return)
		if !ok {
			return Clone(current), false
		}
		result.Return = returned
		return result, true

	case Struct, Enum:
		return result, current.Name == wanted.Name
	}

	return result, true
}

func methodBindingKey(owner, name string) string {
	return owner + "." + name
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

	refined, compatible := refineType(current, wanted)
	if !compatible {
		_ = c.scope.assign(id.Value, Simple(Unknown))
		c.nodeTypes[id] = Simple(Unknown)
		return
	}
	_ = c.scope.assign(id.Value, refined)
	c.nodeTypes[id] = Clone(refined)
	if origin, exists := c.scope.resolveOrigin(id.Value); exists {
		c.nodeTypes[origin] = Clone(refined)
	}
}

func (c *Checker) constrainExpression(expr ast.Expression, wanted *Type, context string) *Type {
	if expr == nil || wanted == nil {
		return c.inferExpression(expr)
	}

	if identifier, ok := expr.(*ast.Identifier); ok {
		current, exists := c.scope.resolve(identifier.Value)
		if !exists || current == nil {
			return c.inferExpression(expr)
		}
		refined, compatible := refineType(current, wanted)
		if !compatible {
			c.addError(fmt.Errorf("%s inferred %s as %s and cannot also use it as %s", context, identifier.Value, current.String(), wanted.String()))
			c.nodeTypes[identifier] = Clone(current)
			return current
		}
		_ = c.scope.assign(identifier.Value, refined)
		c.nodeTypes[identifier] = Clone(refined)
		if origin, originExists := c.scope.resolveOrigin(identifier.Value); originExists {
			c.nodeTypes[origin] = Clone(refined)
		}
		return refined
	}

	actual := c.inferExpression(expr)
	refined, compatible := refineType(actual, wanted)
	if !compatible {
		c.addError(fmt.Errorf("%s expects %s, got %s", context, wanted.String(), actual.String()))
		return actual
	}
	c.nodeTypes[expr] = Clone(refined)
	return refined
}

func (c *Checker) checkStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ConstStatement:
		t := Simple(Unknown)
		if s.Value != nil {
			t = c.inferExpression(s.Value)
		}
		if s.Name != nil {
			c.scope.defineImmutableWithOrigin(s.Name.Value, t, s.Value)
			c.nodeTypes[s] = Clone(t)
			if function, ok := s.Value.(*ast.FunctionLiteral); ok {
				c.scope.assignFunction(s.Name.Value, function)
			}
		}

	case *ast.TypeAliasStatement:
		if s.Name != nil && s.Target != nil {
			t := c.typeFromName(s.Target.Value)
			if t.Kind == Unknown {
				c.addError(fmt.Errorf("unknown type %s", s.Target.Value))
			}
			c.aliases[s.Name.Value] = t
		}

	case *ast.StructStatement:
		c.checkStructStatement(s)

	case *ast.EnumStatement:
		c.checkEnumStatement(s)

	case *ast.ExternBlockStatement:
		c.checkExternBlock(s)

	case *ast.UnsafeStatement:
		c.unsafeDepth++
		if s.Body != nil {
			c.pushScope()
			for _, inner := range s.Body.Statements {
				c.checkStatement(inner)
			}
			c.popScope()
		}
		c.unsafeDepth--

	case *ast.VarStatement:
		var t *Type = Simple(Unknown)
		if s.Value != nil {
			t = c.inferExpression(s.Value)
		}
		if s.Name != nil {
			c.scope.defineWithOrigin(s.Name.Value, t, s.Value)
			c.nodeTypes[s] = Clone(t)
			if function, ok := s.Value.(*ast.FunctionLiteral); ok {
				c.scope.bindFunction(s.Name.Value, function)
			}
		}

	case *ast.AssignStatement:
		if s.Value != nil && s.Name != nil {
			valueType := c.inferExpression(s.Value)
			if current, exists := c.scope.resolve(s.Name.Value); exists {
				if !c.scope.canAssign(s.Name.Value) {
					c.addError(fmt.Errorf("cannot assign to constant %s", s.Name.Value))
					break
				}
				if current.Kind != Unknown && valueType.Kind != Unknown && !Same(current, valueType) {
					c.addError(fmt.Errorf("assignment expects %s, got %s", current.Kind, valueType.Kind))
				} else {
					_ = c.scope.assign(s.Name.Value, valueType)
					if origin, originExists := c.scope.resolveOrigin(s.Name.Value); originExists {
						c.nodeTypes[origin] = Clone(valueType)
					}
				}
			} else {
				c.scope.defineWithOrigin(s.Name.Value, valueType, s.Value)
			}
			c.nodeTypes[s] = Clone(valueType)
			if function, ok := s.Value.(*ast.FunctionLiteral); ok {
				c.scope.assignFunction(s.Name.Value, function)
			} else {
				c.scope.unbindFunction(s.Name.Value)
			}
		}

	case *ast.AttributeAssignStatement:
		c.checkAttributeAssignment(s)

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

	// First validate every statement in the block. Return inference is kept as a
	// separate pass so unreachable expressions do not accidentally become an
	// implicit return when the function already has explicit returns.
	for _, stmt := range block.Statements {
		c.checkStatementWithoutReturnUnification(stmt)
	}

	explicit := c.collectExplicitReturnTypes(block)
	if len(explicit) > 0 {
		result := explicit[0]
		for _, current := range explicit[1:] {
			if result.Kind == Unknown || current.Kind == Unknown {
				result = Simple(Unknown)
				continue
			}
			if !Same(result, current) {
				c.addError(fmt.Errorf("function has conflicting return types: %s and %s", result.Kind, current.Kind))
				result = Simple(Unknown)
			}
		}
		return result
	}

	// Without explicit return statements, a block evaluates to its final
	// expression. This is used by functions, match cases and expression blocks.
	if len(block.Statements) > 0 {
		if expression, ok := block.Statements[len(block.Statements)-1].(*ast.ExpressionStatement); ok && expression.Expression != nil {
			return c.inferExpression(expression.Expression)
		}
	}
	return Simple(Null)
}

func (c *Checker) checkStatementWithoutReturnUnification(stmt ast.Statement) {
	expressionStmt, ok := stmt.(*ast.ExpressionStatement)
	if !ok || expressionStmt.Expression == nil {
		c.checkStatement(stmt)
		return
	}

	switch expression := expressionStmt.Expression.(type) {
	case *ast.IfExpression:
		if expression.Condition != nil {
			condition := c.inferExpression(expression.Condition)
			if condition.Kind != Unknown && condition.Kind != Bool {
				c.addError(fmt.Errorf("if condition must be bool, got %s", condition.Kind))
			}
		}
		for _, branch := range []*ast.BlockStatement{expression.Consequence, expression.Alternative} {
			if branch == nil {
				continue
			}
			c.pushScope()
			for _, inner := range branch.Statements {
				c.checkStatementWithoutReturnUnification(inner)
			}
			c.popScope()
		}
	case *ast.MatchExpression:
		valueType := c.inferExpression(expression.Value)
		for _, candidate := range expression.Cases {
			pattern := c.inferExpression(candidate.Pattern)
			if valueType.Kind != Unknown && pattern.Kind != Unknown && !Same(valueType, pattern) {
				c.addError(fmt.Errorf("match compares %s with %s", valueType.Kind, pattern.Kind))
			}
			c.pushScope()
			for _, inner := range candidate.Body.Statements {
				c.checkStatementWithoutReturnUnification(inner)
			}
			c.popScope()
		}
		if expression.Default != nil {
			c.pushScope()
			for _, inner := range expression.Default.Statements {
				c.checkStatementWithoutReturnUnification(inner)
			}
			c.popScope()
		}
	default:
		c.checkStatement(stmt)
	}
}

func (c *Checker) collectExplicitReturnTypes(block *ast.BlockStatement) []*Type {
	if block == nil {
		return nil
	}
	result := []*Type{}
	for _, stmt := range block.Statements {
		switch value := stmt.(type) {
		case *ast.ReturnStatement:
			if value.ReturnValue == nil {
				result = append(result, Simple(Null))
			} else {
				result = append(result, c.inferExpression(value.ReturnValue))
			}
		case *ast.WhileStatement:
			result = append(result, c.collectExplicitReturnTypes(value.Body)...)
		case *ast.ExpressionStatement:
			result = append(result, c.collectExpressionReturnTypes(value.Expression)...)
		}
	}
	return result
}

func (c *Checker) collectExpressionReturnTypes(expression ast.Expression) []*Type {
	switch value := expression.(type) {
	case *ast.IfExpression:
		result := c.collectExplicitReturnTypes(value.Consequence)
		result = append(result, c.collectExplicitReturnTypes(value.Alternative)...)
		return result
	case *ast.MatchExpression:
		result := []*Type{}
		for _, candidate := range value.Cases {
			result = append(result, c.collectExplicitReturnTypes(candidate.Body)...)
		}
		result = append(result, c.collectExplicitReturnTypes(value.Default)...)
		return result
	case *ast.ForLoop:
		return c.collectExplicitReturnTypes(value.Block)
	case *ast.ForEachArrayLoop:
		return c.collectExplicitReturnTypes(value.Block)
	case *ast.ForEachMapLoop:
		return c.collectExplicitReturnTypes(value.Block)
	case *ast.ForEachDotRange:
		return c.collectExplicitReturnTypes(value.Block)
	case *ast.ForEverLoop:
		return c.collectExplicitReturnTypes(value.Block)
	case *ast.ErrorHandlerExpression:
		return c.collectExplicitReturnTypes(value.Handler)
	default:
		return nil
	}
}

func (c *Checker) inferHandlerBlockType(block *ast.BlockStatement) *Type {
	if block == nil {
		return Simple(Null)
	}
	return c.inferBlockReturnType(block)
}

func (c *Checker) checkBuiltinCall(name string, args []ast.Expression) *Type {
	switch name {
	case "join":
		if len(args) != 1 {
			c.addError(fmt.Errorf("join expects 1 argument, got %d", len(args)))
			return Simple(Unknown)
		}
		t := c.inferExpression(args[0])
		if t.Kind == Task && t.Elem != nil {
			return t.Elem
		}
		if t.Kind != Unknown {
			c.addError(fmt.Errorf("join expects Task, got %s", t.String()))
		}
		return Simple(Unknown)
	case "cancel", "taskDone", "taskCancelled":
		if len(args) != 1 {
			c.addError(fmt.Errorf("%s expects 1 argument, got %d", name, len(args)))
			return Simple(Bool)
		}
		t := c.inferExpression(args[0])
		if t.Kind != Unknown && t.Kind != Task {
			c.addError(fmt.Errorf("%s expects Task, got %s", name, t.String()))
		}
		return Simple(Bool)
	case "joinTimeout":
		if len(args) != 2 {
			c.addError(fmt.Errorf("joinTimeout expects 2 arguments, got %d", len(args)))
			return ArrayOf(Simple(Unknown))
		}
		c.inferExpression(args[0])
		c.inferExpression(args[1])
		return ArrayOf(Simple(Unknown))
	case "sleepMs":
		if len(args) != 1 {
			c.addError(fmt.Errorf("sleepMs expects 1 argument, got %d", len(args)))
		} else {
			d := c.inferExpression(args[0])
			if d.Kind != Unknown && !IsInteger(d) {
				c.addError(fmt.Errorf("sleepMs expects integer milliseconds, got %s", d.String()))
			}
		}
		return Simple(Null)
	case "channel":
		if len(args) > 1 {
			c.addError(fmt.Errorf("channel expects 0 or 1 arguments, got %d", len(args)))
		}
		if len(args) == 1 {
			c.inferExpression(args[0])
		}
		return ChannelOf(Simple(Unknown))
	case "send":
		if len(args) != 2 {
			c.addError(fmt.Errorf("send expects 2 arguments, got %d", len(args)))
			return Simple(Null)
		}
		ch := c.inferExpression(args[0])
		val := c.inferExpression(args[1])
		if ch.Kind == Unknown {
			c.constrainIdentifier(args[0], ChannelOf(Clone(val)))
			ch = c.inferExpression(args[0])
		}
		if ch.Kind != Unknown && ch.Kind != Channel {
			c.addError(fmt.Errorf("send expects Channel, got %s", ch.String()))
		}
		if ch.Kind == Channel && ch.Elem != nil {
			if ch.Elem.Kind == Unknown {
				ch.Elem = Clone(val)
			} else if val.Kind != Unknown && !Compatible(ch.Elem, val) {
				c.addError(fmt.Errorf("send expects %s values, got %s", ch.Elem.String(), val.String()))
			}
			c.constrainExpression(args[0], ch, "send channel")
		}
		return Simple(Null)
	case "receive":
		if len(args) != 1 {
			c.addError(fmt.Errorf("receive expects 1 argument, got %d", len(args)))
			return Simple(Unknown)
		}
		ch := c.inferExpression(args[0])
		if ch.Kind == Unknown {
			c.constrainIdentifier(args[0], ChannelOf(Simple(Unknown)))
			ch = c.inferExpression(args[0])
		}
		if ch.Kind == Channel && ch.Elem != nil {
			return ch.Elem
		}
		if ch.Kind != Unknown {
			c.addError(fmt.Errorf("receive expects Channel, got %s", ch.String()))
		}
		return Simple(Unknown)
	case "receiveOk", "receiveTimeout":
		for _, arg := range args {
			c.inferExpression(arg)
		}
		return ArrayOf(Simple(Unknown))
	case "closeChannel", "channelClosed":
		if len(args) != 1 {
			c.addError(fmt.Errorf("%s expects 1 argument, got %d", name, len(args)))
		} else {
			ch := c.inferExpression(args[0])
			if ch.Kind != Unknown && ch.Kind != Channel {
				c.addError(fmt.Errorf("%s expects Channel, got %s", name, ch.String()))
			}
		}
		return Simple(Bool)
	case "channelLen", "channelCap":
		if len(args) != 1 {
			c.addError(fmt.Errorf("%s expects 1 argument, got %d", name, len(args)))
		} else {
			c.inferExpression(args[0])
		}
		return Simple(Int)
	case "mutex":
		return Simple(Mutex)
	case "rwMutex":
		return Simple(RWMutex)
	case "waitGroup":
		return Simple(WaitGroup)
	case "semaphore":
		if len(args) == 1 {
			c.inferExpression(args[0])
		}
		return Simple(Semaphore)
	case "atomicInt":
		if len(args) == 1 {
			c.inferExpression(args[0])
		}
		return Simple(AtomicInt)
	case "atomicLoad", "atomicAdd", "atomicSwap":
		if len(args) > 0 {
			c.constrainExpression(args[0], Simple(AtomicInt), name+" counter")
		}
		for _, arg := range args[1:] {
			c.inferExpression(arg)
		}
		return Simple(Int)
	case "atomicCompareSwap":
		if len(args) > 0 {
			c.constrainExpression(args[0], Simple(AtomicInt), name+" counter")
		}
		for _, arg := range args[1:] {
			c.inferExpression(arg)
		}
		return Simple(Bool)
	case "atomicStore":
		if len(args) > 0 {
			c.constrainExpression(args[0], Simple(AtomicInt), "atomicStore counter")
		}
		for _, arg := range args[1:] {
			c.inferExpression(arg)
		}
		return Simple(Null)
	case "rLock", "rUnlock":
		if len(args) > 0 {
			c.constrainExpression(args[0], Simple(RWMutex), name+" guard")
		}
		return Simple(Null)
	case "wgAdd", "wgDone", "wgWait":
		if len(args) > 0 {
			c.constrainExpression(args[0], Simple(WaitGroup), name+" group")
		}
		for _, arg := range args[1:] {
			c.inferExpression(arg)
		}
		return Simple(Null)
	case "acquire", "release":
		if len(args) > 0 {
			c.constrainExpression(args[0], Simple(Semaphore), name+" semaphore")
		}
		return Simple(Null)
	case "lock", "unlock":
		for _, arg := range args {
			c.inferExpression(arg)
		}
		return Simple(Null)
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

func containsUnknown(value *Type) bool {
	if value == nil {
		return true
	}
	if value.Kind == Unknown {
		return true
	}
	switch value.Kind {
	case Array, ByteArray, TypedArray, Slice, Task, Channel:
		return value.Elem == nil || containsUnknown(value.Elem)
	case Dict:
		return value.Key == nil || value.Value == nil || containsUnknown(value.Key) || containsUnknown(value.Value)
	case Func:
		if value.Return == nil || containsUnknown(value.Return) {
			return true
		}
		for _, parameter := range value.Params {
			if containsUnknown(parameter) {
				return true
			}
		}
	}
	return false
}

func (c *Checker) inferExpressionExpected(exp ast.Expression, expected *Type) *Type {
	if exp == nil || expected == nil || expected.Kind != Func {
		return c.inferExpression(exp)
	}

	switch value := exp.(type) {
	case *ast.FunctionLiteral:
		result := c.inferFunctionLiteral(value, expected, c.scope)
		c.nodeTypes[value] = Clone(result)
		return result

	case *ast.Identifier:
		current, exists := c.scope.resolve(value.Value)
		if exists && current != nil && current.Kind == Func && !containsUnknown(current) {
			if !Compatible(expected, current) || !Compatible(current, expected) {
				c.addError(fmt.Errorf("callback %s expects %s, got %s", value.Value, expected.String(), current.String()))
			}
			c.nodeTypes[value] = Clone(current)
			return current
		}

		if binding, ok := c.scope.resolveFunction(value.Value); ok {
			result := c.inferFunctionLiteral(binding.literal, expected, binding.declarationScope)
			_ = c.scope.assign(value.Value, result)
			c.nodeTypes[value] = Clone(result)
			c.nodeTypes[binding.literal] = Clone(result)
			return result
		}
	}

	actual := c.inferExpression(exp)
	if actual.Kind != Unknown && (!Compatible(expected, actual) || !Compatible(actual, expected)) {
		c.addError(fmt.Errorf("callback expects %s, got %s", expected.String(), actual.String()))
	}
	return actual
}

func (c *Checker) inferFunctionLiteral(function *ast.FunctionLiteral, expected *Type, parent *scope) *Type {
	if function == nil {
		return Simple(Unknown)
	}

	if contextual, ok := c.contextual[function]; ok {
		if expected != nil && (!Compatible(expected, contextual) || !Compatible(contextual, expected)) {
			c.addError(fmt.Errorf("function %s was inferred as %s and cannot also satisfy %s", function.Name, contextual.String(), expected.String()))
		}
		return contextual
	}

	if parent == nil {
		parent = c.scope
	}
	previous := c.scope
	c.scope = newScope(parent)
	defer func() { c.scope = previous }()

	if expected != nil && expected.Kind == Func && len(expected.Params) != len(function.Parameters) {
		c.addError(fmt.Errorf("callback expects %d parameters, got %d", len(expected.Params), len(function.Parameters)))
	}

	for index, parameter := range function.Parameters {
		if parameter == nil {
			continue
		}
		parameterType := Simple(Unknown)
		if expected != nil && expected.Kind == Func && index < len(expected.Params) && expected.Params[index] != nil {
			parameterType = Clone(expected.Params[index])
		}
		c.scope.define(parameter.Value, parameterType)
	}

	returned := Simple(Null)
	if function.Body != nil {
		returned = c.inferBlockReturnType(function.Body)
	}

	refinedParams := make([]*Type, 0, len(function.Parameters))
	for _, parameter := range function.Parameters {
		if parameter == nil {
			continue
		}
		if parameterType, ok := c.scope.resolve(parameter.Value); ok {
			refinedParams = append(refinedParams, parameterType)
		} else {
			refinedParams = append(refinedParams, Simple(Unknown))
		}
	}

	if expected != nil && expected.Kind == Func && expected.Return != nil {
		if returned == nil || returned.Kind == Unknown {
			returned = Clone(expected.Return)
		} else if !Compatible(expected.Return, returned) || !Compatible(returned, expected.Return) {
			c.addError(fmt.Errorf("callback return expects %s, got %s", expected.Return.String(), returned.String()))
		}
	}

	result := FuncOf(refinedParams, returned)
	result.Async = function.Async
	if expected != nil && expected.Kind == Func {
		if !Compatible(expected, result) || !Compatible(result, expected) {
			c.addError(fmt.Errorf("callback expects %s, got %s", expected.String(), result.String()))
		}
		c.contextual[function] = Clone(result)
	}
	c.nodeTypes[function] = Clone(result)
	return result
}

func (c *Checker) expectedCallType(current *Type, arguments []*Type) *Type {
	params := make([]*Type, len(arguments))
	for index, argument := range arguments {
		if current != nil && current.Kind == Func && index < len(current.Params) {
			refined, compatible := refineType(current.Params[index], argument)
			if compatible {
				params[index] = refined
				continue
			}
			params[index] = Clone(current.Params[index])
			continue
		}
		params[index] = Clone(argument)
	}
	returned := Simple(Unknown)
	async := false
	if current != nil && current.Kind == Func {
		returned = Clone(current.Return)
		async = current.Async
	}
	result := FuncOf(params, returned)
	result.Async = async
	return result
}

func (c *Checker) refineNamedFunctionCall(identifier *ast.Identifier, binding functionBinding, current *Type, arguments []ast.Expression, argumentTypes []*Type) *Type {
	if identifier == nil || binding.literal == nil {
		return current
	}
	if len(binding.literal.Parameters) != len(arguments) {
		return current
	}
	expected := c.expectedCallType(current, argumentTypes)
	refined := c.inferFunctionLiteral(binding.literal, expected, binding.declarationScope)
	if refined == nil || refined.Kind != Func {
		return current
	}
	_ = c.scope.assign(identifier.Value, refined)
	c.nodeTypes[identifier] = Clone(refined)
	c.nodeTypes[binding.literal] = Clone(refined)
	if origin, exists := c.scope.resolveOrigin(identifier.Value); exists {
		c.nodeTypes[origin] = Clone(refined)
	}
	return refined
}

func (c *Checker) inferMethodLiteral(binding methodBinding, current *Type, arguments []ast.Expression, argumentTypes []*Type) *Type {
	if binding.literal == nil || binding.owner == nil {
		return current
	}
	if len(binding.literal.Parameters) != len(arguments)+1 {
		return current
	}
	exposedExpected := c.expectedCallType(current, argumentTypes)
	fullParams := make([]*Type, 0, len(exposedExpected.Params)+1)
	fullParams = append(fullParams, Clone(binding.owner))
	fullParams = append(fullParams, exposedExpected.Params...)
	fullExpected := FuncOf(fullParams, Clone(exposedExpected.Return))
	fullExpected.Async = exposedExpected.Async
	full := c.inferFunctionLiteral(binding.literal, fullExpected, binding.declarationScope)
	if full == nil || full.Kind != Func || len(full.Params) == 0 {
		return current
	}
	exposed := FuncOf(cloneTypes(full.Params[1:]), Clone(full.Return))
	exposed.Async = full.Async
	binding.owner.Methods[binding.name] = exposed
	c.nodeTypes[binding.literal] = Clone(exposed)
	return exposed
}

func cloneTypes(values []*Type) []*Type {
	result := make([]*Type, len(values))
	for index, value := range values {
		result[index] = Clone(value)
	}
	return result
}

func (c *Checker) inferExpression(exp ast.Expression) *Type {
	value := c.inferExpressionImpl(exp)
	if exp != nil {
		c.nodeTypes[exp] = Clone(value)
	}
	return value
}

func (c *Checker) inferExpressionImpl(exp ast.Expression) *Type {
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

			if Same(left, right) {
				return Simple(Bool)
			}
			if left.Kind == right.Kind && left.Kind != Struct && left.Kind != Enum {
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
		return c.inferFunctionLiteral(e, nil, c.scope)

	case *ast.CallExpression:
		if ident, ok := e.Function.(*ast.Identifier); ok {
			if _, builtin := builtinType(ident.Value); builtin {
				return c.checkBuiltinCall(ident.Value, e.Arguments)
			}
			if structType, exists := c.structs[ident.Value]; exists {
				return c.checkStructConstructor(structType, e.Arguments)
			}
		}

		fnType := Simple(Unknown)
		if e.Function != nil {
			fnType = c.inferExpression(e.Function)
		}

		argTypes := make([]*Type, 0, len(e.Arguments))
		for index, arg := range e.Arguments {
			var expected *Type
			if fnType.Kind == Func && index < len(fnType.Params) {
				expected = fnType.Params[index]
			}
			argTypes = append(argTypes, c.inferExpressionExpected(arg, expected))
		}

		if identifier, ok := e.Function.(*ast.Identifier); ok {
			if binding, exists := c.scope.resolveFunction(identifier.Value); exists && containsUnknown(fnType) {
				fnType = c.refineNamedFunctionCall(identifier, binding, fnType, e.Arguments, argTypes)
				c.nodeTypes[e.Function] = Clone(fnType)
			}
		}

		if attribute, ok := e.Function.(*ast.AttributeAccess); ok {
			ownerType := c.inferExpression(attribute.Object)
			if ownerType.Kind == Struct {
				if binding, exists := c.methods[methodBindingKey(ownerType.Name, attribute.Property.Value)]; exists && containsUnknown(fnType) {
					fnType = c.inferMethodLiteral(binding, fnType, e.Arguments, argTypes)
					c.nodeTypes[e.Function] = Clone(fnType)
				}
			}
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
					if Compatible(paramType, argType) {
						argTypes[i] = c.constrainExpression(e.Arguments[i], paramType, fmt.Sprintf("argument %d", i+1))
						argType = argTypes[i]
					}
					if paramType.Kind == Unknown || argType.Kind == Unknown {
						continue
					}
					if !Compatible(paramType, argType) {
						c.addError(fmt.Errorf("argument %d expects %s, got %s", i+1, paramType.String(), argType.String()))
					}
				}
			}

			if fnType.Return != nil {
				if fnType.Async {
					return TaskOf(Clone(fnType.Return))
				}
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

	case *ast.SpawnExpression:
		if e.Value == nil {
			return TaskOf(Simple(Unknown))
		}
		callType := c.inferExpression(e.Value)
		if callType.Kind == Task {
			return callType
		}
		return TaskOf(callType)

	case *ast.AwaitExpression:
		if e.Value != nil {
			valueType := c.inferExpression(e.Value)
			if valueType.Kind == Task {
				if valueType.Elem != nil {
					return valueType.Elem
				}
				return Simple(Unknown)
			}
			return valueType
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

	case *ast.MatchExpression:
		return c.inferMatchExpression(e)

	case *ast.AttributeAccess:
		left := c.inferExpression(e.Object)
		if left.Kind == Struct {
			if field, ok := left.Fields[e.Property.Value]; ok {
				return field
			}
			if method, ok := left.Methods[e.Property.Value]; ok {
				return method
			}
			c.addError(fmt.Errorf("unknown field or method %s on %s", e.Property.Value, left.Name))
		}
		if left.Kind == Enum {
			if left.Members[e.Property.Value] {
				return left
			}
			c.addError(fmt.Errorf("unknown enum member %s.%s", left.Name, e.Property.Value))
		}
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

func (c *Checker) typeFromName(name string) *Type {
	if alias, ok := c.aliases[name]; ok {
		return alias
	}
	if value, ok := c.structs[name]; ok {
		return value
	}
	if value, ok := c.enums[name]; ok {
		return value
	}
	switch Kind(name) {
	case Int, U8, U16, U32, U64, I8, I16, I32, I64, Float, Bool, String, Pointer, Null:
		return Simple(Kind(name))
	default:
		return Simple(Unknown)
	}
}

func (c *Checker) checkExternBlock(stmt *ast.ExternBlockStatement) {
	if stmt == nil {
		return
	}
	if stmt.ABI != "C" {
		c.addError(fmt.Errorf("unsupported extern ABI %q", stmt.ABI))
	}
	for _, fn := range stmt.Functions {
		if fn == nil || fn.Name == nil {
			continue
		}
		params := make([]*Type, 0, len(fn.Parameters))
		for _, param := range fn.Parameters {
			t := c.typeFromExtern(param.Type)
			if t.Kind == Unknown {
				c.addError(fmt.Errorf("extern function %s uses unsupported parameter type %s", fn.Name.Value, param.Type.String()))
			}
			params = append(params, t)
		}
		ret := c.typeFromExtern(fn.ReturnType)
		if ret.Kind == Unknown {
			c.addError(fmt.Errorf("extern function %s uses unsupported return type %s", fn.Name.Value, fn.ReturnType.String()))
		}
		functionType := FuncOf(params, ret)
		c.externals[fn.Name.Value] = functionType
		c.scope.defineImmutable(fn.Name.Value, functionType)
	}
}

func (c *Checker) typeFromExtern(t *ast.ExternType) *Type {
	if t == nil {
		return Simple(Null)
	}
	if t.Name == "callback" {
		params := make([]*Type, 0, len(t.CallbackParams))
		for _, param := range t.CallbackParams {
			params = append(params, c.typeFromExtern(param))
		}
		return FuncOf(params, c.typeFromExtern(t.CallbackReturn))
	}
	switch t.Name {
	case "void":
		return Simple(Null)
	case "cstring", "string":
		return Simple(String)
	case "ptr":
		return Simple(Pointer)
	case "usize":
		return Simple(U64)
	default:
		return c.typeFromName(t.Name)
	}
}

func (c *Checker) checkStructStatement(stmt *ast.StructStatement) {
	if stmt == nil || stmt.Name == nil {
		return
	}
	fields := map[string]*Type{}
	for _, field := range stmt.Fields {
		if _, exists := fields[field.Name.Value]; exists {
			c.addError(fmt.Errorf("duplicate field %s in %s", field.Name.Value, stmt.Name.Value))
			continue
		}
		t := Simple(Unknown)
		if field.TypeName != "" {
			t = c.typeFromName(field.TypeName)
			if t.Kind == Unknown {
				c.addError(fmt.Errorf("unknown field type %s", field.TypeName))
			}
		}
		fields[field.Name.Value] = t
	}
	structType := StructOf(stmt.Name.Value, fields, map[string]*Type{})
	c.structs[stmt.Name.Value] = structType
	params := make([]*Type, 0, len(stmt.Fields))
	for _, field := range stmt.Fields {
		params = append(params, fields[field.Name.Value])
	}
	c.scope.defineImmutable(stmt.Name.Value, FuncOf(params, structType))
	declarationScope := c.scope

	for _, method := range stmt.Methods {
		if method == nil || method.Name == nil || method.Function == nil {
			continue
		}
		if _, exists := structType.Methods[method.Name.Value]; exists {
			c.addError(fmt.Errorf("duplicate method %s in %s", method.Name.Value, stmt.Name.Value))
			continue
		}
		c.pushScope()
		methodParams := []*Type{}
		for index, param := range method.Function.Parameters {
			pt := Simple(Unknown)
			if index == 0 && param.Value == "self" {
				pt = structType
			} else {
				methodParams = append(methodParams, pt)
			}
			c.scope.define(param.Value, pt)
		}
		ret := c.inferBlockReturnType(method.Function.Body)
		refined := []*Type{}
		for index, param := range method.Function.Parameters {
			if index == 0 && param.Value == "self" {
				continue
			}
			if pt, ok := c.scope.resolve(param.Value); ok {
				refined = append(refined, pt)
			} else {
				refined = append(refined, Simple(Unknown))
			}
		}
		c.popScope()
		methodType := FuncOf(refined, ret)
		methodType.Async = method.Function.Async
		structType.Methods[method.Name.Value] = methodType
		c.nodeTypes[method.Function] = Clone(methodType)
		c.methods[methodBindingKey(structType.Name, method.Name.Value)] = methodBinding{
			literal:          method.Function,
			declarationScope: declarationScope,
			owner:            structType,
			name:             method.Name.Value,
		}
	}
}

func (c *Checker) checkEnumStatement(stmt *ast.EnumStatement) {
	if stmt == nil || stmt.Name == nil {
		return
	}
	members := []string{}
	seen := map[string]bool{}
	for _, member := range stmt.Members {
		if seen[member.Value] {
			c.addError(fmt.Errorf("duplicate enum member %s.%s", stmt.Name.Value, member.Value))
			continue
		}
		seen[member.Value] = true
		members = append(members, member.Value)
	}
	enumType := EnumOf(stmt.Name.Value, members)
	c.enums[stmt.Name.Value] = enumType
	c.scope.defineImmutable(stmt.Name.Value, enumType)
}

func (c *Checker) checkAttributeAssignment(stmt *ast.AttributeAssignStatement) {
	if stmt == nil || stmt.Target == nil {
		return
	}
	left := c.inferExpression(stmt.Target.Object)
	value := c.inferExpression(stmt.Value)
	if left.Kind == Unknown {
		return
	}
	if left.Kind != Struct {
		c.addError(fmt.Errorf("attribute assignment requires struct, got %s", left.Kind))
		return
	}
	field, ok := left.Fields[stmt.Target.Property.Value]
	if !ok {
		c.addError(fmt.Errorf("unknown field %s on %s", stmt.Target.Property.Value, left.Name))
		return
	}
	if field.Kind != Unknown && value.Kind != Unknown && !Same(field, value) {
		c.addError(fmt.Errorf("field %s expects %s, got %s", stmt.Target.Property.Value, field.Kind, value.Kind))
	}
}

func (c *Checker) inferMatchExpression(expr *ast.MatchExpression) *Type {
	valueType := c.inferExpression(expr.Value)
	var result *Type
	for _, candidate := range expr.Cases {
		pattern := c.inferExpression(candidate.Pattern)
		if valueType.Kind != Unknown && pattern.Kind != Unknown && !Same(valueType, pattern) {
			c.addError(fmt.Errorf("match compares %s with %s", valueType.Kind, pattern.Kind))
		}
		c.pushScope()
		caseType := c.inferBlockReturnType(candidate.Body)
		c.popScope()
		if result == nil {
			result = caseType
		} else {
			result = c.unifyTypesOrError("match", result, caseType)
		}
	}
	if expr.Default != nil {
		c.pushScope()
		defaultType := c.inferBlockReturnType(expr.Default)
		c.popScope()
		if result == nil {
			result = defaultType
		} else {
			result = c.unifyTypesOrError("match", result, defaultType)
		}
	}
	if result == nil {
		return Simple(Null)
	}
	return result
}

func (c *Checker) checkStructConstructor(structType *Type, arguments []ast.Expression) *Type {
	if len(arguments) == 1 {
		if named, ok := arguments[0].(*ast.DictLiteral); ok {
			seen := map[string]bool{}
			for key, value := range named.Pairs {
				name, ok := key.(*ast.StringLiteral)
				if !ok {
					c.addError(fmt.Errorf("named struct fields must use string keys"))
					continue
				}
				fieldType, exists := structType.Fields[name.Value]
				if !exists {
					c.addError(fmt.Errorf("unknown field %s on %s", name.Value, structType.Name))
					continue
				}
				seen[name.Value] = true
				valueType := c.inferExpression(value)
				if fieldType.Kind != Unknown && valueType.Kind != Unknown && !Same(fieldType, valueType) {
					c.addError(fmt.Errorf("field %s expects %s, got %s", name.Value, fieldType.Kind, valueType.Kind))
				}
			}
			for field := range structType.Fields {
				if !seen[field] {
					c.addError(fmt.Errorf("missing field %s for %s", field, structType.Name))
				}
			}
			return structType
		}
	}
	if len(arguments) != len(structType.Fields) {
		c.addError(fmt.Errorf("struct %s expects %d fields, got %d", structType.Name, len(structType.Fields), len(arguments)))
		return structType
	}
	// Fields are maps, so use the constructor function's ordered parameters when available.
	constructor, _ := c.scope.resolve(structType.Name)
	for index, argument := range arguments {
		argType := c.inferExpression(argument)
		if constructor != nil && constructor.Kind == Func && index < len(constructor.Params) {
			expected := constructor.Params[index]
			if expected.Kind != Unknown && argType.Kind != Unknown && !Same(expected, argType) {
				c.addError(fmt.Errorf("field %d expects %s, got %s", index+1, expected.Kind, argType.Kind))
			}
		}
	}
	return structType
}
