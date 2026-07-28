package builtins

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync/atomic"

	"zumbra/object"
)

var uiNodeSequence atomic.Int64

func uiNodeArg(value object.Object, name string) (*object.UINode, *object.Error) {
	node, ok := value.(*object.UINode)
	if !ok {
		return nil, NewError("%s expects UINode, got %s", name, value.Type())
	}
	return node, nil
}
func uiContextArg(value object.Object, name string) (*object.UIContext, *object.Error) {
	ctx, ok := value.(*object.UIContext)
	if !ok {
		return nil, NewError("%s expects UIContext, got %s", name, value.Type())
	}
	return ctx, nil
}
func uiStateArg(value object.Object, name string) (*object.UIState, *object.Error) {
	state, ok := value.(*object.UIState)
	if !ok {
		return nil, NewError("%s expects UIState, got %s", name, value.Type())
	}
	return state, nil
}
func uiThemeArg(value object.Object, name string) (*object.UITheme, *object.Error) {
	theme, ok := value.(*object.UITheme)
	if !ok {
		return nil, NewError("%s expects UITheme, got %s", name, value.Type())
	}
	return theme, nil
}
func uiChildren(value object.Object, name string) ([]*object.UINode, *object.Error) {
	array, ok := value.(*object.Array)
	if !ok {
		return nil, NewError("%s expects array of UINode, got %s", name, value.Type())
	}
	result := make([]*object.UINode, 0, len(array.Elements))
	for index, child := range array.Elements {
		node, ok := child.(*object.UINode)
		if !ok {
			return nil, NewError("%s child %d expects UINode, got %s", name, index, child.Type())
		}
		result = append(result, node)
	}
	return result, nil
}
func uiProps(value object.Object, name string) (map[string]object.Object, *object.Error) {
	if _, ok := value.(*object.Null); ok {
		return map[string]object.Object{}, nil
	}
	return objectDictMap(value, name)
}
func uiNewNode(kind string, props map[string]object.Object, children []*object.UINode) *object.UINode {
	if nested, ok := props["options"].(*object.Dict); ok {
		if values, err := objectDictMap(nested, "UI options"); err == nil {
			for key, value := range values {
				if _, exists := props[key]; !exists {
					props[key] = value
				}
			}
		}
		delete(props, "options")
	}
	id := optionString(props, "id", "")
	if id == "" {
		id = fmt.Sprintf("ui-%d", uiNodeSequence.Add(1))
	}
	visible := optionBool(props, "visible", true)
	enabled := !optionBool(props, "disabled", false)
	node := &object.UINode{ID: id, Kind: kind, Props: props, Bindings: map[string]*object.UIState{}, Children: children, Visible: visible, Enabled: enabled}
	for _, child := range children {
		child.Parent = node
	}
	return node
}
func uiNodeBuiltin(kind string) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("ui%s expects props and children", strings.Title(kind))
		}
		props, e := uiProps(args[0], "ui"+kind+" props")
		if e != nil {
			return e
		}
		children, e := uiChildren(args[1], "ui"+kind+" children")
		if e != nil {
			return e
		}
		return uiNewNode(kind, props, children)
	}}
}
func UINodeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("uiNode expects kind, props and children")
		}
		kind, e := desktopString(args[0], "uiNode kind")
		if e != nil {
			return e
		}
		props, e := uiProps(args[1], "uiNode props")
		if e != nil {
			return e
		}
		children, e := uiChildren(args[2], "uiNode children")
		if e != nil {
			return e
		}
		kind = strings.ToLower(strings.TrimSpace(kind))
		if kind == "" {
			return NewError("uiNode kind cannot be empty")
		}
		return uiNewNode(kind, props, children)
	}}
}

func UIThemeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("uiTheme expects a name or dictionary")
		}
		values := defaultUITheme("light")
		name := "custom"
		switch value := args[0].(type) {
		case *object.String:
			name = strings.ToLower(strings.TrimSpace(value.Value))
			values = defaultUITheme(name)
		case *object.Dict:
			values = defaultUITheme("light")
			custom, _ := objectDictMap(value, "uiTheme")
			for key, item := range custom {
				values[key] = item
			}
		default:
			return NewError("uiTheme expects string or dictionary, got %s", args[0].Type())
		}
		return &object.UITheme{Name: name, Values: values}
	}}
}
func defaultUITheme(name string) map[string]object.Object {
	dark := name == "dark"
	color := func(light, darkValue string) object.Object {
		if dark {
			return NewString(darkValue)
		}
		return NewString(light)
	}
	return map[string]object.Object{
		"background": color("#f5f7fb", "#141822"), "surface": color("#ffffff", "#202633"),
		"surfaceAlt": color("#eef2f7", "#2a3242"), "text": color("#172033", "#f2f5fa"),
		"muted": color("#697386", "#a9b4c7"), "primary": NewString("#3867e8"),
		"primaryText": NewString("#ffffff"), "border": color("#cfd6e2", "#465066"),
		"danger": NewString("#c73737"), "focus": NewString("#6e95ff"), "radius": NewInteger(6),
		"fontFamily": NewString("sans"), "fontPath": NewString(""), "fontWeight": NewString("normal"),
		"fontStyle": NewString("normal"), "fontSize": NewInteger(14), "lineHeight": NewFloat(1.25),
		"spacing": NewInteger(8), "controlHeight": NewInteger(36),
	}
}

func UIStateBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("uiState expects value")
		}
		return &object.UIState{Value: args[0]}
	}}
}
func UIStateGetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("uiStateGet expects state")
		}
		s, e := uiStateArg(args[0], "uiStateGet")
		if e != nil {
			return e
		}
		s.Mu.RLock()
		defer s.Mu.RUnlock()
		return s.Value
	}}
}
func UIStateSetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiStateSet expects state and value")
		}
		s, e := uiStateArg(args[0], "uiStateSet")
		if e != nil {
			return e
		}
		s.Mu.Lock()
		s.Value = args[1]
		bindings := append([]object.UIStateBinding{}, s.Bindings...)
		subscribers := append([]object.Object{}, s.Subscribers...)
		s.Mu.Unlock()
		contexts := map[*object.UIContext]bool{}
		for _, binding := range bindings {
			if binding.Node == nil {
				continue
			}
			binding.Node.Mu.Lock()
			binding.Node.Props[binding.Property] = args[1]
			ctx := binding.Node.Context
			binding.Node.Mu.Unlock()
			if ctx != nil {
				contexts[ctx] = true
			}
		}
		for ctx := range contexts {
			ctx.Mu.Lock()
			ctx.Dirty = true
			ctx.Mu.Unlock()
			_ = renderUIContext(ctx)
		}
		for _, subscriber := range subscribers {
			if _, err := invokeDesktopHandler(subscriber, args[1]); err != nil {
				return NewError("uiStateSet subscriber: %s", err)
			}
		}
		return args[1]
	}}
}
func UIStateSubscribeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiStateSubscribe expects state and handler")
		}
		s, e := uiStateArg(args[0], "uiStateSubscribe")
		if e != nil {
			return e
		}
		s.Mu.Lock()
		s.Subscribers = append(s.Subscribers, args[1])
		s.Mu.Unlock()
		return s
	}}
}
func UIBindBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("uiBind expects node, property and state")
		}
		n, e := uiNodeArg(args[0], "uiBind")
		if e != nil {
			return e
		}
		property, e := desktopString(args[1], "uiBind property")
		if e != nil {
			return e
		}
		s, e := uiStateArg(args[2], "uiBind")
		if e != nil {
			return e
		}
		s.Mu.Lock()
		value := s.Value
		s.Bindings = append(s.Bindings, object.UIStateBinding{Node: n, Property: property})
		s.Mu.Unlock()
		n.Mu.Lock()
		n.Props[property] = value
		if n.Bindings == nil {
			n.Bindings = map[string]*object.UIState{}
		}
		n.Bindings[property] = s
		n.Mu.Unlock()
		return n
	}}
}

func UIMountBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("uiMount expects app, window, root and options")
		}
		app, e := desktopAppArg(args[0], "uiMount")
		if e != nil {
			return e
		}
		window, e := desktopWindowArg(args[1], "uiMount")
		if e != nil {
			return e
		}
		root, e := uiNodeArg(args[2], "uiMount")
		if e != nil {
			return e
		}
		options, e := uiProps(args[3], "uiMount options")
		if e != nil {
			return e
		}
		theme := &object.UITheme{Name: "light", Values: defaultUITheme("light")}
		if themeValue, ok := options["theme"].(*object.UITheme); ok {
			theme = themeValue
		}
		ctx := &object.UIContext{App: app, Window: window, Root: root, Theme: theme, Nodes: map[string]*object.UINode{}, Dirty: true}
		indexUINodes(ctx, root)
		app.Mu.Lock()
		if app.UIContexts == nil {
			app.UIContexts = map[int64]*object.UIContext{}
		}
		app.UIContexts[window.Runtime.ID()] = ctx
		app.Mu.Unlock()
		if err := renderUIContext(ctx); err != nil {
			return NewError("uiMount: %s", err)
		}
		return ctx
	}}
}
func indexUINodes(ctx *object.UIContext, node *object.UINode) {
	if node == nil {
		return
	}
	node.Mu.Lock()
	node.Context = ctx
	node.Mu.Unlock()
	ctx.Nodes[node.ID] = node
	for _, child := range node.Children {
		indexUINodes(ctx, child)
	}
}
func UIUnmountBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("uiUnmount expects context")
		}
		ctx, e := uiContextArg(args[0], "uiUnmount")
		if e != nil {
			return e
		}
		ctx.Mu.Lock()
		if ctx.Closed {
			ctx.Mu.Unlock()
			return NewBoolean(false)
		}
		ctx.Closed = true
		ctx.Mu.Unlock()
		if ctx.App != nil && ctx.Window != nil {
			ctx.App.Mu.Lock()
			delete(ctx.App.UIContexts, ctx.Window.Runtime.ID())
			ctx.App.Mu.Unlock()
		}
		return NewBoolean(true)
	}}
}
func UIRenderBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("uiRender expects context")
		}
		ctx, e := uiContextArg(args[0], "uiRender")
		if e != nil {
			return e
		}
		if err := renderUIContext(ctx); err != nil {
			return NewError("uiRender: %s", err)
		}
		return NewBoolean(true)
	}}
}
func UISnapshotBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("uiSnapshot expects context")
		}
		ctx, e := uiContextArg(args[0], "uiSnapshot")
		if e != nil {
			return e
		}
		ctx.Mu.RLock()
		frame := ctx.LastFrame
		ctx.Mu.RUnlock()
		if frame == nil {
			if err := renderUIContext(ctx); err != nil {
				return NewError("uiSnapshot: %s", err)
			}
			ctx.Mu.RLock()
			frame = ctx.LastFrame
			ctx.Mu.RUnlock()
		}
		return uiFrameObject(frame)
	}}
}
func UISetThemeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiSetTheme expects context and theme")
		}
		ctx, e := uiContextArg(args[0], "uiSetTheme")
		if e != nil {
			return e
		}
		theme, e := uiThemeArg(args[1], "uiSetTheme")
		if e != nil {
			return e
		}
		ctx.Mu.Lock()
		ctx.Theme = theme
		ctx.Dirty = true
		ctx.Mu.Unlock()
		if err := renderUIContext(ctx); err != nil {
			return NewError("uiSetTheme: %s", err)
		}
		return ctx
	}}
}

func UISetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("uiSet expects node, property and value")
		}
		n, e := uiNodeArg(args[0], "uiSet")
		if e != nil {
			return e
		}
		property, e := desktopString(args[1], "uiSet property")
		if e != nil {
			return e
		}
		n.Mu.Lock()
		n.Props[property] = args[2]
		if property == "visible" {
			if b, ok := args[2].(*object.Boolean); ok {
				n.Visible = b.Value
			}
		}
		if property == "disabled" {
			if b, ok := args[2].(*object.Boolean); ok {
				n.Enabled = !b.Value
			}
		}
		ctx := n.Context
		n.Mu.Unlock()
		markUIContextDirty(ctx)
		return n
	}}
}
func UIGetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiGet expects node and property")
		}
		n, e := uiNodeArg(args[0], "uiGet")
		if e != nil {
			return e
		}
		property, e := desktopString(args[1], "uiGet property")
		if e != nil {
			return e
		}
		n.Mu.RLock()
		defer n.Mu.RUnlock()
		if property == "id" {
			return NewString(n.ID)
		}
		if property == "kind" {
			return NewString(n.Kind)
		}
		if v, ok := n.Props[property]; ok {
			return v
		}
		return &object.Null{}
	}}
}
func UIAddBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiAdd expects parent and child")
		}
		p, e := uiNodeArg(args[0], "uiAdd")
		if e != nil {
			return e
		}
		c, e := uiNodeArg(args[1], "uiAdd")
		if e != nil {
			return e
		}
		p.Mu.Lock()
		p.Children = append(p.Children, c)
		c.Parent = p
		ctx := p.Context
		p.Mu.Unlock()
		if ctx != nil {
			ctx.Mu.Lock()
			indexUINodes(ctx, c)
			ctx.Dirty = true
			ctx.Mu.Unlock()
		}
		return p
	}}
}
func UIRemoveBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiRemove expects parent and child id")
		}
		p, e := uiNodeArg(args[0], "uiRemove")
		if e != nil {
			return e
		}
		id, e := desktopString(args[1], "uiRemove id")
		if e != nil {
			return e
		}
		removed := false
		p.Mu.Lock()
		next := p.Children[:0]
		for _, c := range p.Children {
			if c.ID == id {
				removed = true
				c.Parent = nil
			} else {
				next = append(next, c)
			}
		}
		p.Children = next
		ctx := p.Context
		p.Mu.Unlock()
		if ctx != nil {
			ctx.Mu.Lock()
			delete(ctx.Nodes, id)
			ctx.Dirty = true
			ctx.Mu.Unlock()
		}
		return NewBoolean(removed)
	}}
}
func UIFindBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiFind expects context and id")
		}
		ctx, e := uiContextArg(args[0], "uiFind")
		if e != nil {
			return e
		}
		id, e := desktopString(args[1], "uiFind id")
		if e != nil {
			return e
		}
		ctx.Mu.RLock()
		n := ctx.Nodes[id]
		ctx.Mu.RUnlock()
		if n == nil {
			return &object.Null{}
		}
		return n
	}}
}
func UIFocusBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiFocus expects context and node")
		}
		ctx, e := uiContextArg(args[0], "uiFocus")
		if e != nil {
			return e
		}
		n, e := uiNodeArg(args[1], "uiFocus")
		if e != nil {
			return e
		}
		if !uiNodeFocusable(n) {
			return NewBoolean(false)
		}
		ctx.Mu.Lock()
		ctx.FocusID = n.ID
		ctx.Dirty = true
		ctx.Mu.Unlock()
		_ = renderUIContext(ctx)
		return NewBoolean(true)
	}}
}
func UIFocusNextBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiFocusNext expects context and reverse")
		}
		ctx, e := uiContextArg(args[0], "uiFocusNext")
		if e != nil {
			return e
		}
		reverse, e := desktopBool(args[1], "uiFocusNext reverse")
		if e != nil {
			return e
		}
		n := focusNextUI(ctx, reverse)
		if n == nil {
			return &object.Null{}
		}
		_ = renderUIContext(ctx)
		return n
	}}
}
func UIAccessibilityBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("uiAccessibility expects context")
		}
		ctx, e := uiContextArg(args[0], "uiAccessibility")
		if e != nil {
			return e
		}
		ctx.Mu.RLock()
		root := ctx.Root
		ctx.Mu.RUnlock()
		items := []object.Object{}
		collectAccessibility(root, &items)
		return &object.Array{Elements: items}
	}}
}
func UICanvasCommandBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("uiCanvasCommand expects canvas, kind and values")
		}
		n, e := uiNodeArg(args[0], "uiCanvasCommand")
		if e != nil {
			return e
		}
		if n.Kind != "canvas" {
			return NewError("uiCanvasCommand expects canvas node")
		}
		kind, e := desktopString(args[1], "uiCanvasCommand kind")
		if e != nil {
			return e
		}
		values, e := uiProps(args[2], "uiCanvasCommand values")
		if e != nil {
			return e
		}
		n.Mu.Lock()
		commands, ok := n.Props["commands"].(*object.Array)
		if !ok {
			commands = &object.Array{}
			n.Props["commands"] = commands
		}
		commands.Elements = append(commands.Elements, objectMapDict(map[string]object.Object{"kind": NewString(kind), "values": objectMapDict(values)}))
		ctx := n.Context
		n.Mu.Unlock()
		markUIContextDirty(ctx)
		return n
	}}
}
func UIDispatchBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("uiDispatch expects context and event")
		}
		ctx, e := uiContextArg(args[0], "uiDispatch")
		if e != nil {
			return e
		}
		eventMap, e := uiProps(args[1], "uiDispatch event")
		if e != nil {
			return e
		}
		event := &object.DesktopEvent{Type: optionString(eventMap, "type", "custom"), WindowID: ctx.Window.Runtime.ID(), Data: eventMap}
		if err := dispatchUIEvent(ctx, event); err != nil {
			return err
		}
		_ = renderUIContext(ctx)
		return NewBoolean(true)
	}}
}

