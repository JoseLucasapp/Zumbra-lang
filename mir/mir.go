// Package mir defines Zumbra's structured middle-level IR. Expressions are
// converted to explicit virtual values while control flow remains represented
// by regions. This makes the IR compact, deterministic and suitable for the VM,
// native backends and debugging tools.
package mir

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"zumbra/hir"
	"zumbra/types"
)

type ValueID int

type Op string

const (
	OpConst       Op = "const"
	OpLoad        Op = "load"
	OpDeclare     Op = "declare"
	OpStore       Op = "store"
	OpUnary       Op = "unary"
	OpBinary      Op = "binary"
	OpCall        Op = "call"
	OpArray       Op = "array"
	OpDict        Op = "dict"
	OpPair        Op = "pair"
	OpIndex       Op = "index"
	OpSetIndex    Op = "set_index"
	OpField       Op = "field"
	OpSetField    Op = "set_field"
	OpIf          Op = "if"
	OpMatch       Op = "match"
	OpCase        Op = "case"
	OpWhile       Op = "while"
	OpFor         Op = "for"
	OpForEach     Op = "for_each"
	OpForRange    Op = "for_range"
	OpForever     Op = "forever"
	OpReturn      Op = "return"
	OpBreak       Op = "break"
	OpContinue    Op = "continue"
	OpDrop        Op = "drop"
	OpSpawn       Op = "spawn"
	OpAwait       Op = "await"
	OpTry         Op = "try"
	OpHandler     Op = "error_handler"
	OpImport      Op = "import"
	OpExtern      Op = "extern"
	OpUnsafe      Op = "unsafe"
	OpTypeAlias   Op = "type_alias"
	OpStruct      Op = "struct"
	OpStructField Op = "struct_field"
	OpEnum        Op = "enum"
	OpFunctionRef Op = "function_ref"
	OpUnknown     Op = "unknown"
)

type Instruction struct {
	ID       int
	Op       Op
	Result   ValueID
	Type     *types.Type
	Name     string
	Operator string
	Literal  string
	Args     []ValueID
	Regions  []*Region
	Meta     map[string]string
	SourceID int
}

type Region struct {
	Name         string
	Instructions []*Instruction
	Result       ValueID
}

type Function struct {
	Name       string
	Owner      string
	Parameters []string
	ReturnType *types.Type
	Body       *Region
	Async      bool
	Method     bool
}

type Module struct {
	Filename     string
	Entry        *Region
	Functions    []*Function
	Declarations []*Instruction
	Optimized    bool
}

type lowerer struct {
	nextInstruction int
	nextValue       ValueID
	module          *Module
}

