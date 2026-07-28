package object

import (
	"fmt"
	"sync"
)

// UIRect is a logical-pixel rectangle produced by the Z14 layout engine.
type UIRect struct{ X, Y, Width, Height float64 }

// UICanvasCommand is a portable drawing operation understood by the headless,
// SDL3 and native C11 renderers.
type UICanvasCommand struct {
	Kind   string
	Values map[string]Object
}

// UIRenderItem is a flattened immutable render record for one retained node.
type UIRenderItem struct {
	ID       string
	Kind     string
	Bounds   UIRect
	Props    map[string]Object
	Focused  bool
	Hovered  bool
	Commands []UICanvasCommand
}

// UIRenderFrame is the portable output of layout + styling.
type UIRenderFrame struct {
	Width, Height float64
	Background    string
	Items         []UIRenderItem
}

// DesktopUIRenderer is an optional capability implemented by desktop windows.
// Headless windows keep the last frame for tests; SDL3 windows draw it.
type DesktopUIRenderer interface {
	RenderUI(frame *UIRenderFrame) error
	LastUIFrame() *UIRenderFrame
}

type UINode struct {
	Mu       sync.RWMutex
	ID       string
	Kind     string
	Props    map[string]Object
	Bindings map[string]*UIState
	Children []*UINode
	Parent   *UINode
	Context  *UIContext
	Bounds   UIRect
	Visible  bool
	Enabled  bool
}

func (n *UINode) Type() ObjectType { return UI_NODE_OBJ }
func (n *UINode) Inspect() string {
	if n == nil {
		return "UINode<nil>"
	}
	n.Mu.RLock()
	defer n.Mu.RUnlock()
	return fmt.Sprintf("UINode<%s id=%q children=%d>", n.Kind, n.ID, len(n.Children))
}

type UIStateBinding struct {
	Node     *UINode
	Property string
}
type UIState struct {
	Mu          sync.RWMutex
	Value       Object
	Bindings    []UIStateBinding
	Subscribers []Object
}

func (s *UIState) Type() ObjectType { return UI_STATE_OBJ }
func (s *UIState) Inspect() string {
	if s == nil || s.Value == nil {
		return "UIState<null>"
	}
	s.Mu.RLock()
	defer s.Mu.RUnlock()
	return fmt.Sprintf("UIState<%s>", s.Value.Inspect())
}

type UITheme struct {
	Name   string
	Values map[string]Object
}

func (t *UITheme) Type() ObjectType { return UI_THEME_OBJ }
func (t *UITheme) Inspect() string {
	if t == nil {
		return "UITheme<nil>"
	}
	return fmt.Sprintf("UITheme<%s>", t.Name)
}

type UIContext struct {
	Mu        sync.RWMutex
	App       *DesktopApp
	Window    *DesktopWindow
	Root      *UINode
	Theme     *UITheme
	Nodes     map[string]*UINode
	FocusID   string
	HoverID   string
	Dirty     bool
	Closed    bool
	LastFrame *UIRenderFrame
}

func (c *UIContext) Type() ObjectType { return UI_CONTEXT_OBJ }
func (c *UIContext) Inspect() string {
	if c == nil || c.Window == nil || c.Window.Runtime == nil {
		return "UIContext<nil>"
	}
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	return fmt.Sprintf("UIContext<window=%d nodes=%d dirty=%t>", c.Window.Runtime.ID(), len(c.Nodes), c.Dirty)
}
