package evaluator

import (
	"fmt"
	"math"
	"os"
	"zumbra/ast"
	"zumbra/lexer"
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
		value := Eval(node.ReturnValue, env)
		if isError(value) {
			return value
		}
		return &object.ReturnValue{Value: value}

	case *ast.VarStatement:

		if _, ok := env.Get(node.Name.Value); ok {
			return newError("variável '%s' já declarada", node.Name.Value)
		}

		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}
		env.Set(node.Name.Value, value)

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.IntegerLiteral:
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
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
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

func evalInfixExpression(operator string, left, right object.Object) object.Object {
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
		return nativeBoolToBooleanObject(isTruthy(left) && isTruthy(right))
	case "or":
		return nativeBoolToBooleanObject(isTruthy(left) || isTruthy(right))
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
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
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

func applyFunction(fct object.Object, args []object.Object) object.Object {
	switch fct := fct.(type) {
	case *object.Function:
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

func extendFunctionEnv(fct *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fct.Env)

	for paramIdx, param := range fct.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
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
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObj.Elements) - 1)

	if idx < 0 || idx > max {
		return NULL
	}

	return arrayObj.Elements[idx]
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
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.DICT_OBJ:
		return evalDictIndexExpression(left, index)
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
	path := node.Path.Value

	if env.IsImported(path) {
		return nil
	}

	env.MarkImported(path)

	content, err := os.ReadFile(path)
	if err != nil {
		return newError("Could not read imported file: %s", path)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return newError("Could not parse imported file: %s", path)
	}

	return Eval(program, env)
}
