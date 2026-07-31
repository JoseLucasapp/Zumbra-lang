package numeric

import (
	"fmt"
	"math/big"

	"zumbra/object"
)

type ArithmeticMode string

const (
	Wrapping   ArithmeticMode = "wrapping"
	Checked    ArithmeticMode = "checked"
	Saturating ArithmeticMode = "saturating"
)

func Convert(kind object.FixedIntegerKind, value object.Object) (*object.FixedInteger, error) {
	switch value := value.(type) {
	case *object.Integer:
		return object.NewFixedIntegerFromInt64(kind, value.Value)
	case *object.FixedInteger:
		if value.Kind.Signed() {
			return object.NewFixedIntegerFromInt64(kind, value.SignedValue())
		}
		return object.NewFixedIntegerFromUint64(kind, value.UnsignedValue())
	default:
		return nil, fmt.Errorf("%s expects int or fixed integer, got %s", kind, value.Type())
	}
}

func Binary(operator string, left, right object.Object) (object.Object, bool, error) {
	_, leftOK := left.(*object.FixedInteger)
	_, rightOK := right.(*object.FixedInteger)
	if !leftOK && !rightOK {
		return nil, false, nil
	}

	if operator == "shl" || operator == "shr" {
		return shift(operator, left, right)
	}

	kind, normalizedLeft, normalizedRight, err := normalizePair(left, right)
	if err != nil {
		return nil, true, err
	}

	leftRaw := normalizedLeft.UnsignedValue()
	rightRaw := normalizedRight.UnsignedValue()
	mask := kind.Mask()

	switch operator {
	case "+":
		return object.NewFixedIntegerRaw(kind, (leftRaw+rightRaw)&mask), true, nil
	case "-":
		return object.NewFixedIntegerRaw(kind, (leftRaw-rightRaw)&mask), true, nil
	case "*":
		return object.NewFixedIntegerRaw(kind, (leftRaw*rightRaw)&mask), true, nil
	case "/", "%":
		if rightRaw == 0 {
			return nil, true, fmt.Errorf("division by zero")
		}
		if kind.Signed() {
			leftValue := normalizedLeft.SignedValue()
			rightValue := normalizedRight.SignedValue()
			if rightValue == 0 {
				return nil, true, fmt.Errorf("division by zero")
			}
			min, _ := kind.SignedBounds()
			if leftValue == min && rightValue == -1 {
				if operator == "/" {
					return object.NewFixedIntegerRaw(kind, uint64(min)), true, nil
				}
				return object.NewFixedIntegerRaw(kind, 0), true, nil
			}
			if operator == "/" {
				return object.NewFixedIntegerRaw(kind, uint64(leftValue/rightValue)), true, nil
			}
			return object.NewFixedIntegerRaw(kind, uint64(leftValue%rightValue)), true, nil
		}
		if operator == "/" {
			return object.NewFixedIntegerRaw(kind, leftRaw/rightRaw), true, nil
		}
		return object.NewFixedIntegerRaw(kind, leftRaw%rightRaw), true, nil
	case "**":
		if kind.Signed() && normalizedRight.SignedValue() < 0 {
			return nil, true, fmt.Errorf("fixed integer exponent must be non-negative")
		}
		result := uint64(1)
		base := leftRaw
		exponent := rightRaw
		for exponent > 0 {
			if exponent&1 == 1 {
				result = (result * base) & mask
			}
			base = (base * base) & mask
			exponent >>= 1
		}
		return object.NewFixedIntegerRaw(kind, result), true, nil
	case "band":
		return object.NewFixedIntegerRaw(kind, leftRaw&rightRaw), true, nil
	case "bor":
		return object.NewFixedIntegerRaw(kind, leftRaw|rightRaw), true, nil
	case "bxor":
		return object.NewFixedIntegerRaw(kind, leftRaw^rightRaw), true, nil
	case "==", "!=", "<", ">", "<=", ">=":
		value := compare(operator, normalizedLeft, normalizedRight)
		return &object.Boolean{Value: value}, true, nil
	default:
		return nil, true, fmt.Errorf("unsupported fixed integer operator %q", operator)
	}

}

func Unary(operator string, value object.Object) (object.Object, bool, error) {
	fixed, ok := value.(*object.FixedInteger)
	if !ok {
		return nil, false, nil
	}

	switch operator {
	case "bnot":
		return object.NewFixedIntegerRaw(fixed.Kind, ^fixed.UnsignedValue()), true, nil
	case "-":
		return object.NewFixedIntegerRaw(fixed.Kind, -fixed.UnsignedValue()), true, nil
	default:
		return nil, true, fmt.Errorf("unsupported fixed integer prefix operator %q", operator)
	}
}

func Arithmetic(mode ArithmeticMode, operator string, left, right object.Object) (object.Object, error) {
	kind, normalizedLeft, normalizedRight, err := normalizePair(left, right)
	if err != nil {
		return nil, err
	}
	if operator != "+" && operator != "-" && operator != "*" {
		return nil, fmt.Errorf("unsupported arithmetic operation %q", operator)
	}

	result := calculateBig(operator, normalizedLeft, normalizedRight)
	min, max := bigBounds(kind)

	switch mode {
	case Checked:
		if result.Cmp(min) < 0 || result.Cmp(max) > 0 {
			return nil, fmt.Errorf("%s overflow in %s operation", kind, operationName(operator))
		}
		return fixedFromBig(kind, result)
	case Saturating:
		if result.Cmp(min) < 0 {
			result = new(big.Int).Set(min)
		}
		if result.Cmp(max) > 0 {
			result = new(big.Int).Set(max)
		}
		return fixedFromBig(kind, result)
	case Wrapping:
		modulus := new(big.Int).Lsh(big.NewInt(1), uint(kind.Bits()))
		result.Mod(result, modulus)
		if result.Sign() < 0 {
			result.Add(result, modulus)
		}
		return object.NewFixedIntegerRaw(kind, result.Uint64()), nil
	default:
		return nil, fmt.Errorf("unknown arithmetic mode %q", mode)
	}
}

