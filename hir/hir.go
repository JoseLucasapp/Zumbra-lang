// Package hir defines Zumbra's typed high-level intermediate representation.
// HIR preserves source structure while removing parser-specific details and
// attaching the type checker's result to every expression.
package hir

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"zumbra/ast"
	"zumbra/token"
	"zumbra/types"
)

type Kind string

const (
	ModuleKind      Kind = "module"
	BlockKind       Kind = "block"
	VarKind         Kind = "var"
	ConstKind       Kind = "const"
	AssignKind      Kind = "assign"
	SetIndexKind    Kind = "set_index"
	SetFieldKind    Kind = "set_field"
	ExprStmtKind    Kind = "expression_statement"
	ReturnKind      Kind = "return"
	WhileKind       Kind = "while"
	ImportKind      Kind = "import"
	TypeAliasKind   Kind = "type_alias"
	StructKind      Kind = "struct"
	StructFieldKind Kind = "struct_field"
	MethodKind      Kind = "method"
	EnumKind        Kind = "enum"
	FunctionKind    Kind = "function"
	IfKind          Kind = "if"
	MatchKind       Kind = "match"
	CaseKind        Kind = "case"
	CallKind        Kind = "call"
	IdentifierKind  Kind = "identifier"
	IntegerKind     Kind = "integer"
	FloatKind       Kind = "float"
	StringKind      Kind = "string"
	BooleanKind     Kind = "boolean"
	ArrayKind       Kind = "array"
	DictKind        Kind = "dict"
	PairKind        Kind = "pair"
	PrefixKind      Kind = "prefix"
	BinaryKind      Kind = "binary"
	IndexKind       Kind = "index"
	FieldKind       Kind = "field"
	AwaitKind       Kind = "await"
	TryKind         Kind = "try"
	HandlerKind     Kind = "error_handler"
	ForKind         Kind = "for"
	ForEachKind     Kind = "for_each"
	ForRangeKind    Kind = "for_range"
	ForeverKind     Kind = "forever"
	BreakKind       Kind = "break"
	ContinueKind    Kind = "continue"
	UnknownKind     Kind = "unknown"
)

type Node struct {
	ID       int
	Kind     Kind
	Type     *types.Type
	Position token.Position
	Name     string
	Operator string
	Literal  string
	Flags    map[string]bool
	Meta     map[string]string
	Children []*Node
}

type Module struct {
	Filename string
	Program  *ast.Program
	Root     *Node
	Types    *types.Analysis
}

type lowerer struct {
	nextID   int
	analysis *types.Analysis
}

func Lower(filename string, program *ast.Program, analysis *types.Analysis) (*Module, error) {
	if program == nil {
		return nil, fmt.Errorf("cannot lower nil program")
	}
	l := &lowerer{analysis: analysis}
	root := l.node(ModuleKind, program, "", "", "")
	for _, stmt := range program.Statements {
		root.Children = append(root.Children, l.lowerStatement(stmt))
	}
	return &Module{Filename: filename, Program: program, Root: root, Types: analysis}, nil
}

func (l *lowerer) node(kind Kind, source ast.Node, name, operator, literal string) *Node {
	l.nextID++
	return &Node{
		ID:       l.nextID,
		Kind:     kind,
		Type:     l.typeOf(source),
		Position: positionOf(source),
		Name:     name,
		Operator: operator,
		Literal:  literal,
		Flags:    map[string]bool{},
		Meta:     map[string]string{},
	}
}

func (l *lowerer) typeOf(source ast.Node) *types.Type {
	if l.analysis == nil || source == nil {
		return types.Simple(types.Unknown)
	}
	return l.analysis.TypeOf(source)
}

func (l *lowerer) lowerBlock(block *ast.BlockStatement) *Node {
	n := l.node(BlockKind, block, "", "", "")
	if block == nil {
		return n
	}
	for _, stmt := range block.Statements {
		n.Children = append(n.Children, l.lowerStatement(stmt))
	}
	return n
}