func markUIContextDirty(ctx *object.UIContext) {
	if ctx == nil {
		return
	}
	ctx.Mu.Lock()
	ctx.Dirty = true
	ctx.Mu.Unlock()
}
func uiNodeFocusable(n *object.UINode) bool {
	if n == nil {
		return false
	}
	n.Mu.RLock()
	defer n.Mu.RUnlock()
	if !n.Visible || !n.Enabled {
		return false
	}
	if optionInt(n.Props, "tabIndex", 0) < 0 {
		return false
	}
	switch n.Kind {
	case "button", "input", "textarea", "select", "checkbox", "radio", "tabs", "menu":
		return true
	}
	return optionBool(n.Props, "focusable", false)
}
func focusNextUI(ctx *object.UIContext, reverse bool) *object.UINode {
	ctx.Mu.RLock()
	root := ctx.Root
	current := ctx.FocusID
	ctx.Mu.RUnlock()
	nodes := []*object.UINode{}
	walkUI(root, func(n *object.UINode) {
		if uiNodeFocusable(n) {
			nodes = append(nodes, n)
		}
	})
	sort.SliceStable(nodes, func(i, j int) bool {
		return optionInt(nodes[i].Props, "tabIndex", 0) < optionInt(nodes[j].Props, "tabIndex", 0)
	})
	if len(nodes) == 0 {
		return nil
	}
	index := -1
	for i, n := range nodes {
		if n.ID == current {
			index = i
			break
		}
	}
	if reverse {
		index--
		if index < 0 {
			index = len(nodes) - 1
		}
	} else {
		index++
		if index >= len(nodes) {
			index = 0
		}
	}
	ctx.Mu.Lock()
	ctx.FocusID = nodes[index].ID
	ctx.Dirty = true
	ctx.Mu.Unlock()
	return nodes[index]
}
func walkUI(node *object.UINode, fn func(*object.UINode)) {
	if node == nil {
		return
	}
	fn(node)
	node.Mu.RLock()
	children := append([]*object.UINode{}, node.Children...)
	node.Mu.RUnlock()
	for _, child := range children {
		walkUI(child, fn)
	}
}
func collectAccessibility(node *object.UINode, items *[]object.Object) {
	if node == nil {
		return
	}
	node.Mu.RLock()
	props := cloneUIProps(node.Props)
	visible := node.Visible
	enabled := node.Enabled
	bounds := node.Bounds
	children := append([]*object.UINode{}, node.Children...)
	node.Mu.RUnlock()
	if visible {
		role := optionString(props, "role", defaultUIRole(node.Kind))
		label := optionString(props, "accessibilityLabel", optionString(props, "text", optionString(props, "label", "")))
		*items = append(*items, objectMapDict(map[string]object.Object{"id": NewString(node.ID), "role": NewString(role), "label": NewString(label), "description": NewString(optionString(props, "accessibilityDescription", "")), "enabled": NewBoolean(enabled), "focusable": NewBoolean(uiNodeFocusable(node)), "x": NewFloat(bounds.X), "y": NewFloat(bounds.Y), "width": NewFloat(bounds.Width), "height": NewFloat(bounds.Height)}))
	}
	for _, child := range children {
		collectAccessibility(child, items)
	}
}
func defaultUIRole(kind string) string {
	switch kind {
	case "text":
		return "text"
	case "button":
		return "button"
	case "input":
		return "textbox"
	case "textarea":
		return "textbox"
	case "select":
		return "combobox"
	case "checkbox":
		return "checkbox"
	case "radio":
		return "radio"
	case "table":
		return "table"
	case "list":
		return "list"
	case "tree":
		return "tree"
	case "tabs":
		return "tablist"
	case "menu":
		return "menu"
	case "progress":
		return "progressbar"
	case "image":
		return "img"
	case "modal":
		return "dialog"
	}
	return "group"
}

