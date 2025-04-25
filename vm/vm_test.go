package vm

import (
	"fmt"
	"testing"
	"zumbra/ast"
	"zumbra/compiler"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/parser"
)

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func testIntegerObject(expected int64, actual object.Object) error {
	result, ok := actual.(*object.Integer)

	if !ok {
		return fmt.Errorf("object is not *object.Integer. got=%T (%+v)", actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. want=%d, got=%d", expected, result.Value)
	}

	return nil
}

type vmTestCase struct {
	input    string
	expected interface{}
}

func runVmTests(t *testing.T, tests []vmTestCase) {
	t.Helper()

	for _, tt := range tests {
		program := parse(tt.input)
		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		/*
			for i, constant := range comp.Bytecode().Constants {
				fmt.Printf("CONSTANT %d %p (%T):\n", i, constant, constant)
				switch constant := constant.(type) {
				case *object.CompiledFunction:
					fmt.Printf(" Instructions:\n%s", constant.Instructions)
				case *object.Integer:
					fmt.Printf(" Value: %d\n", constant.Value)
				}
				fmt.Printf("\n")
			}*/

		vm := New(comp.Bytecode())
		err = vm.Run()
		if err != nil {
			t.Fatalf("vm error: %s", err)
		}

		stackElem := vm.LastPoppedStackElem()

		testExpectedObject(t, tt.expected, stackElem)
	}
}

func testExpectedObject(t *testing.T, expected interface{}, actual object.Object) {
	t.Helper()

	switch expected := expected.(type) {
	case int:
		err := testIntegerObject(int64(expected), actual)
		if err != nil {
			t.Errorf("testIntegerObject failed: %s", err)
		}

	case bool:
		err := testBooleanObject(bool(expected), actual)
		if err != nil {
			t.Errorf("testBooleanObject failed: %s", err)
		}

	case *object.Null:
		if actual != Null {
			t.Errorf("object is not *object.Null: %T (%+v)", actual, actual)
		}

	case string:
		err := testStringObject(expected, actual)

		if err != nil {
			t.Errorf("testStringObject failed: %s", err)
		}

	case []int:
		array, ok := actual.(*object.Array)

		if !ok {
			t.Errorf("object not Array: %T (%+v)", actual, actual)
			return
		}

		if len(array.Elements) != len(expected) {
			t.Errorf("wrong num of elements. want=%d, got=%d",
				len(expected), len(array.Elements))
			return
		}

		for i, expectedElem := range expected {
			err := testIntegerObject(int64(expectedElem), array.Elements[i])
			if err != nil {
				t.Errorf("testIntegerObject failed: %s", err)
			}
		}

	case map[object.DictKey]int64:
		dict, ok := actual.(*object.Dict)
		if !ok {
			t.Errorf("object is not Dict. got=%T (%+v)", actual, actual)
			return
		}

		if len(dict.Pairs) != len(expected) {
			t.Errorf("dict has wrong number of Pairs. want=%d, got=%d",
				len(expected), len(dict.Pairs))
			return
		}

		for expectedKey, expectedValue := range expected {
			pair, ok := dict.Pairs[expectedKey]
			if !ok {
				t.Errorf("no pair for given key in Pairs")
			}

			err := testIntegerObject(expectedValue, pair.Value)
			if err != nil {
				t.Errorf("testIntegerObject failed: %s", err)
			}
		}

	case *object.Error:
		errObj, ok := actual.(*object.Error)
		if !ok {
			t.Errorf("object is not Error: %T (%+v)", actual, actual)
			return
		}
		if errObj.Message != expected.Message {
			t.Errorf("wrong error message. expected=%q, got=%q",
				expected.Message, errObj.Message)
		}

	}
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []vmTestCase{
		{"1", 1},
		{"2", 2},
		{"1 + 2", 3},
		{"1 - 2", -1},
		{"1 * 2", 2},
		{"4 / 2", 2},
		{"50 / 2 * 2 + 10 - 5", 55},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"5 * (2 + 10)", 60},
		{"-5", -5},
		{"-10", -10},
		{"-50 + 100 + -50", 0},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
	}

	runVmTests(t, tests)
}

func TestBooleanExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"false != true", true},
		{"(1 < 2) == true", true},
		{"(1 < 2) == false", false},
		{"(1 > 2) == true", false},
		{"(1 > 2) == false", true},
		{"!true", false},
		{"!false", true},
		{"!5", false},
		{"!!true", true},
		{"!!false", false},
		{"!!5", true},
		{"!(if (false) { 5; })", true},
	}
	runVmTests(t, tests)
}

func testBooleanObject(expected bool, actual object.Object) error {
	result, ok := actual.(*object.Boolean)

	if !ok {
		return fmt.Errorf("object is not *object.Boolean. got=%T (%+v)", actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. want=%t, got=%t", expected, result.Value)
	}

	return nil
}

func TestConditionals(t *testing.T) {
	tests := []vmTestCase{
		{"if (true) { 10 }", 10},
		{"if (true) { 10 } else { 20 }", 10},
		{"if (false) { 10 } else { 20 } ", 20},
		{"if (1) { 10 }", 10},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 < 2) { 10 } else { 20 }", 10},
		{"if (1 > 2) { 10 } else { 20 }", 20},
		{"if (1 > 2) { 10 }", Null},
		{"if (false) { 10 }", Null},
		{"if ((if (false) { 10 })) { 10 } else { 20 }", 20},
	}
	runVmTests(t, tests)
}

func TestGlobalVarStatements(t *testing.T) {
	tests := []vmTestCase{
		{"var one << 1; one", 1},
		{"var one << 1; var two << 2; one + two", 3},
		{"var one << 1; var two << one + one; one + two", 3},
	}
	runVmTests(t, tests)
}

func TestStringExpressions(t *testing.T) {
	tests := []vmTestCase{
		{`"Zumbra"`, "Zumbra"},
		{`"Zum" + "bra"`, "Zumbra"},
		{`"Zum" + "bra" + "lang"`, "Zumbralang"},
	}
	runVmTests(t, tests)
}

func testStringObject(expected string, actual object.Object) error {
	result, ok := actual.(*object.String)

	if !ok {
		return fmt.Errorf("object is not String. got=%T (%+v)",
			actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%q, want=%q",
			result.Value, expected)

	}

	return nil
}

func TestArrayLiterals(t *testing.T) {
	tests := []vmTestCase{
		{"[]", []int{}},
		{"[1, 2, 3]", []int{1, 2, 3}},
		{"[1 + 2, 3 * 4, 5 + 6]", []int{3, 12, 11}},
	}
	runVmTests(t, tests)
}

func TestDictLiterals(t *testing.T) {
	tests := []vmTestCase{
		{
			"{}", map[object.DictKey]int64{},
		},
		{
			"{1: 2, 2: 3}",
			map[object.DictKey]int64{
				(&object.Integer{Value: 1}).DictKey(): 2,
				(&object.Integer{Value: 2}).DictKey(): 3,
			},
		},
		{
			"{1 + 1: 2 * 2, 3 + 3: 4 * 4}",
			map[object.DictKey]int64{
				(&object.Integer{Value: 2}).DictKey(): 4,
				(&object.Integer{Value: 6}).DictKey(): 16,
			},
		},
	}
	runVmTests(t, tests)
}

func TestIndexExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"[1, 2, 3][1]", 2},
		{"[1, 2, 3][0 + 2]", 3},
		{"[[1, 1, 1]][0][0]", 1},
		{"[][0]", Null},
		{"[1, 2, 3][99]", Null},
		{"[1][-1]", Null},
		{"{1: 1, 2: 2}[1]", 1},
		{"{1: 1, 2: 2}[2]", 2},
		{"{1: 1}[0]", Null},
		{"{}[0]", Null},
	}
	runVmTests(t, tests)
}

func TestCallingFunctionsWithoutArguments(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	var fivePlusTen << fct() { 5 + 10; };
	fivePlusTen();
	`,
			expected: 15,
		},
		{
			input: `
			var one<< fct() { 1; };
			var two << fct() { 2; };
			one() + two()
			`,
			expected: 3,
		},
		{
			input: `
			var a << fct() { 1 };
			var b << fct() { a() + 1 };
			var c << fct() { b() + 1 };
			c();
			`,
			expected: 3,
		},
	}
	runVmTests(t, tests)
}

func TestFunctionsWithReturnStatement(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
		var earlyExit<< fct() { return 99; 100; };
		earlyExit();
		`,
			expected: 99,
		},
		{
			input: `
		var earlyExit << fct() { return 99; return 100; };
		earlyExit();
		`,
			expected: 99,
		},
	}
	runVmTests(t, tests)
}