func (l *lowerer) lowerStatement(stmt ast.Statement) *Node {
	switch s := stmt.(type) {
	case *ast.VarStatement:
		n := l.node(VarKind, s, s.Name.Value, "", "")
		n.Flags["mutable"] = true
		n.Children = append(n.Children, l.lowerExpression(s.Value))
		n.Type = n.Children[0].Type
		return n
	case *ast.ConstStatement:
		n := l.node(ConstKind, s, s.Name.Value, "", "")
		n.Children = append(n.Children, l.lowerExpression(s.Value))
		n.Type = n.Children[0].Type
		return n
	case *ast.AssignStatement:
		n := l.node(AssignKind, s, s.Name.Value, "", "")
		n.Children = append(n.Children, l.lowerExpression(s.Value))
		n.Type = n.Children[0].Type
		return n
	case *ast.IndexAssignStatement:
		n := l.node(SetIndexKind, s, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(s.Target.Left), l.lowerExpression(s.Target.Index), l.lowerExpression(s.Value))
		n.Type = n.Children[2].Type
		return n
	case *ast.AttributeAssignStatement:
		n := l.node(SetFieldKind, s, s.Target.Property.Value, "", "")
		n.Children = append(n.Children, l.lowerExpression(s.Target.Object), l.lowerExpression(s.Value))
		n.Type = n.Children[1].Type
		return n
	case *ast.ExpressionStatement:
		n := l.node(ExprStmtKind, s, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(s.Expression))
		n.Type = n.Children[0].Type
		return n
	case *ast.ReturnStatement:
		n := l.node(ReturnKind, s, "", "", "")
		if s.ReturnValue != nil {
			n.Children = append(n.Children, l.lowerExpression(s.ReturnValue))
			n.Type = n.Children[0].Type
		}
		return n
	case *ast.WhileStatement:
		n := l.node(WhileKind, s, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(s.Condition), l.lowerBlock(s.Body))
		return n
	case *ast.ImportStatement:
		path := ""
		if s.Path != nil {
			path = s.Path.Value
		}
		return l.node(ImportKind, s, "", "", path)
	case *ast.TypeAliasStatement:
		target := ""
		if s.Target != nil {
			target = s.Target.Value
		}
		return l.node(TypeAliasKind, s, s.Name.Value, "", target)
	case *ast.StructStatement:
		n := l.node(StructKind, s, s.Name.Value, "", "")
		if l.analysis != nil {
			if t, ok := l.analysis.Named(s.Name.Value); ok {
				n.Type = t
			}
		}
		for _, field := range s.Fields {
			child := l.node(StructFieldKind, field.Name, field.Name.Value, "", field.TypeName)
			child.Type = typeField(n.Type, field.Name.Value)
			n.Children = append(n.Children, child)
		}
		for _, method := range s.Methods {
			child := l.lowerExpression(method.Function)
			child.Kind = MethodKind
			child.Name = method.Name.Value
			child.Meta["owner"] = s.Name.Value
			n.Children = append(n.Children, child)
		}
		return n
	case *ast.EnumStatement:
		n := l.node(EnumKind, s, s.Name.Value, "", "")
		if l.analysis != nil {
			if t, ok := l.analysis.Named(s.Name.Value); ok {
				n.Type = t
			}
		}
		for i, member := range s.Members {
			child := l.node(IdentifierKind, member, member.Value, "", strconv.Itoa(i))
			child.Type = n.Type
			n.Children = append(n.Children, child)
		}
		return n
	default:
		return l.node(UnknownKind, stmt, "", "", fmt.Sprintf("%T", stmt))
	}
}

