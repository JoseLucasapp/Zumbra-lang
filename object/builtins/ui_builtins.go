package builtins

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync/atomic"
	"unicode"

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
func applyUINodePropertyLocked(node *object.UINode, property string, value object.Object) {
	node.Props[property] = value
	if property == "visible" {
		if boolean, ok := value.(*object.Boolean); ok {
			node.Visible = boolean.Value
		}
	}
	if property == "disabled" {
		if boolean, ok := value.(*object.Boolean); ok {
			node.Enabled = !boolean.Value
		}
	}
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
			applyUINodePropertyLocked(binding.Node, binding.Property, args[1])
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
		applyUINodePropertyLocked(n, property, value)
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
		ctx.FocusID = ""
		ctx.Mu.Unlock()
		if textErr := syncUITextInput(ctx, nil); textErr != nil {
			return textErr
		}
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
		applyUINodePropertyLocked(n, property, args[2])
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
		if textErr := syncUITextInput(ctx, n); textErr != nil {
			return textErr
		}
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
		accessibilityRoot := root
		if modal := topmostVisibleModal(root); modal != nil {
			accessibilityRoot = modal
		}
		items := []object.Object{}
		collectAccessibility(accessibilityRoot, &items)
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
func uiNodeAcceptsTextInput(n *object.UINode) bool {
	if n == nil {
		return false
	}
	n.Mu.RLock()
	defer n.Mu.RUnlock()
	return n.Visible && n.Enabled && (n.Kind == "input" || n.Kind == "textarea")
}

func syncUITextInput(ctx *object.UIContext, node *object.UINode) *object.Error {
	if ctx == nil || ctx.Window == nil || ctx.Window.Runtime == nil {
		return nil
	}
	if err := ctx.Window.Runtime.SetTextInput(uiNodeAcceptsTextInput(node)); err != nil {
		return NewError("UI text input: %s", err)
	}
	return nil
}

func uiNodeEffectivelyVisible(n *object.UINode) bool {
	for current := n; current != nil; {
		current.Mu.RLock()
		visible := current.Visible
		parent := current.Parent
		current.Mu.RUnlock()
		if !visible {
			return false
		}
		current = parent
	}
	return n != nil
}

func uiNodeFocusable(n *object.UINode) bool {
	if n == nil || !uiNodeEffectivelyVisible(n) {
		return false
	}
	n.Mu.RLock()
	defer n.Mu.RUnlock()
	if !n.Enabled {
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
	focusRoot := root
	if modal := topmostVisibleModal(root); modal != nil {
		focusRoot = modal
	}
	nodes := []*object.UINode{}
	walkUI(focusRoot, func(n *object.UINode) {
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
	_ = syncUITextInput(ctx, nodes[index])
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
	if !visible {
		return
	}
	role := optionString(props, "role", defaultUIRole(node.Kind))
	label := optionString(props, "accessibilityLabel", optionString(props, "text", optionString(props, "label", "")))
	*items = append(*items, objectMapDict(map[string]object.Object{"id": NewString(node.ID), "role": NewString(role), "label": NewString(label), "description": NewString(optionString(props, "accessibilityDescription", "")), "enabled": NewBoolean(enabled), "focusable": NewBoolean(uiNodeFocusable(node)), "x": NewFloat(bounds.X), "y": NewFloat(bounds.Y), "width": NewFloat(bounds.Width), "height": NewFloat(bounds.Height)}))
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
		items = append(items, objectMapDict(map[string]object.Object{"id": NewString(item.ID), "kind": NewString(item.Kind), "x": NewFloat(item.Bounds.X), "y": NewFloat(item.Bounds.Y), "width": NewFloat(item.Bounds.Width), "height": NewFloat(item.Bounds.Height), "contentX": NewFloat(item.ContentBounds.X), "contentY": NewFloat(item.ContentBounds.Y), "contentWidth": NewFloat(item.ContentBounds.Width), "contentHeight": NewFloat(item.ScrollContentHeight), "scrollOffsetY": NewFloat(item.ScrollOffsetY), "focused": NewBoolean(item.Focused), "hovered": NewBoolean(item.Hovered), "props": objectMapDict(item.Props)}))
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

func uiSelectCurrentIndex(props map[string]object.Object, items []string) int {
	value := optionString(props, "value", "")
	for index, item := range items {
		if item == value {
			return index
		}
	}
	index := int(optionInt(props, "selectedIndex", 0))
	if index < 0 || index >= len(items) {
		return 0
	}
	return index
}

func uiSelectPopupGeometry(node *object.UINode, viewport object.UIRect) (object.UIRect, float64, int) {
	if node == nil {
		return object.UIRect{}, 0, 0
	}
	node.Mu.RLock()
	bounds := node.Bounds
	props := cloneUIProps(node.Props)
	node.Mu.RUnlock()
	items := uiArrayStrings(props["options"])
	if len(items) == 0 {
		return object.UIRect{}, 0, 0
	}
	rowHeight := math.Max(28, uiPropNumber(props, "optionHeight", math.Max(34, bounds.Height)))
	maxVisible := int(math.Max(1, uiPropNumber(props, "maxVisibleOptions", 8)))
	visible := len(items)
	if visible > maxVisible {
		visible = maxVisible
	}
	height := rowHeight * float64(visible)
	width := math.Max(bounds.Width, uiPropNumber(props, "dropdownWidth", bounds.Width))
	x := bounds.X
	if x+width > viewport.X+viewport.Width {
		x = math.Max(viewport.X, viewport.X+viewport.Width-width)
	}
	y := bounds.Y + bounds.Height + 2
	if y+height > viewport.Y+viewport.Height && bounds.Y-2-height >= viewport.Y {
		y = bounds.Y - 2 - height
	}
	if y+height > viewport.Y+viewport.Height {
		height = math.Max(rowHeight, viewport.Y+viewport.Height-y)
	}
	return object.UIRect{X: x, Y: y, Width: width, Height: height}, rowHeight, visible
}

func topmostOpenSelect(node *object.UINode) *object.UINode {
	if node == nil {
		return nil
	}
	node.Mu.RLock()
	visible := node.Visible
	kind := node.Kind
	props := cloneUIProps(node.Props)
	children := append([]*object.UINode{}, node.Children...)
	node.Mu.RUnlock()
	if !visible {
		return nil
	}
	for index := len(children) - 1; index >= 0; index-- {
		if selected := topmostOpenSelect(children[index]); selected != nil {
			return selected
		}
	}
	if kind == "select" && optionBool(props, "open", false) {
		return node
	}
	return nil
}

func setUISelectOpen(node *object.UINode, open bool) {
	if node == nil || node.Kind != "select" {
		return
	}
	node.Mu.Lock()
	node.Props["open"] = NewBoolean(open)
	node.Mu.Unlock()
}

func closeOpenUISelects(node *object.UINode, except *object.UINode) bool {
	if node == nil {
		return false
	}
	changed := false
	node.Mu.RLock()
	children := append([]*object.UINode{}, node.Children...)
	kind := node.Kind
	open := optionBool(node.Props, "open", false)
	node.Mu.RUnlock()
	if kind == "select" && node != except && open {
		setUISelectOpen(node, false)
		changed = true
	}
	for _, child := range children {
		if closeOpenUISelects(child, except) {
			changed = true
		}
	}
	return changed
}

func updateUISelectValue(target *object.UINode, props map[string]object.Object, index int, invoke func(string) *object.Error) *object.Error {
	items := uiArrayStrings(props["options"])
	if len(items) == 0 {
		return nil
	}
	if index < 0 {
		index = 0
	}
	if index >= len(items) {
		index = len(items) - 1
	}
	selectedIndex := NewInteger(int64(index))
	selectedValue := NewString(items[index])
	target.Mu.Lock()
	target.Props["selectedIndex"] = selectedIndex
	target.Props["value"] = selectedValue
	target.Props["open"] = NewBoolean(false)
	target.Mu.Unlock()
	if e := updateUIBoundState(target, "selectedIndex", selectedIndex); e != nil {
		return e
	}
	if e := updateUIBoundState(target, "value", selectedValue); e != nil {
		return e
	}
	return invoke("onChange")
}

func uiTextIndices(props map[string]object.Object) ([]rune, int, int, int, int) {
	runes := []rune(optionString(props, "value", ""))
	length := len(runes)
	caret := int(optionInt(props, "caretIndex", int64(length)))
	anchor := int(optionInt(props, "selectionAnchor", int64(caret)))
	if caret < 0 {
		caret = 0
	}
	if caret > length {
		caret = length
	}
	if anchor < 0 {
		anchor = 0
	}
	if anchor > length {
		anchor = length
	}
	start, end := caret, anchor
	if start > end {
		start, end = end, start
	}
	return runes, caret, anchor, start, end
}

func setUITextSelection(target *object.UINode, caret, anchor int) {
	target.Mu.Lock()
	length := len([]rune(optionString(target.Props, "value", "")))
	if caret < 0 {
		caret = 0
	}
	if caret > length {
		caret = length
	}
	if anchor < 0 {
		anchor = 0
	}
	if anchor > length {
		anchor = length
	}
	start, end := caret, anchor
	if start > end {
		start, end = end, start
	}
	target.Props["caretIndex"] = NewInteger(int64(caret))
	target.Props["selectionAnchor"] = NewInteger(int64(anchor))
	target.Props["selectionStart"] = NewInteger(int64(start))
	target.Props["selectionEnd"] = NewInteger(int64(end))
	target.Mu.Unlock()
}

func updateUITextValue(target *object.UINode, runes []rune, caret int, invoke func(string) *object.Error) *object.Error {
	value := NewString(string(runes))
	target.Mu.Lock()
	target.Props["value"] = value
	target.Mu.Unlock()
	setUITextSelection(target, caret, caret)
	if e := updateUIBoundState(target, "value", value); e != nil {
		return e
	}
	if e := invoke("onInput"); e != nil {
		return e
	}
	return invoke("onChange")
}

func replaceUITextSelection(target *object.UINode, props map[string]object.Object, replacement string, invoke func(string) *object.Error) *object.Error {
	runes, _, _, start, end := uiTextIndices(props)
	added := []rune(replacement)
	next := make([]rune, 0, len(runes)-(end-start)+len(added))
	next = append(next, runes[:start]...)
	next = append(next, added...)
	next = append(next, runes[end:]...)
	return updateUITextValue(target, next, start+len(added), invoke)
}

func uiTextWordLeft(runes []rune, index int) int {
	if index > len(runes) {
		index = len(runes)
	}
	for index > 0 && unicode.IsSpace(runes[index-1]) {
		index--
	}
	for index > 0 && !unicode.IsSpace(runes[index-1]) {
		index--
	}
	return index
}
func uiTextWordRight(runes []rune, index int) int {
	if index < 0 {
		index = 0
	}
	for index < len(runes) && unicode.IsSpace(runes[index]) {
		index++
	}
	for index < len(runes) && !unicode.IsSpace(runes[index]) {
		index++
	}
	return index
}

func uiTextMeasurer(ctx *object.UIContext) object.DesktopUITextMeasurer {
	if ctx != nil && ctx.Window != nil && ctx.Window.Runtime != nil {
		if m, ok := ctx.Window.Runtime.(object.DesktopUITextMeasurer); ok {
			return m
		}
	}
	return nil
}

func uiTextVisibleStart(ctx *object.UIContext, props map[string]object.Object, runes []rune, caret int, available float64) int {
	if available <= 0 {
		return caret
	}
	start := 0
	for start < caret {
		metrics := uiMeasureText(string(runes[start:caret]), props, ctx.Theme, uiTextMeasurer(ctx), 0)
		if metrics.Width <= available {
			break
		}
		start++
	}
	return start
}

func uiTextCaretFromX(ctx *object.UIContext, target *object.UINode, x float64) int {
	target.Mu.RLock()
	props := cloneUIProps(target.Props)
	bounds := target.Bounds
	target.Mu.RUnlock()
	runes, caret, _, _, _ := uiTextIndices(props)
	available := math.Max(0, bounds.Width-16)
	start := uiTextVisibleStart(ctx, props, runes, caret, available)
	relative := x - bounds.X - 8
	if relative <= 0 {
		return start
	}
	previous := 0.0
	for index := start; index < len(runes); index++ {
		width := uiMeasureText(string(runes[start:index+1]), props, ctx.Theme, uiTextMeasurer(ctx), 0).Width
		if relative < previous+(width-previous)/2 {
			return index
		}
		if relative < width {
			return index + 1
		}
		previous = width
		if width > available {
			break
		}
	}
	return len(runes)
}

func uiTextSelectWord(target *object.UINode, index int) {
	target.Mu.RLock()
	props := cloneUIProps(target.Props)
	target.Mu.RUnlock()
	runes, _, _, _, _ := uiTextIndices(props)
	if len(runes) == 0 {
		setUITextSelection(target, 0, 0)
		return
	}
	if index >= len(runes) {
		index = len(runes) - 1
	}
	if index < 0 {
		index = 0
	}
	start, end := index, index
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}
	for end < len(runes) && !unicode.IsSpace(runes[end]) {
		end++
	}
	setUITextSelection(target, end, start)
}

func uiTextSelected(props map[string]object.Object) string {
	runes, _, _, start, end := uiTextIndices(props)
	if start == end {
		return ""
	}
	return string(runes[start:end])
}

func handleUITextKey(ctx *object.UIContext, target *object.UINode, props map[string]object.Object, event *object.DesktopEvent, invoke func(string) *object.Error) (bool, *object.Error) {
	key := strings.ToLower(optionString(event.Data, "key", ""))
	shortcut := strings.ToLower(optionString(event.Data, "shortcut", ""))
	runes, caret, anchor, start, end := uiTextIndices(props)
	shift := strings.Contains(shortcut, "shift+")
	ctrl := strings.Contains(shortcut, "ctrl+") || strings.Contains(shortcut, "super+")
	move := func(next int) {
		if shift {
			setUITextSelection(target, next, anchor)
		} else {
			setUITextSelection(target, next, next)
		}
	}
	switch shortcut {
	case "ctrl+a", "super+a":
		setUITextSelection(target, len(runes), 0)
		return true, nil
	case "ctrl+c", "super+c":
		selected := uiTextSelected(props)
		if selected != "" && ctx.App != nil && ctx.App.Backend != nil {
			if err := ctx.App.Backend.SetClipboard(selected); err != nil {
				return true, NewError("UI clipboard copy: %s", err)
			}
		}
		return true, nil
	case "ctrl+x", "super+x":
		selected := uiTextSelected(props)
		if selected != "" && ctx.App != nil && ctx.App.Backend != nil {
			if err := ctx.App.Backend.SetClipboard(selected); err != nil {
				return true, NewError("UI clipboard cut: %s", err)
			}
			return true, updateUITextValue(target, append(append([]rune{}, runes[:start]...), runes[end:]...), start, invoke)
		}
		return true, nil
	case "ctrl+v", "super+v":
		if ctx.App != nil && ctx.App.Backend != nil {
			text, err := ctx.App.Backend.Clipboard()
			if err != nil {
				return true, NewError("UI clipboard paste: %s", err)
			}
			return true, replaceUITextSelection(target, props, text, invoke)
		}
		return true, nil
	}
	left := key == "left" || key == "arrowleft"
	right := key == "right" || key == "arrowright"
	if left {
		next := caret
		if !shift && start != end {
			next = start
		} else if ctrl {
			next = uiTextWordLeft(runes, caret)
		} else if next > 0 {
			next--
		}
		move(next)
		return true, nil
	}
	if right {
		next := caret
		if !shift && start != end {
			next = end
		} else if ctrl {
			next = uiTextWordRight(runes, caret)
		} else if next < len(runes) {
			next++
		}
		move(next)
		return true, nil
	}
	if key == "home" {
		move(0)
		return true, nil
	}
	if key == "end" {
		move(len(runes))
		return true, nil
	}
	if key == "backspace" {
		if start != end {
			return true, updateUITextValue(target, append(append([]rune{}, runes[:start]...), runes[end:]...), start, invoke)
		}
		if caret > 0 {
			return true, updateUITextValue(target, append(append([]rune{}, runes[:caret-1]...), runes[caret:]...), caret-1, invoke)
		}
		return true, nil
	}
	if key == "delete" {
		if start != end {
			return true, updateUITextValue(target, append(append([]rune{}, runes[:start]...), runes[end:]...), start, invoke)
		}
		if caret < len(runes) {
			return true, updateUITextValue(target, append(append([]rune{}, runes[:caret]...), runes[caret+1:]...), caret, invoke)
		}
		return true, nil
	}
	return false, nil
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
	popupRoot := root
	if modal := topmostVisibleModal(root); modal != nil {
		popupRoot = modal
	}
	openSelect := topmostOpenSelect(popupRoot)
	var openPopup object.UIRect
	var openPopupRowHeight float64
	if openSelect != nil {
		root.Mu.RLock()
		viewport := root.Bounds
		root.Mu.RUnlock()
		openPopup, openPopupRowHeight, _ = uiSelectPopupGeometry(openSelect, viewport)
	}
	switch event.Type {
	case "mouse_motion", "mouse_down", "mouse_up", "mouse_button_down", "mouse_button_up", "mouse_wheel":
		if openSelect != nil && uiPointInRect(x, y, openPopup) {
			target = openSelect
		} else {
			target = hitTestUI(root, x, y)
		}
		if event.Type == "mouse_motion" && uiEventNumber(event.Data, "buttons") != 0 && focusID != "" {
			ctx.Mu.RLock()
			focused := ctx.Nodes[focusID]
			ctx.Mu.RUnlock()
			if focused != nil {
				focused.Mu.RLock()
				selecting := optionBool(focused.Props, "selecting", false)
				editable := focused.Kind == "input" || focused.Kind == "textarea"
				focused.Mu.RUnlock()
				if selecting && editable {
					target = focused
				}
			}
		}
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
			closeOpenUISelects(root, nil)
			focusNextUI(ctx, shortcut == "shift+tab")
			return nil
		}
	}
	if event.Type == "mouse_wheel" && openSelect != nil && target == openSelect && uiPointInRect(x, y, openPopup) {
		openSelect.Mu.RLock()
		selectProps := cloneUIProps(openSelect.Props)
		openSelect.Mu.RUnlock()
		items := uiArrayStrings(selectProps["options"])
		maxVisible := int(math.Max(1, uiPropNumber(selectProps, "maxVisibleOptions", 8)))
		maxOffset := maxInt(0, len(items)-maxVisible)
		offset := int(optionInt(selectProps, "popupOffset", 0))
		dy := uiEventNumber(event.Data, "dy")
		if dy == 0 {
			dy = uiEventNumber(event.Data, "wheelY")
		}
		if dy > 0 {
			offset--
		} else if dy < 0 {
			offset++
		}
		if offset < 0 {
			offset = 0
		}
		if offset > maxOffset {
			offset = maxOffset
		}
		openSelect.Mu.Lock()
		openSelect.Props["popupOffset"] = NewInteger(int64(offset))
		openSelect.Mu.Unlock()
		ctx.Mu.Lock()
		ctx.Dirty = true
		ctx.Mu.Unlock()
		return nil
	}
	if event.Type == "mouse_wheel" {
		scrollNode := target
		for scrollNode != nil {
			scrollNode.Mu.RLock()
			props := cloneUIProps(scrollNode.Props)
			parent := scrollNode.Parent
			scrollNode.Mu.RUnlock()
			if uiScrollableY(props) {
				dy := uiEventNumber(event.Data, "dy")
				if dy == 0 {
					dy = uiEventNumber(event.Data, "wheelY")
				}
				step := uiPropNumber(props, "scrollStep", 48)
				scrollNode.Mu.Lock()
				maxOffset := math.Max(0, scrollNode.ContentHeight-scrollNode.ContentBounds.Height)
				scrollNode.ScrollOffsetY = uiClamp(scrollNode.ScrollOffsetY-dy*step, 0, maxOffset)
				scrollNode.Mu.Unlock()
				ctx.Mu.Lock()
				ctx.Dirty = true
				ctx.Mu.Unlock()
				return nil
			}
			scrollNode = parent
		}
		return nil
	}
	down := event.Type == "mouse_down" || event.Type == "mouse_button_down"
	if down && openSelect != nil && target != openSelect {
		if closeOpenUISelects(root, nil) {
			ctx.Mu.Lock()
			ctx.Dirty = true
			ctx.Mu.Unlock()
		}
	}
	if target == nil {
		if down {
			ctx.Mu.Lock()
			ctx.FocusID = ""
			ctx.Dirty = true
			ctx.Mu.Unlock()
			if textErr := syncUITextInput(ctx, nil); textErr != nil {
				return textErr
			}
		}
		return nil
	}
	if !target.Enabled {
		return nil
	}
	target.Mu.RLock()
	kind := target.Kind
	targetID := target.ID
	props := cloneUIProps(target.Props)
	target.Mu.RUnlock()
	if event.Data == nil {
		event.Data = map[string]object.Object{}
	}
	event.Data["targetId"] = NewString(targetID)
	event.Data["targetKind"] = NewString(kind)
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

	if down && (kind == "input" || kind == "textarea") {
		index := uiTextCaretFromX(ctx, target, x)
		clicks := int(uiEventNumber(event.Data, "clicks"))
		if clicks >= 2 {
			uiTextSelectWord(target, index)
		} else {
			setUITextSelection(target, index, index)
		}
		target.Mu.Lock()
		target.Props["selecting"] = NewBoolean(true)
		target.Mu.Unlock()
		ctx.Mu.Lock()
		ctx.Dirty = true
		ctx.Mu.Unlock()
	}
	if event.Type == "key_down" && kind == "select" {
		key := ""
		if value, ok := event.Data["key"].(*object.String); ok {
			key = strings.ToLower(value.Value)
		}
		items := uiArrayStrings(props["options"])
		if key == "escape" {
			setUISelectOpen(target, false)
			ctx.Mu.Lock()
			ctx.Dirty = true
			ctx.Mu.Unlock()
			return nil
		}
		if (key == "arrowdown" || key == "down" || key == "arrowup" || key == "up") && len(items) > 0 {
			index := uiSelectCurrentIndex(props, items)
			if key == "arrowup" || key == "up" {
				index = (index + len(items) - 1) % len(items)
			} else {
				index = (index + 1) % len(items)
			}
			if e := updateUISelectValue(target, props, index, invoke); e != nil {
				return e
			}
			setUISelectOpen(target, true)
			ctx.Mu.Lock()
			ctx.Dirty = true
			ctx.Mu.Unlock()
			return nil
		}
	}
	up := event.Type == "mouse_up" || event.Type == "mouse_button_up"
	if event.Type == "mouse_motion" && (kind == "input" || kind == "textarea") && optionBool(props, "selecting", false) && uiEventNumber(event.Data, "buttons") != 0 {
		_, _, anchor, _, _ := uiTextIndices(props)
		setUITextSelection(target, uiTextCaretFromX(ctx, target, x), anchor)
		ctx.Mu.Lock()
		ctx.Dirty = true
		ctx.Mu.Unlock()
		return nil
	}
	if up && (kind == "input" || kind == "textarea") {
		target.Mu.Lock()
		target.Props["selecting"] = NewBoolean(false)
		target.Mu.Unlock()
	}
	if up && kind == "select" && openSelect == target && uiPointInRect(x, y, openPopup) && openPopupRowHeight > 0 {
		index := int(optionInt(props, "popupOffset", 0)) + int((y-openPopup.Y)/openPopupRowHeight)
		if e := updateUISelectValue(target, props, index, invoke); e != nil {
			return e
		}
		ctx.Mu.Lock()
		ctx.Dirty = true
		ctx.Mu.Unlock()
		return nil
	}
	if down {
		if uiNodeFocusable(target) {
			ctx.Mu.Lock()
			ctx.FocusID = target.ID
			ctx.Dirty = true
			ctx.Mu.Unlock()
			if textErr := syncUITextInput(ctx, target); textErr != nil {
				return textErr
			}
			if e := invoke("onFocus"); e != nil {
				return e
			}
		} else {
			ctx.Mu.Lock()
			ctx.FocusID = ""
			ctx.Dirty = true
			ctx.Mu.Unlock()
			if textErr := syncUITextInput(ctx, nil); textErr != nil {
				return textErr
			}
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
		case "select":
			open := optionBool(props, "open", false)
			closeOpenUISelects(root, target)
			if !open {
				items := uiArrayStrings(props["options"])
				selected := uiSelectCurrentIndex(props, items)
				maxVisible := int(math.Max(1, uiPropNumber(props, "maxVisibleOptions", 8)))
				offset := 0
				if selected >= maxVisible {
					offset = selected - maxVisible + 1
				}
				target.Mu.Lock()
				target.Props["popupOffset"] = NewInteger(int64(offset))
				target.Mu.Unlock()
			}
			setUISelectOpen(target, !open)
		case "tabs":
			items := uiArrayStrings(props["tabs"])
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
		handled, e := handleUITextKey(ctx, target, props, event, invoke)
		if e != nil {
			return e
		}
		if handled {
			ctx.Mu.Lock()
			ctx.Dirty = true
			ctx.Mu.Unlock()
			return nil
		}
	}
	if event.Type == "text_input" && (kind == "input" || kind == "textarea") {
		text := optionString(event.Data, "text", "")
		if text != "" {
			target.Mu.RLock()
			latest := cloneUIProps(target.Props)
			target.Mu.RUnlock()
			if e := replaceUITextSelection(target, latest, text, invoke); e != nil {
				return e
			}
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
	if modal := topmostVisibleModal(node); modal != nil {
		if hit := hitTestUIClipped(modal, x, y, nil); hit != nil {
			return hit
		}
		// The modal backdrop consumes clicks outside the dialog content so the
		// underlying interface cannot be activated accidentally.
		return modal
	}
	return hitTestUIClipped(node, x, y, nil)
}

func topmostVisibleModal(node *object.UINode) *object.UINode {
	if node == nil {
		return nil
	}
	node.Mu.RLock()
	visible := node.Visible
	kind := node.Kind
	children := append([]*object.UINode{}, node.Children...)
	node.Mu.RUnlock()
	if !visible {
		return nil
	}
	for index := len(children) - 1; index >= 0; index-- {
		if modal := topmostVisibleModal(children[index]); modal != nil {
			return modal
		}
	}
	if kind == "modal" {
		return node
	}
	return nil
}

func hitTestUIClipped(node *object.UINode, x, y float64, inheritedClip *object.UIRect) *object.UINode {
	if node == nil {
		return nil
	}
	node.Mu.RLock()
	visible := node.Visible
	bounds := node.Bounds
	contentBounds := node.ContentBounds
	props := cloneUIProps(node.Props)
	children := append([]*object.UINode{}, node.Children...)
	node.Mu.RUnlock()
	if !visible {
		return nil
	}
	if inheritedClip != nil && !uiPointInRect(x, y, *inheritedClip) {
		return nil
	}
	childClip := inheritedClip
	if uiClipsChildren(props) {
		viewport := contentBounds
		if inheritedClip != nil {
			intersection, ok := uiRectIntersection(*inheritedClip, viewport)
			if !ok {
				return nil
			}
			viewport = intersection
		}
		childClip = &viewport
	}
	for i := len(children) - 1; i >= 0; i-- {
		if hit := hitTestUIClipped(children[i], x, y, childClip); hit != nil {
			return hit
		}
	}
	if uiPointInRect(x, y, bounds) {
		return node
	}
	return nil
}

func uiPointInRect(x, y float64, bounds object.UIRect) bool {
	return x >= bounds.X && y >= bounds.Y && x <= bounds.X+bounds.Width && y <= bounds.Y+bounds.Height
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