func UIContextMethod(ctx *object.UIContext, property string) object.Object {
	methods := map[string]*object.Builtin{"render": UIRenderBuiltin(), "snapshot": UISnapshotBuiltin(), "setTheme": UISetThemeBuiltin(), "dispatch": UIDispatchBuiltin(), "find": UIFindBuiltin(), "focus": UIFocusBuiltin(), "focusNext": UIFocusNextBuiltin(), "accessibility": UIAccessibilityBuiltin(), "close": UIUnmountBuiltin()}
	if b := methods[property]; b != nil {
		return bindUIBuiltin(b, ctx)
	}
	return nil
}
func UINodeMethod(node *object.UINode, property string) object.Object {
	methods := map[string]*object.Builtin{"set": UISetBuiltin(), "get": UIGetBuiltin(), "add": UIAddBuiltin(), "remove": UIRemoveBuiltin(), "command": UICanvasCommandBuiltin()}
	if b := methods[property]; b != nil {
		return bindUIBuiltin(b, node)
	}
	return nil
}
func UIStateMethod(state *object.UIState, property string) object.Object {
	methods := map[string]*object.Builtin{"get": UIStateGetBuiltin(), "set": UIStateSetBuiltin(), "subscribe": UIStateSubscribeBuiltin()}
	if b := methods[property]; b != nil {
		return bindUIBuiltin(b, state)
	}
	return nil
}
func bindUIBuiltin(b *object.Builtin, receiver object.Object) object.Object {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		all := make([]object.Object, 0, len(args)+1)
		all = append(all, receiver)
		all = append(all, args...)
		return b.Fn(all...)
	}}
}

func uiFrameObject(frame *object.UIRenderFrame) object.Object {
	if frame == nil {
		return &object.Null{}
	}
	items := make([]object.Object, 0, len(frame.Items))
	for _, item := range frame.Items {
		items = append(items, objectMapDict(map[string]object.Object{"id": NewString(item.ID), "kind": NewString(item.Kind), "x": NewFloat(item.Bounds.X), "y": NewFloat(item.Bounds.Y), "width": NewFloat(item.Bounds.Width), "height": NewFloat(item.Bounds.Height), "focused": NewBoolean(item.Focused), "hovered": NewBoolean(item.Hovered), "props": objectMapDict(item.Props)}))
	}
	return objectMapDict(map[string]object.Object{"width": NewFloat(frame.Width), "height": NewFloat(frame.Height), "background": NewString(frame.Background), "items": &object.Array{Elements: items}})
}

// Dispatch is called by the Z13 event loop before application handlers.
func dispatchUIForDesktopEvent(app *object.DesktopApp, event *object.DesktopEvent) *object.Error {
	if app == nil || event == nil {
		return nil
	}
	app.Mu.RLock()
	ctx := app.UIContexts[event.WindowID]
	app.Mu.RUnlock()
	if ctx == nil {
		return nil
	}
	return dispatchUIEvent(ctx, event)
}
func renderDesktopUIContexts(app *object.DesktopApp) *object.Error {
	if app == nil {
		return nil
	}
	app.Mu.RLock()
	contexts := make([]*object.UIContext, 0, len(app.UIContexts))
	for _, ctx := range app.UIContexts {
		contexts = append(contexts, ctx)
	}
	app.Mu.RUnlock()
	for _, ctx := range contexts {
		ctx.Mu.RLock()
		dirty := ctx.Dirty
		ctx.Mu.RUnlock()
		if dirty {
			if err := renderUIContext(ctx); err != nil {
				return NewError("desktop UI render: %s", err)
			}
		}
	}
	return nil
}