func TestFunctionsWithoutReturnValue(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	var noReturn << fct() { };
	noReturn();
	`,
			expected: Null,
		},
		{
			input: `
	var noReturn << fct() { };
	var noReturnTwo << fct() { noReturn(); };
	noReturn();
	noReturnTwo();
	`,
			expected: Null,
		},
	}
	runVmTests(t, tests)
}

func TestFirstClassFunctions(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	var returnsOne << fct() { 1; };
	var returnsOneReturner << fct() { returnsOne; };
	returnsOneReturner()();
	`,
			expected: 1,
		},
		{
			input: `
	var returnsOneReturner << fct() {
	var returnsOne << fct() { 1; };
	returnsOne;
	};
	returnsOneReturner()();
	`,
			expected: 1,
		},
	}
	runVmTests(t, tests)
}

func TestCallingFunctionsWithBindings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	var one << fct() { var one << 1; one };
	one();
	`,

			expected: 1,
		},
		{
			input: `
	var oneAndTwo << fct() { var one << 1; var two << 2; one + two; };
	oneAndTwo();
	`,
			expected: 3,
		},
		{
			input: `
	var oneAndTwo << fct() { var one << 1; var two << 2; one + two; };
	var threeAndFour << fct() { var three << 3; var four << 4; three + four; };
	oneAndTwo() + threeAndFour();
	`,
			expected: 10,
		},
		{
			input: `
	var firstFoobar << fct() { var foobar << 50; foobar; };
	var secondFoobar << fct() { var foobar << 100; foobar; };
	firstFoobar() + secondFoobar();
	`,
			expected: 150,
		},
	}
	runVmTests(t, tests)
}

func TestCallingFunctionsWithArgumentsAndBindings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	207
	var identity << fct(a) { a; };
	identity(4);
	`,
			expected: 4,
		},
		{
			input: `
	var sum << fct(a, b) { a + b; };
	sum(1, 2);
	`,
			expected: 3,
		},
		{
			input: `
			var sum << fct(a, b) {
			var c << a + b;
			c;
			};
			sum(1, 2);
			`,
			expected: 3,
		},
		{
			input: `
			var sum << fct(a, b) {
			var c << a + b;
			c;
			};
			sum(1, 2) + sum(3, 4);`,
			expected: 10,
		},
		{
			input: `
			var sum << fct(a, b) {
			var c << a + b;
			c;
			};
			var outer << fct() {
			sum(1, 2) + sum(3, 4);
			};
			outer();
			`,
			expected: 10,
		},
		{
			input: `
			var globalNum << 10;
			var sum << fct(a, b) {
			var c << a + b;
			c + globalNum;
			};
			var outer << fct() {
			sum(1, 2) + sum(3, 4) + globalNum;
			};
			outer() + globalNum;
			`,
			expected: 50,
		},
	}
	runVmTests(t, tests)
}

func TestCallingFunctionsWithWrongArguments(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    `fct() { 1; }(1);`,
			expected: `wrong number of arguments: want=0, got=1`,
		},
		{
			input:    `fct(a) { a; }();`,
			expected: `wrong number of arguments: want=1, got=0`,
		},
		{
			input:    `fct(a, b) { a + b; }(1);`,
			expected: `wrong number of arguments: want=2, got=1`,
		},
	}
	for _, tt := range tests {
		program := parse(tt.input)
		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}
		vm := New(comp.Bytecode())
		err = vm.Run()
		if err == nil {
			t.Fatalf("expected VM error but resulted in none.")
		}
		if err.Error() != tt.expected {
			t.Fatalf("wrong VM error: want=%q, got=%q", tt.expected, err)
		}
	}
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []vmTestCase{
		{`sizeOf("")`, 0},
		{`sizeOf("four")`, 4},
		{`sizeOf("hello world")`, 11},
		{
			`sizeOf(1)`,
			&object.Error{
				Message: "argument to `sizeOf` not supported, got INTEGER",
			},
		},
		{`sizeOf("one", "two")`,
			&object.Error{
				Message: "wrong number of arguments. got=2, want=1",
			},
		},
		{`sizeOf([1, 2, 3])`, 3},
		{`sizeOf([])`, 0},
		{`show("hello", "world!")`, Null},
		{`first([1, 2, 3])`, 1},
		{`first([])`, Null},
		{`first(1)`,
			&object.Error{
				Message: "argument to `first` must be ARRAY, got INTEGER",
			},
		},
		{`last([1, 2, 3])`, 3},
		{`last([])`, Null},
		{`last(1)`,
			&object.Error{
				Message: "argument to `last` must be ARRAY, got INTEGER",
			},
		},
		{`allButFirst([1, 2, 3])`, []int{2, 3}},
		{`allButFirst([])`, Null},
		{`addToArray([], 1)`, []int{1}},
		{`addToArray(1, 1)`,
			&object.Error{
				Message: "argument to `addToArray` must be ARRAY, got INTEGER",
			},
		},
	}
	runVmTests(t, tests)
}

