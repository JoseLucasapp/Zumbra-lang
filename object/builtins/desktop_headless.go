package builtins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"zumbra/object"
)

type headlessDesktopBackend struct {
	mu        sync.Mutex
	nextID    int64
	windows   map[int64]*headlessWindow
	events    chan *object.DesktopEvent
	clipboard string
	closed    bool
}

func newHeadlessDesktopBackend() *headlessDesktopBackend {
	backend := &headlessDesktopBackend{
		windows: map[int64]*headlessWindow{},
		events:  make(chan *object.DesktopEvent, 256),
	}
	backend.emit("ready", 0, map[string]object.Object{"backend": NewString("headless")})
	return backend
}

func (b *headlessDesktopBackend) Name() string { return "headless" }
func (b *headlessDesktopBackend) emit(kind string, windowID int64, data map[string]object.Object) {
	event := &object.DesktopEvent{Type: kind, WindowID: windowID, Timestamp: time.Now().UnixNano(), Data: data}
	select {
	case b.events <- event:
	default:
		// Preserve the runtime instead of deadlocking when user code stops polling.
	}
}
func (b *headlessDesktopBackend) CreateWindow(options map[string]object.Object) (object.DesktopWindowRuntime, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errors.New("desktop backend is closed")
	}
	id := atomic.AddInt64(&b.nextID, 1)
	title := optionString(options, "title", "Zumbra")
	width := optionInt(options, "width", 800)
	height := optionInt(options, "height", 600)
	if width < 1 || height < 1 {
		return nil, errors.New("window width and height must be positive")
	}
	window := &headlessWindow{backend: b, id: id, title: title, width: width, height: height, pixelWidth: width, pixelHeight: height, open: true, visible: !optionBool(options, "hidden", false), scale: 1, density: 1, x: 0, y: 0}
	if optionBool(options, "highDPI", true) {
		window.scale = optionFloat(options, "scale", 1)
		if window.scale <= 0 {
			window.scale = 1
		}
		window.density = window.scale
		window.pixelWidth = int64(float64(width) * window.density)
		window.pixelHeight = int64(float64(height) * window.density)
	}
	window.fullscreen = optionBool(options, "fullscreen", false)
	b.windows[id] = window
	if window.visible {
		b.emit("window_shown", id, map[string]object.Object{})
	}
	b.emit("window_resized", id, map[string]object.Object{"width": NewInteger(width), "height": NewInteger(height)})
	b.emit("window_pixel_size_changed", id, map[string]object.Object{"width": NewInteger(window.pixelWidth), "height": NewInteger(window.pixelHeight), "scale": NewFloat(window.scale)})
	return window, nil
}
func (b *headlessDesktopBackend) PollEvent(timeoutMS int64) (*object.DesktopEvent, error) {
	if timeoutMS < 0 {
		event, ok := <-b.events
		if !ok {
			return nil, nil
		}
		return event, nil
	}
	if timeoutMS == 0 {
		select {
		case event, ok := <-b.events:
			if !ok {
				return nil, nil
			}
			return event, nil
		default:
			return nil, nil
		}
	}
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	defer timer.Stop()
	select {
	case event, ok := <-b.events:
		if !ok {
			return nil, nil
		}
		return event, nil
	case <-timer.C:
		return nil, nil
	}
}
func (b *headlessDesktopBackend) SetClipboard(text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("desktop backend is closed")
	}
	b.clipboard = text
	b.emit("clipboard_updated", 0, map[string]object.Object{})
	return nil
}
func (b *headlessDesktopBackend) Clipboard() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return "", errors.New("desktop backend is closed")
	}
	return b.clipboard, nil
}
func (b *headlessDesktopBackend) PickFile(options map[string]object.Object) ([]string, error) {
	if value := optionString(options, "defaultPath", ""); value != "" {
		return []string{value}, nil
	}
	if value := os.Getenv("ZUMBRA_DESKTOP_PICK_FILE"); value != "" {
		return []string{value}, nil
	}
	return []string{}, nil
}
func (b *headlessDesktopBackend) PickFolder(options map[string]object.Object) (string, error) {
	if value := optionString(options, "defaultPath", ""); value != "" {
		return value, nil
	}
	return os.Getenv("ZUMBRA_DESKTOP_PICK_FOLDER"), nil
}
func (b *headlessDesktopBackend) Notify(options map[string]object.Object) error {
	b.emit("notification", 0, map[string]object.Object{"title": NewString(optionString(options, "title", "Zumbra")), "body": NewString(optionString(options, "body", ""))})
	return nil
}
func (b *headlessDesktopBackend) CreateTray(options map[string]object.Object) (object.DesktopTrayRuntime, error) {
	tray := &headlessTray{backend: b, open: true, tooltip: optionString(options, "tooltip", "")}
	b.emit("tray_created", 0, map[string]object.Object{"tooltip": NewString(tray.tooltip)})
	return tray, nil
}
func (b *headlessDesktopBackend) Paths() map[string]string { return desktopPaths() }
func (b *headlessDesktopBackend) OpenExternal(target string) error {
	if target == "" {
		return errors.New("target cannot be empty")
	}
	b.emit("external_opened", 0, map[string]object.Object{"target": NewString(target)})
	return nil
}
func (b *headlessDesktopBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	for _, window := range b.windows {
		window.mu.Lock()
		window.open = false
		window.mu.Unlock()
	}
	b.emit("terminating", 0, map[string]object.Object{})
	return nil
}
func (b *headlessDesktopBackend) inject(event *object.DesktopEvent) {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixNano()
	}
	select {
	case b.events <- event:
	default:
	}
}