func (l *lowerer) lowerExpression(expr ast.Expression) *Node {
	if expr == nil {
		return l.node(UnknownKind, nil, "", "", "nil")
	}
	switch e := expr.(type) {
	case *ast.Identifier:
		return l.node(IdentifierKind, e, e.Value, "", "")
	case *ast.IntegerLiteral:
		literal := strconv.FormatInt(e.Value, 10)
		if e.FixedType != "" {
			literal = strconv.FormatUint(e.RawValue, 10) + e.FixedType
		}
		return l.node(IntegerKind, e, "", "", literal)
	case *ast.FloatLiteral:
		return l.node(FloatKind, e, "", "", strconv.FormatFloat(e.Value, 'g', -1, 64))
	case *ast.StringLiteral:
		return l.node(StringKind, e, "", "", e.Value)
	case *ast.Boolean:
		return l.node(BooleanKind, e, "", "", strconv.FormatBool(e.Value))
	case *ast.ArrayLiteral:
		n := l.node(ArrayKind, e, "", "", "")
		for _, item := range e.Elements {
			n.Children = append(n.Children, l.lowerExpression(item))
		}
		return n
	case *ast.DictLiteral:
		n := l.node(DictKind, e, "", "", "")
		keys := make([]ast.Expression, 0, len(e.Pairs))
		for key := range e.Pairs {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
		for _, key := range keys {
			pair := l.node(PairKind, key, "", "", "")
			pair.Children = append(pair.Children, l.lowerExpression(key), l.lowerExpression(e.Pairs[key]))
			n.Children = append(n.Children, pair)
		}
		return n
	case *ast.PrefixExpression:
		n := l.node(PrefixKind, e, "", e.Operator, "")
		n.Children = append(n.Children, l.lowerExpression(e.Right))
		return n
	case *ast.InfixExpression:
		n := l.node(BinaryKind, e, "", e.Operator, "")
		n.Children = append(n.Children, l.lowerExpression(e.Left), l.lowerExpression(e.Right))
		return n
	case *ast.CallExpression:
		n := l.node(CallKind, e, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Function))
		for _, arg := range e.Arguments {
			n.Children = append(n.Children, l.lowerExpression(arg))
		}
		return n
	case *ast.IndexExpression:
		n := l.node(IndexKind, e, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Left), l.lowerExpression(e.Index))
		return n
	case *ast.AttributeAccess:
		n := l.node(FieldKind, e, e.Property.Value, "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Object))
		return n
	case *ast.DotExpression:
		n := l.node(FieldKind, e, e.Right.Value, "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Left))
		return n
	case *ast.FunctionLiteral:
		n := l.node(FunctionKind, e, e.Name, "", "")
		n.Flags["async"] = e.Async
		params := make([]string, 0, len(e.Parameters))
		for _, param := range e.Parameters {
			params = append(params, param.Value)
		}
		n.Meta["params"] = strings.Join(params, ",")
		n.Children = append(n.Children, l.lowerBlock(e.Body))
		return n
	case *ast.IfExpression:
		n := l.node(IfKind, e, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Condition), l.lowerBlock(e.Consequence))
		if e.Alternative != nil {
			n.Children = append(n.Children, l.lowerBlock(e.Alternative))
		}
		return n
	case *ast.MatchExpression:
		n := l.node(MatchKind, e, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Value))
		for _, item := range e.Cases {
			caseNode := l.node(CaseKind, item.Pattern, "", "", "")
			caseNode.Children = append(caseNode.Children, l.lowerExpression(item.Pattern), l.lowerBlock(item.Body))
			n.Children = append(n.Children, caseNode)
		}
		if e.Default != nil {
			d := l.node(CaseKind, e.Default, "else", "", "")
			d.Children = append(d.Children, l.lowerBlock(e.Default))
			n.Children = append(n.Children, d)
		}
		return n
	case *ast.AwaitExpression:
		n := l.node(AwaitKind, e, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Value))
		return n
	case *ast.TryExpression:
		n := l.node(TryKind, e, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Value))
		return n
	case *ast.ErrorHandlerExpression:
		name := ""
		if e.ErrorIdent != nil {
			name = e.ErrorIdent.Value
		}
		n := l.node(HandlerKind, e, name, "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Left), l.lowerBlock(e.Handler))
		return n
	case *ast.ForLoop:
		n := l.node(ForKind, e, "", "", "")
		n.Children = append(n.Children, l.lowerExpression(e.Init), l.lowerExpression(e.Cond), l.lowerExpression(e.Update), l.lowerBlock(e.Block))
		return n
	case *ast.ForEachArrayLoop:
		n := l.node(ForEachKind, e, e.Var, "", "array")
		n.Children = append(n.Children, l.lowerExpression(e.Value))
		if e.Cond != nil {
			n.Children = append(n.Children, l.lowerExpression(e.Cond))
		}
		n.Children = append(n.Children, l.lowerBlock(e.Block))
		return n
	case *ast.ForEachMapLoop:
		n := l.node(ForEachKind, e, e.Key+","+e.Value, "", "map")
		n.Children = append(n.Children, l.lowerExpression(e.X))
		if e.Cond != nil {
			n.Children = append(n.Children, l.lowerExpression(e.Cond))
		}
		n.Children = append(n.Children, l.lowerBlock(e.Block))
		return n
	case *ast.ForEachDotRange:
		n := l.node(ForRangeKind, e, e.Var, "", "")
		n.Children = append(n.Children, l.lowerExpression(e.StartIdx), l.lowerExpression(e.EndIdx))
		if e.Cond != nil {
			n.Children = append(n.Children, l.lowerExpression(e.Cond))
		}
		n.Children = append(n.Children, l.lowerBlock(e.Block))
		return n
	case *ast.ForEverLoop:
		n := l.node(ForeverKind, e, "", "", "")
		n.Children = append(n.Children, l.lowerBlock(e.Block))
		return n
	case *ast.BreakExpression:
		return l.node(BreakKind, e, "", "", "")
	case *ast.ContinueExpression:
		return l.node(ContinueKind, e, "", "", "")
	default:
		return l.node(UnknownKind, expr, "", "", fmt.Sprintf("%T", expr))
	}
}