func TestClosures(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	var newClosure << fct(a) {
	fct() { a; };
	};
	var closure << newClosure(99);
	closure();
	`,
			expected: 99,
		},
		{
			input: `
			var newAdder << fct(a, b) {
			fct(c) { a + b + c };
			};
			var adder << newAdder(1, 2);
			adder(8);
			`,
			expected: 11,
		},
		{
			input: `
			var newAdder << fct(a, b) {
			var c << a + b;
			fct(d) { c + d };
			};
			var adder << newAdder(1, 2);
			adder(8);
			`,
			expected: 11,
		},
		{
			input: `
			var newAdderOuter << fct(a, b) {
			var c << a + b;
			fct(d) {
			var e << d + c;
			fct(f) { e + f; };
			};
			};
			var newAdderInner << newAdderOuter(1, 2);
			var adder << newAdderInner(3);
			adder(8);
			`,
			expected: 14,
		},
		{
			input: `
			var a << 1;
			var newAdderOuter << fct(b) {
			fct(c) {
			fct(d) { a + b + c + d };
			};
			};
			var newAdderInner << newAdderOuter(2);
			var adder << newAdderInner(3);
			adder(8);
			`,
			expected: 14,
		},
		{
			input: `
			var newClosure << fct(a, b) {
			var one << fct() { a; };
			var two << fct() { b; };
			fct() { one() + two(); };
			};
			var closure << newClosure(9, 90);
			closure();
			`,
			expected: 99,
		},
	}
	runVmTests(t, tests)
}

func TestRecursiveFunctions(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	var countDown << fct(x) {
	if (x == 0) {
	return 0;
	} else {
	countDown(x - 1);
	}
	};
	countDown(1);
	`,
			expected: 0,
		},
		{
			input: `
			var countDown << fct(x) {
			if (x == 0) {
			return 0;
			} else {
			countDown(x - 1);
			}
			};
			var wrapper << fct() {
			countDown(1);
			};
			wrapper();
			`,
			expected: 0,
		},
		{
			input: `
			var wrapper << fct() {
			var countDown << fct(x) {
				if (x == 0) {
					return 0;
				} else {
				countDown(x - 1);
				};
			};
			
			countDown(1);
			};
			wrapper();
			`,
			expected: 0,
		},
	}
	runVmTests(t, tests)
}

func TestRecursiveFibonacci(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
	var fibonacci << fct(x) {
	if (x == 0) {
	return 0;
	} else {
	if (x == 1) {
	return 1;
	} else {
	fibonacci(x - 1) + fibonacci(x - 2);
	}
	}
	};
	fibonacci(15);
	`,
			expected: 610,
		},
	}
	runVmTests(t, tests)
}

func TestAssign(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
				var x << 0;
				x << 1;
				x;
			`,
			expected: 1,
		},
	}
	runVmTests(t, tests)
}

func TestWhileLoops(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
				var x << 0;
				while (x < 3) {
					x << x + 1;
				}
				x
			`,
			expected: 3,
		},
		{
			input: `
				var result << 1;
				var i << 1;
				while (i < 4) {
					result << result * i;
					i << i + 1;
				}
				result
			`,
			expected: 6, // 1 * 2 * 3
		},
	}
	runVmTests(t, tests)
}
