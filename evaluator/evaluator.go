package evaluator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"zumbra/ast"
	"zumbra/collections"
	"zumbra/lexer"
	"zumbra/numeric"
	"zumbra/object"
	"zumbra/parser"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.ReturnStatement:
		if node.ReturnValue == nil {
			return &object.ReturnValue{Value: NULL}
		}

		value := Eval(node.ReturnValue, env)
		if isError(value) {
			return value
		}
		return &object.ReturnValue{Value: value}

	case *ast.AwaitExpression:
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		return value

	case *ast.TryExpression:
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		return value

	case *ast.ErrorHandlerExpression:
		value := Eval(node.Left, env)
		if isError(value) {
			handlerEnv := object.NewEnclosedEnvironment(env)

			if node.ErrorIdent != nil {
				handlerEnv.Set(node.ErrorIdent.Value, value)
			}

			handlerResult := Eval(node.Handler, handlerEnv)
			if rv, ok := handlerResult.(*object.ReturnValue); ok {
				return rv
			}
			if handlerResult == nil {
				return NULL
			}
			return handlerResult
		}
		return value

	case *ast.VarStatement:

		if _, ok := env.Get(node.Name.Value); ok {
			return newError("variável '%s' já declarada", node.Name.Value)
		}

		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		env.Set(node.Name.Value, value)

	case *ast.AssignStatement:
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		if _, ok := env.Get(node.Name.Value); !ok {
			return newError("identifier not found: %s", node.Name.Value)
		}
		env.Set(node.Name.Value, value)
		return value

	case *ast.IndexAssignStatement:
		left := Eval(node.Target.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Target.Index, env)
		if isError(index) {
			return index
		}
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		return evalIndexAssignment(left, index, value)

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.IntegerLiteral:
		if node.FixedType != "" {
			kind, ok := object.ParseFixedIntegerKind(node.FixedType)
			if !ok {
				return newError("unknown fixed integer type %s", node.FixedType)
			}
			return object.NewFixedIntegerRaw(kind, node.RawValue)
		}
		return &object.Integer{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}

	case *ast.InfixExpression:

		if node.Operator == "<<" {
			ident, ok := node.Left.(*ast.Identifier)
			if !ok {
				return newError("On << left, must be an identifier. Got %T", node.Left)
			}
			val := Eval(node.Right, env)
			if isError(val) {
				return val
			}
			env.Set(ident.Value, val)
			return val
		}

		left := Eval(node.Left, env)

		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)

		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Env: env, Body: body}

	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)

	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)

	case *ast.DictLiteral:
		return evalDictLiteral(node, env)

	case *ast.WhileStatement:
		return evalWhileStatement(node, env)

	case *ast.ImportStatement:
		return evalImportStatement(node, env)
	}

	return nil

}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	return result
}

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func nativeBoolToBooleanObject(input bool) object.Object {
	if input {
		return TRUE
	}
	return FALSE
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	if result, handled, err := numeric.Unary(operator, right); handled {
		if err != nil {
			return newError("%s", err)
		}
		return result
	}

	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	case "bnot":
		return evalBitNotPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func evalBitNotPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: bnot %s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: ^value}
}

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	if result, handled, err := numeric.Binary(operator, left, right); handled {
		if err != nil {
			return newError("%s", err)
		}
		return result
	}

	switch {
	case operator == "and" || operator == "or":
		return evalLogicalInfixExpression(operator, left, right)
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right)
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.FLOAT_OBJ:
		return evalIntLeftFloatRight(operator, left, right)
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntRightFloatLeft(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(objectEquals(left, right))
	case operator == "!=":
		return nativeBoolToBooleanObject(!objectEquals(left, right))
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func objectEquals(left, right object.Object) bool {
	if left.Type() != right.Type() {
		return false
	}

	switch left := left.(type) {
	case *object.Integer:
		return left.Value == right.(*object.Integer).Value
	case *object.FixedInteger:
		rightValue, ok := right.(*object.FixedInteger)
		return ok && left.Kind == rightValue.Kind && left.UnsignedValue() == rightValue.UnsignedValue()
	case *object.Float:
		return left.Value == right.(*object.Float).Value
	case *object.Boolean:
		return left.Value == right.(*object.Boolean).Value
	case *object.String:
		return left.Value == right.(*object.String).Value
	default:
		return left == right
	}
}

func evalLogicalInfixExpression(operator string, left, right object.Object) object.Object {
	switch operator {
	case "and":
		if isTruthy(left) {
			return right
		}
		return left
	case "or":
		if isTruthy(left) {
			return left
		}
		return right
	default:
		return newError("unknown logical operator: %s", operator)
	}
}

func evalIntegerInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "%":
		return &object.Integer{Value: int64(math.Mod(float64(leftVal), float64(rightVal)))}
	case "band":
		return &object.Integer{Value: leftVal & rightVal}
	case "bor":
		return &object.Integer{Value: leftVal | rightVal}
	case "bxor":
		return &object.Integer{Value: leftVal ^ rightVal}
	case "shl", "shr":
		if rightVal < 0 || rightVal > 63 {
			return newError("shift count must be between 0 and 63, got %d", rightVal)
		}
		if operator == "shl" {
			return &object.Integer{Value: leftVal << uint(rightVal)}
		}
		return &object.Integer{Value: leftVal >> uint(rightVal)}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalFloatInfixExpression(operator string, left object.Object, right object.Object) object.Object {
	leftVal := left.(*object.Float).Value
	rightVal := right.(*object.Float).Value

	switch operator {
	case "+":
		return &object.Float{Value: leftVal + rightVal}
	case "-":
		return &object.Float{Value: leftVal - rightVal}
	case "*":
		return &object.Float{Value: leftVal * rightVal}
	case "/":
		return &object.Float{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "%":
		return &object.Float{Value: math.Mod(float64(leftVal), rightVal)}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntLeftFloatRight(operator string, left object.Object, right object.Object) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Float).Value

	switch operator {
	case "+":
		return &object.Float{Value: float64(leftVal) + rightVal}
	case "-":
		return &object.Float{Value: float64(leftVal) - rightVal}
	case "*":
		return &object.Float{Value: float64(leftVal) * rightVal}
	case "/":
		return &object.Float{Value: float64(leftVal) / rightVal}
	case "<":
		return nativeBoolToBooleanObject(float64(leftVal) < rightVal)
	case ">":
		return nativeBoolToBooleanObject(float64(leftVal) > rightVal)
	case "==":
		return nativeBoolToBooleanObject(float64(leftVal) == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(float64(leftVal) != rightVal)
	case "%":
		return &object.Float{Value: math.Mod(float64(leftVal), rightVal)}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntRightFloatLeft(operator string, left object.Object, right object.Object) object.Object {
	leftVal := left.(*object.Float).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Float{Value: leftVal + float64(rightVal)}
	case "-":
		return &object.Float{Value: leftVal - float64(rightVal)}
	case "*":
		return &object.Float{Value: leftVal * float64(rightVal)}
	case "/":
		return &object.Float{Value: leftVal / float64(rightVal)}
	case "<":
		return nativeBoolToBooleanObject(leftVal < float64(rightVal))
	case ">":
		return nativeBoolToBooleanObject(leftVal > float64(rightVal))
	case "==":
		return nativeBoolToBooleanObject(leftVal == float64(rightVal))
	case "!=":
		return nativeBoolToBooleanObject(leftVal != float64(rightVal))
	case "%":
		return &object.Float{Value: math.Mod(leftVal, float64(rightVal))}
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}
	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func isTruthy(obj object.Object) bool {
	if obj == nil {
		return false
	}

	switch v := obj.(type) {
	case *object.Null:
		return false
	case *object.Boolean:
		return v.Value
	case *object.String:
		return v.Value != ""
	case *object.Integer:
		return v.Value != 0
	case *object.FixedInteger:
		return v.UnsignedValue() != 0
	case *object.Float:
		return v.Value != 0
	case *object.Array:
		return len(v.Elements) > 0
	case *object.Dict:
		return len(v.Pairs) > 0
	default:
		return true
	}
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func evalIdentifier(
	node *ast.Identifier,
	env *object.Environment,
) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtinsList[node.Value]; ok {
		return builtin
	}

	return newError("unknown identifier: %s", node.Value)
}

func evalExpressions(
	exps []ast.Expression,
	env *object.Environment,
) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func extendFunctionEnv(fct *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fct.Env)

	limit := len(fct.Parameters)
	if len(args) < limit {
		limit = len(args)
	}

	for paramIdx := 0; paramIdx < limit; paramIdx++ {
		env.Set(fct.Parameters[paramIdx].Value, args[paramIdx])
	}

	return env
}

func applyFunction(fct object.Object, args []object.Object) object.Object {
	switch fct := fct.(type) {
	case *object.Function:
		if len(args) != len(fct.Parameters) {
			return newError(
				"wrong number of arguments: want=%d, got=%d",
				len(fct.Parameters),
				len(args),
			)
		}

		extendedEnv := extendFunctionEnv(fct, args)
		evaluated := Eval(fct.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *object.Builtin:
		if result := fct.Fn(args...); result != nil {
			return result
		}

		return NULL

	default:
		return newError("not a function: %s", fct.Type())
	}
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func evalStringInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

	switch operator {
	case "+":
		return &object.String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalArrayIndexExpression(left, index object.Object) object.Object {
	arrayObj := left.(*object.Array)
	idx, ok := integerIndex(index)
	if !ok {
		return newError("array index must be an integer, got %s", index.Type())
	}
	max := int64(len(arrayObj.Elements) - 1)

	if idx < 0 || idx > max {
		return NULL
	}

	return arrayObj.Elements[idx]
}

func integerIndex(value object.Object) (int64, bool) {
	switch value := value.(type) {
	case *object.Integer:
		return value.Value, true
	case *object.FixedInteger:
		if value.Kind.Signed() {
			return value.SignedValue(), true
		}
		if value.UnsignedValue() > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(value.UnsignedValue()), true
	default:
		return 0, false
	}
}

func evalDictLiteral(node *ast.DictLiteral, env *object.Environment) object.Object {
	pairs := make(map[object.DictKey]object.DictPair)
	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if isError(key) {
			return key
		}
		dictKey, ok := key.(object.Dictable)
		if !ok {
			return newError("unusable as dict key: %s", key.Type())
		}
		value := Eval(valueNode, env)
		if isError(value) {
			return value
		}
		dicted := dictKey.DictKey()
		pairs[dicted] = object.DictPair{Key: key, Value: value}
	}
	return &object.Dict{Pairs: pairs}
}

func evalIndexExpression(left, index object.Object) object.Object {
	switch left.Type() {
	case object.ARRAY_OBJ:
		return evalArrayIndexExpression(left, index)
	case object.DICT_OBJ:
		return evalDictIndexExpression(left, index)
	case object.BYTE_ARRAY_OBJ, object.TYPED_ARRAY_OBJ, object.SLICE_OBJ:
		value, _, err := collections.Get(left, index)
		if err != nil {
			return newError("%s", err)
		}
		return value
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalDictIndexExpression(left, index object.Object) object.Object {
	dictObject := left.(*object.Dict)
	key, ok := index.(object.Dictable)
	if !ok {
		return newError("unusable as dict key: %s", index.Type())
	}
	dicted := key.DictKey()
	pair, ok := dictObject.Pairs[dicted]
	if !ok {
		return NULL
	}
	return pair.Value
}

func evalIndexAssignment(left, index, value object.Object) object.Object {
	switch left.Type() {
	case object.ARRAY_OBJ:
		array := left.(*object.Array)
		i, ok := integerIndex(index)
		if !ok {
			return newError("array index must be an integer, got %s", index.Type())
		}
		if i < 0 || i >= int64(len(array.Elements)) {
			return newError("array index out of bounds: %d (length %d)", i, len(array.Elements))
		}
		array.Elements[i] = value
		return value

	case object.BYTE_ARRAY_OBJ, object.TYPED_ARRAY_OBJ, object.SLICE_OBJ:
		_, err := collections.Set(left, index, value)
		if err != nil {
			return newError("%s", err)
		}
		return value

	case object.DICT_OBJ:
		dict := left.(*object.Dict)
		key, ok := index.(object.Dictable)
		if !ok {
			return newError("unusable as dict key: %s", index.Type())
		}
		dict.Pairs[key.DictKey()] = object.DictPair{Key: index, Value: value}
		return value

	default:
		return newError("index assignment not supported: %s", left.Type())
	}
}

func evalWhileStatement(ws *ast.WhileStatement, env *object.Environment) object.Object {
	var result object.Object

	for {
		condition := Eval(ws.Condition, env)
		if isError(condition) {
			return condition
		}

		if !isTruthy(condition) {
			break
		}

		result = Eval(ws.Body, env)

		if result != nil {
			if result.Type() == object.RETURN_VALUE_OBJ {
				return result
			}
		}
	}

	return result
}

func evalImportStatement(node *ast.ImportStatement, env *object.Environment) object.Object {
	if node == nil || node.Path == nil {
		return newError("import path is required")
	}

	path := strings.TrimSpace(node.Path.Value)
	if path == "" {
		return newError("import path cannot be empty")
	}

	importPath := path
	if !filepath.IsAbs(importPath) {
		absPath, err := filepath.Abs(importPath)
		if err == nil {
			importPath = absPath
		}
	}
	importPath = filepath.Clean(importPath)

	if env.IsImported(importPath) {
		return nil
	}

	content, err := os.ReadFile(importPath)
	if err != nil {
		return newError("could not read imported file: %s", path)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return newError(
			"could not parse imported file %s:\n\t%s",
			path,
			strings.Join(p.Errors(), "\n\t"),
		)
	}

	env.MarkImported(importPath)

	result := Eval(program, env)
	if isError(result) {
		return result
	}

	return result
}