func typeField(value *types.Type, name string) *types.Type {
	if value != nil && value.Fields != nil {
		if item, ok := value.Fields[name]; ok {
			return types.Clone(item)
		}
	}
	return types.Simple(types.Unknown)
}

func positionOf(node ast.Node) token.Position {
	switch n := node.(type) {
	case *ast.VarStatement:
		return n.Token.Pos
	case *ast.ConstStatement:
		return n.Token.Pos
	case *ast.AssignStatement:
		return n.Token.Pos
	case *ast.IndexAssignStatement:
		return n.Token.Pos
	case *ast.AttributeAssignStatement:
		return n.Token.Pos
	case *ast.ExpressionStatement:
		return n.Token.Pos
	case *ast.ReturnStatement:
		return n.Token.Pos
	case *ast.WhileStatement:
		return n.Token.Pos
	case *ast.ImportStatement:
		return n.Token.Pos
	case *ast.TypeAliasStatement:
		return n.Token.Pos
	case *ast.StructStatement:
		return n.Token.Pos
	case *ast.EnumStatement:
		return n.Token.Pos
	case *ast.BlockStatement:
		return n.Token.Pos
	case *ast.Identifier:
		return n.Token.Pos
	case *ast.IntegerLiteral:
		return n.Token.Pos
	case *ast.FloatLiteral:
		return n.Token.Pos
	case *ast.StringLiteral:
		return n.Token.Pos
	case *ast.Boolean:
		return n.Token.Pos
	case *ast.ArrayLiteral:
		return n.Token.Pos
	case *ast.DictLiteral:
		return n.Token.Pos
	case *ast.PrefixExpression:
		return n.Token.Pos
	case *ast.InfixExpression:
		return n.Token.Pos
	case *ast.CallExpression:
		return n.Token.Pos
	case *ast.IndexExpression:
		return n.Token.Pos
	case *ast.DotExpression:
		return n.Token.Pos
	case *ast.FunctionLiteral:
		return n.Token.Pos
	case *ast.IfExpression:
		return n.Token.Pos
	case *ast.MatchExpression:
		return n.Token.Pos
	case *ast.AwaitExpression:
		return n.Token.Pos
	case *ast.TryExpression:
		return n.Token.Pos
	case *ast.ErrorHandlerExpression:
		return n.Token.Pos
	case *ast.ForLoop:
		return n.Token.Pos
	case *ast.ForEachArrayLoop:
		return n.Token.Pos
	case *ast.ForEachMapLoop:
		return n.Token.Pos
	case *ast.ForEachDotRange:
		return n.Token.Pos
	case *ast.ForEverLoop:
		return n.Token.Pos
	case *ast.BreakExpression:
		return n.Token.Pos
	case *ast.ContinueExpression:
		return n.Token.Pos
	case *ast.AttributeAccess:
		return positionOf(n.Object)
	}
	return token.Position{}
}

func (m *Module) Dump() string {
	if m == nil || m.Root == nil {
		return ""
	}
	var out strings.Builder
	dumpNode(&out, m.Root, 0)
	return out.String()
}

func dumpNode(out *strings.Builder, node *Node, depth int) {
	if node == nil {
		return
	}
	out.WriteString(strings.Repeat("  ", depth))
	out.WriteString(fmt.Sprintf("%%%d %s", node.ID, node.Kind))
	if node.Name != "" {
		out.WriteString(" name=" + strconv.Quote(node.Name))
	}
	if node.Operator != "" {
		out.WriteString(" op=" + strconv.Quote(node.Operator))
	}
	if node.Literal != "" {
		out.WriteString(" value=" + strconv.Quote(node.Literal))
	}
	if node.Type != nil {
		out.WriteString(" : " + node.Type.String())
	}
	if node.Position.IsValid() {
		out.WriteString(fmt.Sprintf(" @%d:%d", node.Position.Line, node.Position.Col))
	}
	out.WriteByte('\n')
	for _, child := range node.Children {
		dumpNode(out, child, depth+1)
	}
}