type headlessWindow struct {
	backend                                      *headlessDesktopBackend
	mu                                           sync.RWMutex
	id                                           int64
	title                                        string
	width, height, pixelWidth, pixelHeight, x, y int64
	open, visible, fullscreen                    bool
	scale, density                               float64
	icon                                         string
}

func (w *headlessWindow) ID() int64 { return w.id }
func (w *headlessWindow) Show() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return errors.New("window is closed")
	}
	w.visible = true
	w.backend.emit("window_shown", w.id, nil)
	return nil
}
func (w *headlessWindow) Hide() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return errors.New("window is closed")
	}
	w.visible = false
	w.backend.emit("window_hidden", w.id, nil)
	return nil
}
func (w *headlessWindow) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return nil
	}
	w.open = false
	w.backend.emit("window_closed", w.id, nil)
	return nil
}
func (w *headlessWindow) IsOpen() bool  { w.mu.RLock(); defer w.mu.RUnlock(); return w.open }
func (w *headlessWindow) Title() string { w.mu.RLock(); defer w.mu.RUnlock(); return w.title }
func (w *headlessWindow) SetTitle(v string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return errors.New("window is closed")
	}
	w.title = v
	return nil
}
func (w *headlessWindow) Size() (int64, int64) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.width, w.height
}
func (w *headlessWindow) PixelSize() (int64, int64) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.pixelWidth, w.pixelHeight
}
func (w *headlessWindow) SetSize(a, b int64) error {
	if a < 1 || b < 1 {
		return errors.New("window width and height must be positive")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return errors.New("window is closed")
	}
	w.width = a
	w.height = b
	w.pixelWidth = int64(float64(a) * w.density)
	w.pixelHeight = int64(float64(b) * w.density)
	w.backend.emit("window_resized", w.id, map[string]object.Object{"width": NewInteger(a), "height": NewInteger(b)})
	w.backend.emit("window_pixel_size_changed", w.id, map[string]object.Object{"width": NewInteger(w.pixelWidth), "height": NewInteger(w.pixelHeight)})
	return nil
}
func (w *headlessWindow) Position() (int64, int64) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.x, w.y
}
func (w *headlessWindow) SetPosition(a, b int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return errors.New("window is closed")
	}
	w.x = a
	w.y = b
	w.backend.emit("window_moved", w.id, map[string]object.Object{"x": NewInteger(a), "y": NewInteger(b)})
	return nil
}
func (w *headlessWindow) Fullscreen() bool { w.mu.RLock(); defer w.mu.RUnlock(); return w.fullscreen }
func (w *headlessWindow) SetFullscreen(v bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return errors.New("window is closed")
	}
	w.fullscreen = v
	kind := "window_leave_fullscreen"
	if v {
		kind = "window_enter_fullscreen"
	}
	w.backend.emit(kind, w.id, nil)
	return nil
}
func (w *headlessWindow) Maximize() error {
	if !w.IsOpen() {
		return errors.New("window is closed")
	}
	w.backend.emit("window_maximized", w.id, nil)
	return nil
}
func (w *headlessWindow) Minimize() error {
	if !w.IsOpen() {
		return errors.New("window is closed")
	}
	w.backend.emit("window_minimized", w.id, nil)
	return nil
}
func (w *headlessWindow) Restore() error {
	if !w.IsOpen() {
		return errors.New("window is closed")
	}
	w.backend.emit("window_restored", w.id, nil)
	return nil
}
func (w *headlessWindow) Focus() error {
	if !w.IsOpen() {
		return errors.New("window is closed")
	}
	w.backend.emit("window_focus_gained", w.id, nil)
	return nil
}
func (w *headlessWindow) DisplayScale() float64 { w.mu.RLock(); defer w.mu.RUnlock(); return w.scale }
func (w *headlessWindow) PixelDensity() float64 { w.mu.RLock(); defer w.mu.RUnlock(); return w.density }
func (w *headlessWindow) SetIcon(path string) error {
	if path == "" {
		return errors.New("icon path cannot be empty")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.open {
		return errors.New("window is closed")
	}
	w.icon = path
	return nil
}

type headlessTray struct {
	backend *headlessDesktopBackend
	mu      sync.RWMutex
	open    bool
	tooltip string
	items   []string
}

func (t *headlessTray) Add(id, label string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.open {
		return errors.New("tray is closed")
	}
	if id == "" || label == "" {
		return errors.New("tray item id and label are required")
	}
	t.items = append(t.items, id)
	return nil
}
func (t *headlessTray) SetTooltip(v string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.open {
		return errors.New("tray is closed")
	}
	t.tooltip = v
	return nil
}
func (t *headlessTray) Close() error { t.mu.Lock(); defer t.mu.Unlock(); t.open = false; return nil }
func (t *headlessTray) IsOpen() bool { t.mu.RLock(); defer t.mu.RUnlock(); return t.open }

func desktopPaths() map[string]string {
	home, _ := os.UserHomeDir()
	cache, _ := os.UserCacheDir()
	config, _ := os.UserConfigDir()
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" && home != "" {
		data = filepath.Join(home, ".local", "share")
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	executable, _ := os.Executable()
	cwd, _ := os.Getwd()
	return map[string]string{"home": home, "cache": cache, "config": config, "data": data, "runtime": runtimeDir, "executable": executable, "cwd": cwd, "temp": os.TempDir()}
}

func optionString(values map[string]object.Object, key, fallback string) string {
	if value, ok := values[key].(*object.String); ok {
		return value.Value
	}
	return fallback
}
func optionInt(values map[string]object.Object, key string, fallback int64) int64 {
	if value, ok := values[key].(*object.Integer); ok {
		return value.Value
	}
	if value, ok := values[key].(*object.FixedInteger); ok {
		return value.SignedValue()
	}
	return fallback
}
func optionBool(values map[string]object.Object, key string, fallback bool) bool {
	if value, ok := values[key].(*object.Boolean); ok {
		return value.Value
	}
	return fallback
}
func optionFloat(values map[string]object.Object, key string, fallback float64) float64 {
	if value, ok := values[key].(*object.Float); ok {
		return value.Value
	}
	if value, ok := values[key].(*object.Integer); ok {
		return float64(value.Value)
	}
	return fallback
}

func desktopError(prefix string, err error) *object.Error {
	if err == nil {
		return nil
	}
	return NewError("%s: %s", prefix, err)
}
func validateDesktopPath(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
