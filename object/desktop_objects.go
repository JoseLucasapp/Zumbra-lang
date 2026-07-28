package object

import (
	"fmt"
	"sync"
)

// DesktopEvent is the portable event representation shared by all desktop backends.
type DesktopEvent struct {
	Type      string
	WindowID  int64
	Timestamp int64
	Data      map[string]Object
}

// DesktopBackend abstracts the platform window system. The Linux implementation
// uses SDL3, while tests can use the deterministic headless backend.
type DesktopBackend interface {
	Name() string
	CreateWindow(options map[string]Object) (DesktopWindowRuntime, error)
	PollEvent(timeoutMS int64) (*DesktopEvent, error)
	SetClipboard(text string) error
	Clipboard() (string, error)
	PickFile(options map[string]Object) ([]string, error)
	PickFolder(options map[string]Object) (string, error)
	Notify(options map[string]Object) error
	CreateTray(options map[string]Object) (DesktopTrayRuntime, error)
	Paths() map[string]string
	OpenExternal(target string) error
	Close() error
}

type DesktopWindowRuntime interface {
	ID() int64
	Show() error
	Hide() error
	Close() error
	IsOpen() bool
	Title() string
	SetTitle(string) error
	Size() (int64, int64)
	PixelSize() (int64, int64)
	SetSize(int64, int64) error
	Position() (int64, int64)
	SetPosition(int64, int64) error
	Fullscreen() bool
	SetFullscreen(bool) error
	Maximize() error
	Minimize() error
	Restore() error
	Focus() error
	DisplayScale() float64
	PixelDensity() float64
	SetIcon(string) error
}

type DesktopTrayRuntime interface {
	Add(id, label string) error
	SetTooltip(string) error
	Close() error
	IsOpen() bool
}

type DesktopProcessRuntime interface {
	PID() int64
	Wait() (int64, error)
	Kill() error
	Running() bool
}

type DesktopApp struct {
	Backend    DesktopBackend
	Options    map[string]Object
	Mu         sync.RWMutex
	Windows    map[int64]*DesktopWindow
	Trays      []*DesktopTray
	Handlers   map[string][]Object
	Shortcuts  map[string]Object
	UIContexts map[int64]*UIContext
	Running    bool
	Closed     bool
}

func (a *DesktopApp) Type() ObjectType { return DESKTOP_APP_OBJ }
func (a *DesktopApp) Inspect() string {
	if a == nil || a.Backend == nil {
		return "DesktopApp<nil>"
	}
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return fmt.Sprintf("DesktopApp<backend=%s windows=%d running=%t>", a.Backend.Name(), len(a.Windows), a.Running)
}

type DesktopWindow struct {
	App     *DesktopApp
	Runtime DesktopWindowRuntime
}

func (w *DesktopWindow) Type() ObjectType { return DESKTOP_WINDOW_OBJ }
func (w *DesktopWindow) Inspect() string {
	if w == nil || w.Runtime == nil {
		return "DesktopWindow<nil>"
	}
	return fmt.Sprintf("DesktopWindow<id=%d open=%t title=%q>", w.Runtime.ID(), w.Runtime.IsOpen(), w.Runtime.Title())
}

type DesktopTray struct {
	App      *DesktopApp
	Runtime  DesktopTrayRuntime
	Handlers map[string]Object
	Mu       sync.RWMutex
}

func (t *DesktopTray) Type() ObjectType { return DESKTOP_TRAY_OBJ }
func (t *DesktopTray) Inspect() string {
	if t == nil || t.Runtime == nil {
		return "DesktopTray<nil>"
	}
	return fmt.Sprintf("DesktopTray<open=%t>", t.Runtime.IsOpen())
}

type DesktopProcess struct{ Runtime DesktopProcessRuntime }

func (p *DesktopProcess) Type() ObjectType { return DESKTOP_PROCESS_OBJ }
func (p *DesktopProcess) Inspect() string {
	if p == nil || p.Runtime == nil {
		return "DesktopProcess<nil>"
	}
	return fmt.Sprintf("DesktopProcess<pid=%d running=%t>", p.Runtime.PID(), p.Runtime.Running())
}