func Lower(module *hir.Module) (*Module, error) {
	if module == nil || module.Root == nil {
		return nil, fmt.Errorf("cannot lower nil HIR module")
	}
	result := &Module{Filename: module.Filename, Entry: &Region{Name: "entry"}}
	l := &lowerer{module: result}
	for _, child := range module.Root.Children {
		l.lowerStatement(result.Entry, child)
	}
	if err := Verify(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (l *lowerer) newInstruction(op Op, source *hir.Node) *Instruction {
	l.nextInstruction++
	inst := &Instruction{ID: l.nextInstruction, Op: op, Meta: map[string]string{}}
	if source != nil {
		inst.Type = types.Clone(source.Type)
		inst.Name = source.Name
		inst.Operator = source.Operator
		inst.Literal = source.Literal
		inst.SourceID = source.ID
		for key, value := range source.Meta {
			inst.Meta[key] = value
		}
	}
	return inst
}

func (l *lowerer) value(inst *Instruction) ValueID {
	l.nextValue++
	inst.Result = l.nextValue
	return inst.Result
}

func (l *lowerer) emit(region *Region, inst *Instruction) {
	region.Instructions = append(region.Instructions, inst)
}

func (l *lowerer) lowerStatement(region *Region, node *hir.Node) {
	if node == nil {
		return
	}
	switch node.Kind {
	case hir.VarKind, hir.ConstKind:
		value := l.lowerExpression(region, first(node.Children))
		inst := l.newInstruction(OpDeclare, node)
		inst.Args = appendArg(inst.Args, value)
		if node.Kind == hir.ConstKind {
			inst.Meta["mutable"] = "false"
		} else {
			inst.Meta["mutable"] = "true"
		}
		l.emit(region, inst)
	case hir.AssignKind:
		value := l.lowerExpression(region, first(node.Children))
		inst := l.newInstruction(OpStore, node)
		inst.Args = appendArg(inst.Args, value)
		l.emit(region, inst)
	case hir.SetIndexKind:
		inst := l.newInstruction(OpSetIndex, node)
		for _, child := range node.Children {
			inst.Args = appendArg(inst.Args, l.lowerExpression(region, child))
		}
		l.emit(region, inst)
	case hir.SetFieldKind:
		inst := l.newInstruction(OpSetField, node)
		for _, child := range node.Children {
			inst.Args = appendArg(inst.Args, l.lowerExpression(region, child))
		}
		l.emit(region, inst)
	case hir.ExprStmtKind:
		value := l.lowerExpression(region, first(node.Children))
		if value != 0 {
			inst := l.newInstruction(OpDrop, node)
			inst.Args = []ValueID{value}
			l.emit(region, inst)
		}
	case hir.ReturnKind:
		inst := l.newInstruction(OpReturn, node)
		if len(node.Children) > 0 {
			inst.Args = []ValueID{l.lowerExpression(region, node.Children[0])}
		}
		l.emit(region, inst)
	case hir.WhileKind:
		inst := l.newInstruction(OpWhile, node)
		inst.Regions = []*Region{l.lowerExpressionRegion("condition", childAt(node.Children, 0)), l.lowerBlock("body", childAt(node.Children, 1))}
		l.emit(region, inst)
	case hir.ImportKind:
		l.module.Declarations = append(l.module.Declarations, l.newInstruction(OpImport, node))
	case hir.ExternKind:
		for _, child := range node.Children {
			declaration := l.newInstruction(OpExtern, child)
			for key, value := range node.Meta {
				if _, exists := declaration.Meta[key]; !exists {
					declaration.Meta[key] = value
				}
			}
			l.module.Declarations = append(l.module.Declarations, declaration)
		}
	case hir.UnsafeKind:
		inst := l.newInstruction(OpUnsafe, node)
		if len(node.Children) > 0 {
			inst.Regions = append(inst.Regions, l.lowerBlock("unsafe", node.Children[0]))
		}
		l.emit(region, inst)
	case hir.TypeAliasKind:
		l.module.Declarations = append(l.module.Declarations, l.newInstruction(OpTypeAlias, node))
	case hir.StructKind:
		decl := l.newInstruction(OpStruct, node)
		for _, child := range node.Children {
			if child.Kind == hir.StructFieldKind {
				field := l.newInstruction(OpStructField, child)
				decl.Regions = append(decl.Regions, &Region{Name: "field." + child.Name, Instructions: []*Instruction{field}})
			} else if child.Kind == hir.MethodKind {
				l.lowerFunction(child, true)
			}
		}
		l.module.Declarations = append(l.module.Declarations, decl)
	case hir.EnumKind:
		decl := l.newInstruction(OpEnum, node)
		for _, child := range node.Children {
			decl.Meta[child.Name] = child.Literal
		}
		l.module.Declarations = append(l.module.Declarations, decl)
	default:
		value := l.lowerExpression(region, node)
		if value != 0 {
			inst := l.newInstruction(OpDrop, node)
			inst.Args = []ValueID{value}
			l.emit(region, inst)
		}
	}
}

func (l *lowerer) lowerExpression(region *Region, node *hir.Node) ValueID {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case hir.IntegerKind, hir.FloatKind, hir.StringKind, hir.BooleanKind:
		inst := l.newInstruction(OpConst, node)
		// Preserve the source literal category even when contextual type analysis
		// leaves the expression as unknown (for example, dictionary access inside
		// a for-each body). Native backends must not guess a string literal is an
		// integer merely because its inferred type is unknown.
		inst.Meta["literal_kind"] = string(node.Kind)
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.IdentifierKind:
		inst := l.newInstruction(OpLoad, node)
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.PrefixKind:
		inst := l.newInstruction(OpUnary, node)
		inst.Args = []ValueID{l.lowerExpression(region, first(node.Children))}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.BinaryKind:
		left := l.lowerExpression(region, childAt(node.Children, 0))
		right := l.lowerExpression(region, childAt(node.Children, 1))
		inst := l.newInstruction(OpBinary, node)
		inst.Args = []ValueID{left, right}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.CallKind:
		inst := l.newInstruction(OpCall, node)
		for _, child := range node.Children {
			inst.Args = appendArg(inst.Args, l.lowerExpression(region, child))
		}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.ArrayKind:
		inst := l.newInstruction(OpArray, node)
		for _, child := range node.Children {
			inst.Args = appendArg(inst.Args, l.lowerExpression(region, child))
		}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.DictKind:
		inst := l.newInstruction(OpDict, node)
		for _, child := range node.Children {
			inst.Args = appendArg(inst.Args, l.lowerExpression(region, child))
		}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.PairKind:
		inst := l.newInstruction(OpPair, node)
		for _, child := range node.Children {
			inst.Args = appendArg(inst.Args, l.lowerExpression(region, child))
		}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.IndexKind:
		inst := l.newInstruction(OpIndex, node)
		for _, child := range node.Children {
			inst.Args = appendArg(inst.Args, l.lowerExpression(region, child))
		}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.FieldKind:
		inst := l.newInstruction(OpField, node)
		inst.Args = []ValueID{l.lowerExpression(region, first(node.Children))}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.FunctionKind, hir.MethodKind:
		fn := l.lowerFunction(node, node.Kind == hir.MethodKind)
		inst := l.newInstruction(OpFunctionRef, node)
		inst.Name = fn.Name
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.IfKind:
		condition := l.lowerExpression(region, childAt(node.Children, 0))
		inst := l.newInstruction(OpIf, node)
		inst.Args = []ValueID{condition}
		inst.Regions = append(inst.Regions, l.lowerBlock("then", childAt(node.Children, 1)))
		if len(node.Children) > 2 {
			inst.Regions = append(inst.Regions, l.lowerBlock("else", childAt(node.Children, 2)))
		}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.MatchKind:
		inst := l.newInstruction(OpMatch, node)
		inst.Args = []ValueID{l.lowerExpression(region, first(node.Children))}
		for _, child := range node.Children[1:] {
			caseRegion := &Region{Name: "case"}
			caseInst := l.newInstruction(OpCase, child)
			caseInst.Name = child.Name
			if child.Name == "else" {
				caseInst.Regions = []*Region{l.lowerBlock("body", first(child.Children))}
			} else {
				caseInst.Args = []ValueID{l.lowerExpression(caseRegion, childAt(child.Children, 0))}
				caseInst.Regions = []*Region{l.lowerBlock("body", childAt(child.Children, 1))}
			}
			caseRegion.Instructions = append(caseRegion.Instructions, caseInst)
			inst.Regions = append(inst.Regions, caseRegion)
		}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.SpawnKind:
		inst := l.newInstruction(OpSpawn, node)
		call := first(node.Children)
		if call != nil && call.Kind == hir.CallKind {
			for _, child := range call.Children {
				inst.Args = appendArg(inst.Args, l.lowerExpression(region, child))
			}
		}
		result := l.value(inst)
		l.emit(region, inst)
		return result

	case hir.AwaitKind:
		inst := l.newInstruction(OpAwait, node)
		inst.Args = []ValueID{l.lowerExpression(region, first(node.Children))}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.TryKind:
		inst := l.newInstruction(OpTry, node)
		inst.Args = []ValueID{l.lowerExpression(region, first(node.Children))}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.HandlerKind:
		inst := l.newInstruction(OpHandler, node)
		inst.Args = []ValueID{l.lowerExpression(region, childAt(node.Children, 0))}
		inst.Regions = []*Region{l.lowerBlock("handler", childAt(node.Children, 1))}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	case hir.ForKind:
		inst := l.newInstruction(OpFor, node)
		inst.Regions = []*Region{l.lowerExpressionRegion("init", childAt(node.Children, 0)), l.lowerExpressionRegion("condition", childAt(node.Children, 1)), l.lowerExpressionRegion("update", childAt(node.Children, 2)), l.lowerBlock("body", childAt(node.Children, 3))}
		l.emit(region, inst)
		return 0
	case hir.ForEachKind:
		inst := l.newInstruction(OpForEach, node)
		inst.Meta["source_kind"] = node.Literal
		if len(node.Children) > 0 {
			inst.Args = []ValueID{l.lowerExpression(region, node.Children[0])}
		}
		bodyIndex := len(node.Children) - 1
		if len(node.Children) == 3 {
			inst.Regions = append(inst.Regions, l.lowerExpressionRegion("where", node.Children[1]))
		}
		if bodyIndex >= 1 {
			inst.Regions = append(inst.Regions, l.lowerBlock("body", node.Children[bodyIndex]))
		}
		l.emit(region, inst)
		return 0
	case hir.ForRangeKind:
		inst := l.newInstruction(OpForRange, node)
		inst.Args = []ValueID{l.lowerExpression(region, childAt(node.Children, 0)), l.lowerExpression(region, childAt(node.Children, 1))}
		bodyIndex := len(node.Children) - 1
		if len(node.Children) == 4 {
			inst.Regions = append(inst.Regions, l.lowerExpressionRegion("where", node.Children[2]))
		}
		inst.Regions = append(inst.Regions, l.lowerBlock("body", node.Children[bodyIndex]))
		l.emit(region, inst)
		return 0
	case hir.ForeverKind:
		inst := l.newInstruction(OpForever, node)
		inst.Regions = []*Region{l.lowerBlock("body", first(node.Children))}
		l.emit(region, inst)
		return 0
	case hir.BreakKind:
		l.emit(region, l.newInstruction(OpBreak, node))
		return 0
	case hir.ContinueKind:
		l.emit(region, l.newInstruction(OpContinue, node))
		return 0
	case hir.BlockKind:
		block := l.lowerBlock("block", node)
		inst := l.newInstruction(OpUnknown, node)
		inst.Regions = []*Region{block}
		result := l.value(inst)
		l.emit(region, inst)
		return result
	default:
		inst := l.newInstruction(OpUnknown, node)
		result := l.value(inst)
		l.emit(region, inst)
		return result
	}
}

func (l *lowerer) lowerFunction(node *hir.Node, method bool) *Function {
	name := node.Name
	if name == "" {
		name = fmt.Sprintf("lambda.%d", node.ID)
	}
	params := []string{}
	if raw := node.Meta["params"]; raw != "" {
		params = strings.Split(raw, ",")
	}
	body := l.lowerBlock("body", first(node.Children))
	fn := &Function{Name: name, Owner: node.Meta["owner"], Parameters: params, ReturnType: types.Clone(node.Type), Body: body, Async: node.Flags["async"], Method: method}
	if node.Type != nil && node.Type.Kind == types.Func {
		fn.ReturnType = types.Clone(node.Type.Return)
	}
	l.module.Functions = append(l.module.Functions, fn)
	return fn
}

func (l *lowerer) lowerBlock(name string, node *hir.Node) *Region {
	region := &Region{Name: name}
	if node == nil {
		return region
	}
	for _, child := range node.Children {
		before := len(region.Instructions)
		l.lowerStatement(region, child)
		if len(region.Instructions) > before {
			last := region.Instructions[len(region.Instructions)-1]
			if last.Op == OpDrop && len(last.Args) == 1 {
				region.Result = last.Args[0]
			}
			if last.Op == OpReturn && len(last.Args) == 1 {
				region.Result = last.Args[0]
			}
		}
	}
	return region
}

func (l *lowerer) lowerExpressionRegion(name string, node *hir.Node) *Region {
	region := &Region{Name: name}
	region.Result = l.lowerExpression(region, node)
	return region
}

func first[T any](items []T) T {
	var zero T
	if len(items) == 0 {
		return zero
	}
	return items[0]
}
func childAt[T any](items []T, index int) T {
	var zero T
	if index < 0 || index >= len(items) {
		return zero
	}
	return items[index]
}
func appendArg(values []ValueID, value ValueID) []ValueID {
	if value != 0 {
		return append(values, value)
	}
	return values
}

func (m *Module) Dump() string {
	if m == nil {
		return ""
	}
	var out strings.Builder
	out.WriteString("module " + strconv.Quote(m.Filename))
	if m.Optimized {
		out.WriteString(" optimized")
	}
	out.WriteByte('\n')
	for _, decl := range m.Declarations {
		dumpInstruction(&out, decl, 1)
	}
	for _, fn := range m.Functions {
		functionName := fn.Name
		if fn.Owner != "" {
			functionName = fn.Owner + "." + fn.Name
		}
		out.WriteString("  function " + functionName + "(" + strings.Join(fn.Parameters, ", ") + ")")
		if fn.ReturnType != nil {
			out.WriteString(" -> " + fn.ReturnType.String())
		}
		if fn.Async {
			out.WriteString(" async")
		}
		if fn.Method {
			out.WriteString(" method")
		}
		out.WriteByte('\n')
		dumpRegion(&out, fn.Body, 2)
	}
	dumpRegion(&out, m.Entry, 1)
	return out.String()
}

func dumpRegion(out *strings.Builder, region *Region, depth int) {
	if region == nil {
		return
	}
	out.WriteString(strings.Repeat("  ", depth) + "region " + region.Name)
	if region.Result != 0 {
		out.WriteString(fmt.Sprintf(" result=%%%d", region.Result))
	}
	out.WriteByte('\n')
	for _, inst := range region.Instructions {
		dumpInstruction(out, inst, depth+1)
	}
}

func dumpInstruction(out *strings.Builder, inst *Instruction, depth int) {
	if inst == nil {
		return
	}
	out.WriteString(strings.Repeat("  ", depth))
	if inst.Result != 0 {
		out.WriteString(fmt.Sprintf("%%%d = ", inst.Result))
	}
	out.WriteString(string(inst.Op))
	if inst.Name != "" {
		out.WriteString(" " + strconv.Quote(inst.Name))
	}
	if inst.Operator != "" {
		out.WriteString(" op=" + strconv.Quote(inst.Operator))
	}
	if inst.Literal != "" {
		out.WriteString(" value=" + strconv.Quote(inst.Literal))
	}
	if len(inst.Args) > 0 {
		parts := make([]string, len(inst.Args))
		for i, arg := range inst.Args {
			parts[i] = fmt.Sprintf("%%%d", arg)
		}
		out.WriteString(" (" + strings.Join(parts, ", ") + ")")
	}
	if inst.Type != nil {
		out.WriteString(" : " + inst.Type.String())
	}
	if len(inst.Meta) > 0 {
		keys := make([]string, 0, len(inst.Meta))
		for key := range inst.Meta {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			out.WriteString(" " + key + "=" + strconv.Quote(inst.Meta[key]))
		}
	}
	out.WriteByte('\n')
	for _, region := range inst.Regions {
		dumpRegion(out, region, depth+1)
	}
}