func normalizePair(left, right object.Object) (object.FixedIntegerKind, *object.FixedInteger, *object.FixedInteger, error) {
	leftFixed, leftOK := left.(*object.FixedInteger)
	rightFixed, rightOK := right.(*object.FixedInteger)

	if !leftOK && !rightOK {
		return "", nil, nil, fmt.Errorf("fixed integer operation requires at least one fixed integer")
	}

	kind := object.FixedIntegerKind("")
	if leftOK {
		kind = leftFixed.Kind
	}
	if rightOK {
		if kind != "" && rightFixed.Kind != kind {
			return "", nil, nil, fmt.Errorf("fixed integer types must match: %s and %s", kind, rightFixed.Kind)
		}
		kind = rightFixed.Kind
	}

	if !leftOK {
		converted, err := Convert(kind, left)
		if err != nil {
			return "", nil, nil, err
		}
		leftFixed = converted
	}
	if !rightOK {
		converted, err := Convert(kind, right)
		if err != nil {
			return "", nil, nil, err
		}
		rightFixed = converted
	}

	return kind, leftFixed, rightFixed, nil
}

func shift(operator string, left, right object.Object) (object.Object, bool, error) {
	count, err := shiftCount(right)
	if err != nil {
		return nil, true, err
	}

	switch left := left.(type) {
	case *object.FixedInteger:
		if count >= uint64(left.Kind.Bits()) {
			return nil, true, fmt.Errorf("shift count for %s must be between 0 and %d, got %d", left.Kind, left.Kind.Bits()-1, count)
		}
		if operator == "shl" {
			return object.NewFixedIntegerRaw(left.Kind, left.UnsignedValue()<<count), true, nil
		}
		if left.Kind.Signed() {
			return object.NewFixedIntegerRaw(left.Kind, uint64(left.SignedValue()>>count)), true, nil
		}
		return object.NewFixedIntegerRaw(left.Kind, left.UnsignedValue()>>count), true, nil
	case *object.Integer:
		if count > 63 {
			return nil, true, fmt.Errorf("shift count must be between 0 and 63, got %d", count)
		}
		if operator == "shl" {
			return &object.Integer{Value: left.Value << count}, true, nil
		}
		return &object.Integer{Value: left.Value >> count}, true, nil
	default:
		return nil, true, fmt.Errorf("left shift operand must be an integer, got %s", left.Type())
	}
}

func shiftCount(value object.Object) (uint64, error) {
	switch value := value.(type) {
	case *object.Integer:
		if value.Value < 0 {
			return 0, fmt.Errorf("shift count must be non-negative, got %d", value.Value)
		}
		return uint64(value.Value), nil
	case *object.FixedInteger:
		if value.Kind.Signed() && value.SignedValue() < 0 {
			return 0, fmt.Errorf("shift count must be non-negative, got %d", value.SignedValue())
		}
		return value.UnsignedValue(), nil
	default:
		return 0, fmt.Errorf("shift count must be an integer, got %s", value.Type())
	}
}

func compare(operator string, left, right *object.FixedInteger) bool {
	var comparison int
	if left.Kind.Signed() {
		leftValue := left.SignedValue()
		rightValue := right.SignedValue()
		switch {
		case leftValue < rightValue:
			comparison = -1
		case leftValue > rightValue:
			comparison = 1
		}
	} else {
		leftValue := left.UnsignedValue()
		rightValue := right.UnsignedValue()
		switch {
		case leftValue < rightValue:
			comparison = -1
		case leftValue > rightValue:
			comparison = 1
		}
	}

	switch operator {
	case "==":
		return comparison == 0
	case "!=":
		return comparison != 0
	case "<":
		return comparison < 0
	case ">":
		return comparison > 0
	case "<=":
		return comparison <= 0
	case ">=":
		return comparison >= 0
	default:
		return false
	}
}

func calculateBig(operator string, left, right *object.FixedInteger) *big.Int {
	leftValue := fixedToBig(left)
	rightValue := fixedToBig(right)

	switch operator {
	case "+":
		return new(big.Int).Add(leftValue, rightValue)
	case "-":
		return new(big.Int).Sub(leftValue, rightValue)
	case "*":
		return new(big.Int).Mul(leftValue, rightValue)
	default:
		return big.NewInt(0)
	}
}

func fixedToBig(value *object.FixedInteger) *big.Int {
	if value.Kind.Signed() {
		return big.NewInt(value.SignedValue())
	}
	return new(big.Int).SetUint64(value.UnsignedValue())
}

func bigBounds(kind object.FixedIntegerKind) (*big.Int, *big.Int) {
	if kind.Signed() {
		min, max := kind.SignedBounds()
		return big.NewInt(min), big.NewInt(max)
	}
	return big.NewInt(0), new(big.Int).SetUint64(kind.Mask())
}

func fixedFromBig(kind object.FixedIntegerKind, value *big.Int) (*object.FixedInteger, error) {
	if kind.Signed() {
		return object.NewFixedIntegerFromInt64(kind, value.Int64())
	}
	return object.NewFixedIntegerFromUint64(kind, value.Uint64())
}

func operationName(operator string) string {
	switch operator {
	case "+":
		return "addition"
	case "-":
		return "subtraction"
	case "*":
		return "multiplication"
	default:
		return operator
	}
}