func updateUIBoundState(node *object.UINode, property string, value object.Object) *object.Error {
	if node == nil {
		return nil
	}
	node.Mu.RLock()
	state := node.Bindings[property]
	node.Mu.RUnlock()
	if state == nil {
		return nil
	}
	result := UIStateSetBuiltin().Fn(state, value)
	if err, ok := result.(*object.Error); ok {
		return err
	}
	return nil
}

func dispatchUIEvent(ctx *object.UIContext, event *object.DesktopEvent) *object.Error {
	if ctx == nil || event == nil {
		return nil
	}
	ctx.Mu.RLock()
	if ctx.Closed {
		ctx.Mu.RUnlock()
		return nil
	}
	root := ctx.Root
	focusID := ctx.FocusID
	ctx.Mu.RUnlock()
	x := uiEventNumber(event.Data, "x")
	y := uiEventNumber(event.Data, "y")
	var target *object.UINode
	switch event.Type {
	case "mouse_motion", "mouse_down", "mouse_up", "mouse_button_down", "mouse_button_up":
		target = hitTestUI(root, x, y)
		if event.Type == "mouse_motion" {
			ctx.Mu.Lock()
			if target != nil {
				ctx.HoverID = target.ID
			} else {
				ctx.HoverID = ""
			}
			ctx.Dirty = true
			ctx.Mu.Unlock()
		}
	case "key_down", "text_input":
		ctx.Mu.RLock()
		target = ctx.Nodes[focusID]
		ctx.Mu.RUnlock()
	}
	if event.Type == "key_down" {
		shortcut := ""
		if s, ok := event.Data["shortcut"].(*object.String); ok {
			shortcut = strings.ToLower(s.Value)
		}
		if shortcut == "tab" || shortcut == "shift+tab" {
			focusNextUI(ctx, shortcut == "shift+tab")
			return nil
		}
	}
	if target == nil {
		return nil
	}
	if !target.Enabled {
		return nil
	}
	target.Mu.RLock()
	kind := target.Kind
	props := cloneUIProps(target.Props)
	target.Mu.RUnlock()
	eventObject := desktopEventObject(event)
	invoke := func(name string) *object.Error {
		handler := props[name]
		if handler == nil {
			return nil
		}
		if _, err := invokeDesktopHandler(handler, eventObject); err != nil {
			return NewError("UI %s handler: %s", name, err)
		}
		return nil
	}
	down := event.Type == "mouse_down" || event.Type == "mouse_button_down"
	up := event.Type == "mouse_up" || event.Type == "mouse_button_up"
	if down && uiNodeFocusable(target) {
		ctx.Mu.Lock()
		ctx.FocusID = target.ID
		ctx.Dirty = true
		ctx.Mu.Unlock()
		if e := invoke("onFocus"); e != nil {
			return e
		}
	}
	if event.Type == "mouse_motion" {
		if e := invoke("onHover"); e != nil {
			return e
		}
	}
	activate := up
	if event.Type == "key_down" {
		key := ""
		if v, ok := event.Data["key"].(*object.String); ok {
			key = strings.ToLower(v.Value)
		}
		activate = key == "enter" || key == "space"
	}
	if activate {
		switch kind {
		case "button", "menu":
			if e := invoke("onClick"); e != nil {
				return e
			}
		case "checkbox":
			current := optionBool(props, "checked", false)
			next := NewBoolean(!current)
			target.Mu.Lock()
			target.Props["checked"] = next
			target.Mu.Unlock()
			if e := updateUIBoundState(target, "checked", next); e != nil {
				return e
			}
			if e := invoke("onChange"); e != nil {
				return e
			}
		case "radio":
			checked := NewBoolean(true)
			target.Mu.Lock()
			target.Props["checked"] = checked
			target.Mu.Unlock()
			if e := updateUIBoundState(target, "checked", checked); e != nil {
				return e
			}
			if target.Parent != nil {
				target.Parent.Mu.Lock()
				for _, sibling := range target.Parent.Children {
					if sibling != target && sibling.Kind == "radio" {
						sibling.Mu.Lock()
						sibling.Props["checked"] = NewBoolean(false)
						sibling.Mu.Unlock()
					}
				}
				target.Parent.Mu.Unlock()
			}
			if e := invoke("onChange"); e != nil {
				return e
			}
		case "select", "tabs":
			items := uiArrayStrings(props["options"])
			if kind == "tabs" {
				items = uiArrayStrings(props["tabs"])
			}
			if len(items) > 0 {
				index := int(optionInt(props, "selectedIndex", 0))
				index = (index + 1) % len(items)
				selectedIndex := NewInteger(int64(index))
				selectedValue := NewString(items[index])
				target.Mu.Lock()
				target.Props["selectedIndex"] = selectedIndex
				target.Props["value"] = selectedValue
				target.Mu.Unlock()
				if e := updateUIBoundState(target, "selectedIndex", selectedIndex); e != nil {
					return e
				}
				if e := updateUIBoundState(target, "value", selectedValue); e != nil {
					return e
				}
				if e := invoke("onChange"); e != nil {
					return e
				}
			}
		}
	}
	if event.Type == "key_down" && (kind == "input" || kind == "textarea") {
		key := ""
		if v, ok := event.Data["key"].(*object.String); ok {
			key = strings.ToLower(v.Value)
		}
		if key == "backspace" {
			current := []rune(optionString(props, "value", ""))
			if len(current) > 0 {
				next := NewString(string(current[:len(current)-1]))
				target.Mu.Lock()
				target.Props["value"] = next
				target.Mu.Unlock()
				if e := updateUIBoundState(target, "value", next); e != nil {
					return e
				}
				if e := invoke("onInput"); e != nil {
					return e
				}
				if e := invoke("onChange"); e != nil {
					return e
				}
			}
		}
	}
	if event.Type == "text_input" && (kind == "input" || kind == "textarea") {
		text := ""
		if v, ok := event.Data["text"].(*object.String); ok {
			text = v.Value
		}
		current := optionString(props, "value", "")
		next := NewString(current + text)
		target.Mu.Lock()
		target.Props["value"] = next
		target.Mu.Unlock()
		if e := updateUIBoundState(target, "value", next); e != nil {
			return e
		}
		if e := invoke("onInput"); e != nil {
			return e
		}
		if e := invoke("onChange"); e != nil {
			return e
		}
	}
	ctx.Mu.Lock()
	ctx.Dirty = true
	ctx.Mu.Unlock()
	return nil
}
func uiEventNumber(values map[string]object.Object, key string) float64 {
	if v, ok := values[key].(*object.Float); ok {
		return v.Value
	}
	if v, ok := values[key].(*object.Integer); ok {
		return float64(v.Value)
	}
	return 0
}
func hitTestUI(node *object.UINode, x, y float64) *object.UINode {
	if node == nil {
		return nil
	}
	node.Mu.RLock()
	visible := node.Visible
	bounds := node.Bounds
	children := append([]*object.UINode{}, node.Children...)
	node.Mu.RUnlock()
	if !visible {
		return nil
	}
	for i := len(children) - 1; i >= 0; i-- {
		if hit := hitTestUI(children[i], x, y); hit != nil {
			return hit
		}
	}
	if x >= bounds.X && y >= bounds.Y && x <= bounds.X+bounds.Width && y <= bounds.Y+bounds.Height {
		return node
	}
	return nil
}
func uiArrayStrings(value object.Object) []string {
	a, ok := value.(*object.Array)
	if !ok {
		return nil
	}
	out := []string{}
	for _, item := range a.Elements {
		if s, ok := item.(*object.String); ok {
			out = append(out, s.Value)
		}
	}
	return out
}

// avoid an unused math import when build tags remove renderer helpers
var _ = math.Max
