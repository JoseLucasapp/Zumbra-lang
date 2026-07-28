package builtins

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"zumbra/object"
)

var (
	desktopInvoker   func(handler object.Object, args ...object.Object) (object.Object, error)
	desktopInvokerMu sync.Mutex
)

// SetDesktopInvoker installs the backend-specific callback bridge used by the
// evaluator and VM. Builtin handlers can always be invoked directly.
func SetDesktopInvoker(invoker func(handler object.Object, args ...object.Object) (object.Object, error)) {
	desktopInvokerMu.Lock()
	desktopInvoker = invoker
	desktopInvokerMu.Unlock()
}

func invokeDesktopHandler(handler object.Object, args ...object.Object) (object.Object, error) {
	if builtin, ok := handler.(*object.Builtin); ok {
		return builtin.Fn(args...), nil
	}
	desktopInvokerMu.Lock()
	invoker := desktopInvoker
	desktopInvokerMu.Unlock()
	if invoker == nil {
		// Reuse the already-configured language callback bridge when HTTP has
		// initialized it. This keeps desktop callbacks consistent with HTTP.
		routeInvokerMu.Lock()
		invoker = routeInvoker
		routeInvokerMu.Unlock()
	}
	if invoker == nil {
		return nil, fmt.Errorf("desktop handler invoker is not configured")
	}
	return invoker(handler, args...)
}

func desktopAppArg(value object.Object, name string) (*object.DesktopApp, *object.Error) {
	app, ok := value.(*object.DesktopApp)
	if !ok {
		return nil, NewError("%s expects DesktopApp, got %s", name, value.Type())
	}
	return app, nil
}
func desktopWindowArg(value object.Object, name string) (*object.DesktopWindow, *object.Error) {
	window, ok := value.(*object.DesktopWindow)
	if !ok {
		return nil, NewError("%s expects DesktopWindow, got %s", name, value.Type())
	}
	return window, nil
}
func desktopTrayArg(value object.Object, name string) (*object.DesktopTray, *object.Error) {
	tray, ok := value.(*object.DesktopTray)
	if !ok {
		return nil, NewError("%s expects DesktopTray, got %s", name, value.Type())
	}
	return tray, nil
}
func desktopProcessArg(value object.Object, name string) (*object.DesktopProcess, *object.Error) {
	process, ok := value.(*object.DesktopProcess)
	if !ok {
		return nil, NewError("%s expects DesktopProcess, got %s", name, value.Type())
	}
	return process, nil
}
func desktopString(value object.Object, name string) (string, *object.Error) {
	text, ok := value.(*object.String)
	if !ok {
		return "", NewError("%s expects string, got %s", name, value.Type())
	}
	return text.Value, nil
}
func desktopInt(value object.Object, name string) (int64, *object.Error) {
	switch integer := value.(type) {
	case *object.Integer:
		return integer.Value, nil
	case *object.FixedInteger:
		return integer.SignedValue(), nil
	default:
		return 0, NewError("%s expects int, got %s", name, value.Type())
	}
}
func desktopBool(value object.Object, name string) (bool, *object.Error) {
	boolean, ok := value.(*object.Boolean)
	if !ok {
		return false, NewError("%s expects bool, got %s", name, value.Type())
	}
	return boolean.Value, nil
}
func desktopOptions(value object.Object, name string) (map[string]object.Object, *object.Error) {
	return objectDictMap(value, name)
}
func desktopPair(a, b int64, first, second string) object.Object {
	return objectMapDict(map[string]object.Object{first: NewInteger(a), second: NewInteger(b)})
}
func desktopStrings(values []string) object.Object {
	elements := make([]object.Object, len(values))
	for i, value := range values {
		elements[i] = NewString(value)
	}
	return &object.Array{Elements: elements}
}

func desktopEventObject(event *object.DesktopEvent) object.Object {
	if event == nil {
		return &object.Null{}
	}
	data := map[string]object.Object{}
	for key, value := range event.Data {
		data[key] = value
	}
	result := map[string]object.Object{
		"type":      NewString(event.Type),
		"windowId":  NewInteger(event.WindowID),
		"timestamp": NewInteger(event.Timestamp),
		"data":      objectMapDict(data),
	}
	// Common event fields are also available directly for ergonomic handlers.
	for key, value := range data {
		result[key] = value
	}
	return objectMapDict(result)
}

func normalizeShortcut(value string) string {
	aliases := map[string]string{
		"control": "ctrl", "cmd": "super", "command": "super", "meta": "super",
		"option": "alt", "return": "enter", "esc": "escape", " ": "space",
	}
	seen := map[string]bool{}
	modifiers := []string{}
	key := ""
	for _, raw := range strings.Split(strings.ToLower(strings.TrimSpace(value)), "+") {
		part := strings.TrimSpace(raw)
		if replacement, ok := aliases[part]; ok {
			part = replacement
		}
		if part == "ctrl" || part == "shift" || part == "alt" || part == "super" {
			if !seen[part] {
				seen[part] = true
				modifiers = append(modifiers, part)
			}
		} else if part != "" {
			key = part
		}
	}
	order := map[string]int{"ctrl": 0, "shift": 1, "alt": 2, "super": 3}
	sort.SliceStable(modifiers, func(i, j int) bool { return order[modifiers[i]] < order[modifiers[j]] })
	if key != "" {
		modifiers = append(modifiers, key)
	}
	return strings.Join(modifiers, "+")
}

func DesktopAppBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopApp expects options")
		}
		options, e := desktopOptions(args[0], "desktopApp")
		if e != nil {
			return e
		}
		backendName := strings.ToLower(optionString(options, "backend", "auto"))
		var backend object.DesktopBackend
		var err error
		switch backendName {
		case "headless":
			backend = newHeadlessDesktopBackend()
		case "auto", "sdl", "sdl3":
			if backendName == "headless" {
				backend = newHeadlessDesktopBackend()
			} else {
				backend, err = newPlatformDesktopBackend(options)
			}
		default:
			return NewError("desktopApp unknown backend %q", backendName)
		}
		if err != nil {
			return desktopError("desktopApp", err)
		}
		return &object.DesktopApp{
			Backend: backend, Options: options, Windows: map[int64]*object.DesktopWindow{},
			Trays: []*object.DesktopTray{}, Handlers: map[string][]object.Object{},
			Shortcuts: map[string]object.Object{}, UIContexts: map[int64]*object.UIContext{},
		}
	}}
}

func DesktopBackendBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopBackend expects app")
		}
		app, e := desktopAppArg(args[0], "desktopBackend")
		if e != nil {
			return e
		}
		if app.Backend == nil {
			return NewString("")
		}
		return NewString(app.Backend.Name())
	}}
}

func DesktopWindowBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopWindow expects app and options")
		}
		app, e := desktopAppArg(args[0], "desktopWindow")
		if e != nil {
			return e
		}
		options, e := desktopOptions(args[1], "desktopWindow")
		if e != nil {
			return e
		}
		app.Mu.RLock()
		closed := app.Closed
		app.Mu.RUnlock()
		if closed {
			return NewError("desktopWindow cannot create a window after app close")
		}
		runtimeWindow, err := app.Backend.CreateWindow(options)
		if err != nil {
			return desktopError("desktopWindow", err)
		}
		window := &object.DesktopWindow{App: app, Runtime: runtimeWindow}
		app.Mu.Lock()
		app.Windows[runtimeWindow.ID()] = window
		app.Mu.Unlock()
		return window
	}}
}

func DesktopOnBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("desktopOn expects app, event name and handler")
		}
		app, e := desktopAppArg(args[0], "desktopOn")
		if e != nil {
			return e
		}
		name, e := desktopString(args[1], "desktopOn event")
		if e != nil {
			return e
		}
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			return NewError("desktopOn event cannot be empty")
		}
		app.Mu.Lock()
		app.Handlers[name] = append(app.Handlers[name], args[2])
		app.Mu.Unlock()
		return app
	}}
}

func DesktopShortcutBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("desktopShortcut expects app, shortcut and handler")
		}
		app, e := desktopAppArg(args[0], "desktopShortcut")
		if e != nil {
			return e
		}
		shortcut, e := desktopString(args[1], "desktopShortcut shortcut")
		if e != nil {
			return e
		}
		shortcut = normalizeShortcut(shortcut)
		if shortcut == "" {
			return NewError("desktopShortcut shortcut cannot be empty")
		}
		app.Mu.Lock()
		app.Shortcuts[shortcut] = args[2]
		app.Mu.Unlock()
		return app
	}}
}

func dispatchDesktopEvent(app *object.DesktopApp, event *object.DesktopEvent) *object.Error {
	if event == nil {
		return nil
	}
	if uiError := dispatchUIForDesktopEvent(app, event); uiError != nil {
		return uiError
	}
	eventObject := desktopEventObject(event)
	app.Mu.RLock()
	handlers := append([]object.Object{}, app.Handlers[strings.ToLower(event.Type)]...)
	handlers = append(handlers, app.Handlers["event"]...)
	var shortcutHandler object.Object
	if event.Type == "key_down" {
		if shortcut, ok := event.Data["shortcut"].(*object.String); ok {
			shortcutHandler = app.Shortcuts[normalizeShortcut(shortcut.Value)]
		}
	}
	trays := append([]*object.DesktopTray{}, app.Trays...)
	app.Mu.RUnlock()
	for _, handler := range handlers {
		if _, err := invokeDesktopHandler(handler, eventObject); err != nil {
			return NewError("desktop event %s: %s", event.Type, err)
		}
	}
	if shortcutHandler != nil {
		if _, err := invokeDesktopHandler(shortcutHandler, eventObject); err != nil {
			return NewError("desktop shortcut: %s", err)
		}
	}
	if event.Type == "tray" {
		id := ""
		if value, ok := event.Data["id"].(*object.String); ok {
			id = value.Value
		}
		for _, tray := range trays {
			tray.Mu.RLock()
			handler := tray.Handlers[id]
			tray.Mu.RUnlock()
			if handler != nil {
				if _, err := invokeDesktopHandler(handler, eventObject); err != nil {
					return NewError("desktop tray item %s: %s", id, err)
				}
			}
		}
	}
	if event.Type == "window_close_requested" && optionBool(app.Options, "closeOnRequest", true) {
		app.Mu.RLock()
		window := app.Windows[event.WindowID]
		app.Mu.RUnlock()
		if window != nil && window.Runtime != nil {
			_ = window.Runtime.Close()
		}
	}
	if event.Type == "window_resized" || event.Type == "window_pixel_size_changed" {
		app.Mu.RLock()
		ctx := app.UIContexts[event.WindowID]
		app.Mu.RUnlock()
		markUIContextDirty(ctx)
	}
	if event.Type == "window_closed" {
		app.Mu.Lock()
		delete(app.Windows, event.WindowID)
		remaining := len(app.Windows)
		app.Mu.Unlock()
		if remaining == 0 && optionBool(app.Options, "quitOnLastWindow", true) {
			app.Mu.Lock()
			app.Running = false
			app.Mu.Unlock()
		}
	}
	if event.Type == "quit" || event.Type == "terminating" {
		app.Mu.Lock()
		app.Running = false
		app.Mu.Unlock()
	}
	return nil
}

func DesktopPollBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopPoll expects app and timeout milliseconds")
		}
		app, e := desktopAppArg(args[0], "desktopPoll")
		if e != nil {
			return e
		}
		timeout, e := desktopInt(args[1], "desktopPoll timeout")
		if e != nil {
			return e
		}
		event, err := app.Backend.PollEvent(timeout)
		if err != nil {
			return desktopError("desktopPoll", err)
		}
		if event == nil {
			if renderError := renderDesktopUIContexts(app); renderError != nil {
				return renderError
			}
			return &object.Null{}
		}
		if dispatchError := dispatchDesktopEvent(app, event); dispatchError != nil {
			return dispatchError
		}
		if renderError := renderDesktopUIContexts(app); renderError != nil {
			return renderError
		}
		return desktopEventObject(event)
	}}
}

func DesktopRunBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopRun expects app")
		}
		app, e := desktopAppArg(args[0], "desktopRun")
		if e != nil {
			return e
		}
		app.Mu.Lock()
		if app.Closed {
			app.Mu.Unlock()
			return NewError("desktopRun cannot run a closed app")
		}
		app.Running = true
		app.Mu.Unlock()
		timeout := optionInt(app.Options, "pollIntervalMs", 16)
		if timeout < 0 {
			timeout = 16
		}
		for {
			app.Mu.RLock()
			running := app.Running && !app.Closed
			app.Mu.RUnlock()
			if !running {
				break
			}
			event, err := app.Backend.PollEvent(timeout)
			if err != nil {
				app.Mu.Lock()
				app.Running = false
				app.Mu.Unlock()
				return desktopError("desktopRun", err)
			}
			if event == nil {
				if renderError := renderDesktopUIContexts(app); renderError != nil {
					app.Mu.Lock()
					app.Running = false
					app.Mu.Unlock()
					return renderError
				}
				continue
			}
			if dispatchError := dispatchDesktopEvent(app, event); dispatchError != nil {
				app.Mu.Lock()
				app.Running = false
				app.Mu.Unlock()
				return dispatchError
			}
			if renderError := renderDesktopUIContexts(app); renderError != nil {
				app.Mu.Lock()
				app.Running = false
				app.Mu.Unlock()
				return renderError
			}
		}
		return NewBoolean(true)
	}}
}

func DesktopQuitBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopQuit expects app")
		}
		app, e := desktopAppArg(args[0], "desktopQuit")
		if e != nil {
			return e
		}
		app.Mu.Lock()
		app.Running = false
		app.Mu.Unlock()
		return NewBoolean(true)
	}}
}
func DesktopRunningBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopRunning expects app")
		}
		app, e := desktopAppArg(args[0], "desktopRunning")
		if e != nil {
			return e
		}
		app.Mu.RLock()
		running := app.Running
		app.Mu.RUnlock()
		return NewBoolean(running)
	}}
}
func DesktopCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopClose expects app")
		}
		app, e := desktopAppArg(args[0], "desktopClose")
		if e != nil {
			return e
		}
		app.Mu.Lock()
		if app.Closed {
			app.Mu.Unlock()
			return NewBoolean(true)
		}
		app.Running = false
		app.Closed = true
		trays := append([]*object.DesktopTray{}, app.Trays...)
		app.Mu.Unlock()
		for _, tray := range trays {
			if tray.Runtime != nil {
				_ = tray.Runtime.Close()
			}
		}
		if err := app.Backend.Close(); err != nil {
			return desktopError("desktopClose", err)
		}
		return NewBoolean(true)
	}}
}

func DesktopEmitBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopEmit expects app and event")
		}
		app, e := desktopAppArg(args[0], "desktopEmit")
		if e != nil {
			return e
		}
		values, e := desktopOptions(args[1], "desktopEmit event")
		if e != nil {
			return e
		}
		kind := optionString(values, "type", "custom")
		windowID := optionInt(values, "windowId", 0)
		data := map[string]object.Object{}
		if raw, ok := values["data"]; ok {
			mapped, errObj := objectDictMap(raw, "desktopEmit data")
			if errObj != nil {
				return errObj
			}
			data = mapped
		}
		for key, value := range values {
			if key != "type" && key != "windowId" && key != "timestamp" && key != "data" {
				data[key] = value
			}
		}
		event := &object.DesktopEvent{Type: kind, WindowID: windowID, Timestamp: time.Now().UnixNano(), Data: data}
		if backend, ok := app.Backend.(*headlessDesktopBackend); ok {
			backend.inject(event)
			return NewBoolean(true)
		}
		// On a graphical backend custom events are dispatched synchronously,
		// avoiding reliance on private native event IDs.
		if errObj := dispatchDesktopEvent(app, event); errObj != nil {
			return errObj
		}
		return NewBoolean(true)
	}}
}

func DesktopSetClipboardBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopSetClipboard expects app and text")
		}
		app, e := desktopAppArg(args[0], "desktopSetClipboard")
		if e != nil {
			return e
		}
		text, e := desktopString(args[1], "desktopSetClipboard text")
		if e != nil {
			return e
		}
		if err := app.Backend.SetClipboard(text); err != nil {
			return desktopError("desktopSetClipboard", err)
		}
		return NewBoolean(true)
	}}
}
func DesktopClipboardBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopClipboard expects app")
		}
		app, e := desktopAppArg(args[0], "desktopClipboard")
		if e != nil {
			return e
		}
		value, err := app.Backend.Clipboard()
		if err != nil {
			return desktopError("desktopClipboard", err)
		}
		return NewString(value)
	}}
}
func DesktopPickFileBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopPickFile expects app and options")
		}
		app, e := desktopAppArg(args[0], "desktopPickFile")
		if e != nil {
			return e
		}
		options, e := desktopOptions(args[1], "desktopPickFile")
		if e != nil {
			return e
		}
		values, err := app.Backend.PickFile(options)
		if err != nil {
			return desktopError("desktopPickFile", err)
		}
		return desktopStrings(values)
	}}
}
func DesktopPickFolderBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopPickFolder expects app and options")
		}
		app, e := desktopAppArg(args[0], "desktopPickFolder")
		if e != nil {
			return e
		}
		options, e := desktopOptions(args[1], "desktopPickFolder")
		if e != nil {
			return e
		}
		value, err := app.Backend.PickFolder(options)
		if err != nil {
			return desktopError("desktopPickFolder", err)
		}
		if value == "" {
			return &object.Null{}
		}
		return NewString(value)
	}}
}
func DesktopNotifyBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopNotify expects app and options")
		}
		app, e := desktopAppArg(args[0], "desktopNotify")
		if e != nil {
			return e
		}
		options, e := desktopOptions(args[1], "desktopNotify")
		if e != nil {
			return e
		}
		if err := app.Backend.Notify(options); err != nil {
			return desktopError("desktopNotify", err)
		}
		return NewBoolean(true)
	}}
}
func DesktopPathsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopPaths expects app")
		}
		app, e := desktopAppArg(args[0], "desktopPaths")
		if e != nil {
			return e
		}
		values := map[string]object.Object{}
		for key, value := range app.Backend.Paths() {
			values[key] = NewString(value)
		}
		return objectMapDict(values)
	}}
}
func DesktopOpenExternalBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopOpenExternal expects app and target")
		}
		app, e := desktopAppArg(args[0], "desktopOpenExternal")
		if e != nil {
			return e
		}
		target, e := desktopString(args[1], "desktopOpenExternal target")
		if e != nil {
			return e
		}
		if err := app.Backend.OpenExternal(target); err != nil {
			return desktopError("desktopOpenExternal", err)
		}
		return NewBoolean(true)
	}}
}

func DesktopTrayBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopTray expects app and options")
		}
		app, e := desktopAppArg(args[0], "desktopTray")
		if e != nil {
			return e
		}
		options, e := desktopOptions(args[1], "desktopTray")
		if e != nil {
			return e
		}
		runtimeTray, err := app.Backend.CreateTray(options)
		if err != nil {
			return desktopError("desktopTray", err)
		}
		tray := &object.DesktopTray{App: app, Runtime: runtimeTray, Handlers: map[string]object.Object{}}
		app.Mu.Lock()
		app.Trays = append(app.Trays, tray)
		app.Mu.Unlock()
		return tray
	}}
}
func DesktopTrayAddBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("desktopTrayAdd expects tray, id, label and handler")
		}
		tray, e := desktopTrayArg(args[0], "desktopTrayAdd")
		if e != nil {
			return e
		}
		id, e := desktopString(args[1], "desktopTrayAdd id")
		if e != nil {
			return e
		}
		label, e := desktopString(args[2], "desktopTrayAdd label")
		if e != nil {
			return e
		}
		if err := tray.Runtime.Add(id, label); err != nil {
			return desktopError("desktopTrayAdd", err)
		}
		tray.Mu.Lock()
		tray.Handlers[id] = args[3]
		tray.Mu.Unlock()
		return tray
	}}
}
func DesktopTrayTooltipBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopTrayTooltip expects tray and text")
		}
		tray, e := desktopTrayArg(args[0], "desktopTrayTooltip")
		if e != nil {
			return e
		}
		text, e := desktopString(args[1], "desktopTrayTooltip text")
		if e != nil {
			return e
		}
		if err := tray.Runtime.SetTooltip(text); err != nil {
			return desktopError("desktopTrayTooltip", err)
		}
		return tray
	}}
}
func DesktopTrayCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopTrayClose expects tray")
		}
		tray, e := desktopTrayArg(args[0], "desktopTrayClose")
		if e != nil {
			return e
		}
		if err := tray.Runtime.Close(); err != nil {
			return desktopError("desktopTrayClose", err)
		}
		return NewBoolean(true)
	}}
}
func DesktopTrayOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopTrayOpen expects tray")
		}
		tray, e := desktopTrayArg(args[0], "desktopTrayOpen")
		if e != nil {
			return e
		}
		return NewBoolean(tray.Runtime.IsOpen())
	}}
}

func DesktopSpawnBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopSpawn expects command and options")
		}
		command, e := desktopString(args[0], "desktopSpawn command")
		if e != nil {
			return e
		}
		options, e := desktopOptions(args[1], "desktopSpawn")
		if e != nil {
			return e
		}
		arguments := []string{}
		if raw, ok := options["args"]; ok {
			arguments, e = parseProcessArgs(raw)
			if e != nil {
				return e
			}
		}
		process, err := startDesktopProcess(command, arguments, options)
		if err != nil {
			return desktopError("desktopSpawn", err)
		}
		return &object.DesktopProcess{Runtime: process}
	}}
}
func DesktopProcessWaitBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopProcessWait expects process")
		}
		process, e := desktopProcessArg(args[0], "desktopProcessWait")
		if e != nil {
			return e
		}
		code, err := process.Runtime.Wait()
		if err != nil {
			return desktopError("desktopProcessWait", err)
		}
		return NewInteger(code)
	}}
}
func DesktopProcessKillBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopProcessKill expects process")
		}
		process, e := desktopProcessArg(args[0], "desktopProcessKill")
		if e != nil {
			return e
		}
		if err := process.Runtime.Kill(); err != nil {
			return desktopError("desktopProcessKill", err)
		}
		return NewBoolean(true)
	}}
}
func DesktopProcessRunningBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopProcessRunning expects process")
		}
		process, e := desktopProcessArg(args[0], "desktopProcessRunning")
		if e != nil {
			return e
		}
		return NewBoolean(process.Runtime.Running())
	}}
}
func DesktopProcessIDBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopProcessId expects process")
		}
		process, e := desktopProcessArg(args[0], "desktopProcessId")
		if e != nil {
			return e
		}
		return NewInteger(process.Runtime.PID())
	}}
}

func DesktopWindowShowBuiltin() *object.Builtin {
	return desktopWindowAction("desktopWindowShow", func(w object.DesktopWindowRuntime) error { return w.Show() })
}
func DesktopWindowHideBuiltin() *object.Builtin {
	return desktopWindowAction("desktopWindowHide", func(w object.DesktopWindowRuntime) error { return w.Hide() })
}
func DesktopWindowCloseBuiltin() *object.Builtin {
	return desktopWindowAction("desktopWindowClose", func(w object.DesktopWindowRuntime) error { return w.Close() })
}
func DesktopWindowMaximizeBuiltin() *object.Builtin {
	return desktopWindowAction("desktopWindowMaximize", func(w object.DesktopWindowRuntime) error { return w.Maximize() })
}
func DesktopWindowMinimizeBuiltin() *object.Builtin {
	return desktopWindowAction("desktopWindowMinimize", func(w object.DesktopWindowRuntime) error { return w.Minimize() })
}
func DesktopWindowRestoreBuiltin() *object.Builtin {
	return desktopWindowAction("desktopWindowRestore", func(w object.DesktopWindowRuntime) error { return w.Restore() })
}
func DesktopWindowFocusBuiltin() *object.Builtin {
	return desktopWindowAction("desktopWindowFocus", func(w object.DesktopWindowRuntime) error { return w.Focus() })
}
func desktopWindowAction(name string, action func(object.DesktopWindowRuntime) error) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects window", name)
		}
		window, e := desktopWindowArg(args[0], name)
		if e != nil {
			return e
		}
		if err := action(window.Runtime); err != nil {
			return desktopError(name, err)
		}
		return window
	}}
}
func DesktopWindowOpenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowOpen expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowOpen")
		if e != nil {
			return e
		}
		return NewBoolean(w.Runtime.IsOpen())
	}}
}
func DesktopWindowIDBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowId expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowId")
		if e != nil {
			return e
		}
		return NewInteger(w.Runtime.ID())
	}}
}
func DesktopWindowTitleBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowTitle expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowTitle")
		if e != nil {
			return e
		}
		return NewString(w.Runtime.Title())
	}}
}
func DesktopWindowSetTitleBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopWindowSetTitle expects window and title")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowSetTitle")
		if e != nil {
			return e
		}
		v, e := desktopString(args[1], "desktopWindowSetTitle title")
		if e != nil {
			return e
		}
		if err := w.Runtime.SetTitle(v); err != nil {
			return desktopError("desktopWindowSetTitle", err)
		}
		return w
	}}
}
func DesktopWindowSizeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowSize expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowSize")
		if e != nil {
			return e
		}
		a, b := w.Runtime.Size()
		return desktopPair(a, b, "width", "height")
	}}
}
func DesktopWindowPixelSizeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowPixelSize expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowPixelSize")
		if e != nil {
			return e
		}
		a, b := w.Runtime.PixelSize()
		return desktopPair(a, b, "width", "height")
	}}
}
func DesktopWindowSetSizeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("desktopWindowSetSize expects window, width and height")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowSetSize")
		if e != nil {
			return e
		}
		a, e := desktopInt(args[1], "desktopWindowSetSize width")
		if e != nil {
			return e
		}
		b, e := desktopInt(args[2], "desktopWindowSetSize height")
		if e != nil {
			return e
		}
		if err := w.Runtime.SetSize(a, b); err != nil {
			return desktopError("desktopWindowSetSize", err)
		}
		return w
	}}
}
func DesktopWindowPositionBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowPosition expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowPosition")
		if e != nil {
			return e
		}
		a, b := w.Runtime.Position()
		return desktopPair(a, b, "x", "y")
	}}
}
func DesktopWindowSetPositionBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("desktopWindowSetPosition expects window, x and y")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowSetPosition")
		if e != nil {
			return e
		}
		a, e := desktopInt(args[1], "desktopWindowSetPosition x")
		if e != nil {
			return e
		}
		b, e := desktopInt(args[2], "desktopWindowSetPosition y")
		if e != nil {
			return e
		}
		if err := w.Runtime.SetPosition(a, b); err != nil {
			return desktopError("desktopWindowSetPosition", err)
		}
		return w
	}}
}
func DesktopWindowFullscreenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowFullscreen expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowFullscreen")
		if e != nil {
			return e
		}
		return NewBoolean(w.Runtime.Fullscreen())
	}}
}
func DesktopWindowSetFullscreenBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopWindowSetFullscreen expects window and bool")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowSetFullscreen")
		if e != nil {
			return e
		}
		v, e := desktopBool(args[1], "desktopWindowSetFullscreen value")
		if e != nil {
			return e
		}
		if err := w.Runtime.SetFullscreen(v); err != nil {
			return desktopError("desktopWindowSetFullscreen", err)
		}
		return w
	}}
}
func DesktopWindowDisplayScaleBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowDisplayScale expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowDisplayScale")
		if e != nil {
			return e
		}
		return NewFloat(w.Runtime.DisplayScale())
	}}
}
func DesktopWindowPixelDensityBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopWindowPixelDensity expects window")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowPixelDensity")
		if e != nil {
			return e
		}
		return NewFloat(w.Runtime.PixelDensity())
	}}
}
func DesktopWindowSetIconBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopWindowSetIcon expects window and path")
		}
		w, e := desktopWindowArg(args[0], "desktopWindowSetIcon")
		if e != nil {
			return e
		}
		v, e := desktopString(args[1], "desktopWindowSetIcon path")
		if e != nil {
			return e
		}
		if err := w.Runtime.SetIcon(v); err != nil {
			return desktopError("desktopWindowSetIcon", err)
		}
		return w
	}}
}

func DesktopAppMethod(app *object.DesktopApp, property string) object.Object {
	methods := map[string]*object.Builtin{
		"backend": DesktopBackendBuiltin(), "window": DesktopWindowBuiltin(), "on": DesktopOnBuiltin(),
		"shortcut": DesktopShortcutBuiltin(), "poll": DesktopPollBuiltin(), "run": DesktopRunBuiltin(),
		"quit": DesktopQuitBuiltin(), "running": DesktopRunningBuiltin(), "close": DesktopCloseBuiltin(),
		"emit": DesktopEmitBuiltin(), "setClipboard": DesktopSetClipboardBuiltin(), "clipboard": DesktopClipboardBuiltin(),
		"pickFile": DesktopPickFileBuiltin(), "pickFolder": DesktopPickFolderBuiltin(), "notify": DesktopNotifyBuiltin(),
		"paths": DesktopPathsBuiltin(), "openExternal": DesktopOpenExternalBuiltin(), "tray": DesktopTrayBuiltin(),
	}
	if builtin := methods[property]; builtin != nil {
		return bindDesktopBuiltin(builtin, app)
	}
	return nil
}
func DesktopWindowMethod(window *object.DesktopWindow, property string) object.Object {
	methods := map[string]*object.Builtin{
		"show": DesktopWindowShowBuiltin(), "hide": DesktopWindowHideBuiltin(), "close": DesktopWindowCloseBuiltin(), "isOpen": DesktopWindowOpenBuiltin(), "id": DesktopWindowIDBuiltin(),
		"title": DesktopWindowTitleBuiltin(), "setTitle": DesktopWindowSetTitleBuiltin(), "size": DesktopWindowSizeBuiltin(), "pixelSize": DesktopWindowPixelSizeBuiltin(), "setSize": DesktopWindowSetSizeBuiltin(),
		"position": DesktopWindowPositionBuiltin(), "setPosition": DesktopWindowSetPositionBuiltin(), "fullscreen": DesktopWindowFullscreenBuiltin(), "setFullscreen": DesktopWindowSetFullscreenBuiltin(),
		"maximize": DesktopWindowMaximizeBuiltin(), "minimize": DesktopWindowMinimizeBuiltin(), "restore": DesktopWindowRestoreBuiltin(), "focus": DesktopWindowFocusBuiltin(),
		"displayScale": DesktopWindowDisplayScaleBuiltin(), "pixelDensity": DesktopWindowPixelDensityBuiltin(), "setIcon": DesktopWindowSetIconBuiltin(),
	}
	if builtin := methods[property]; builtin != nil {
		return bindDesktopBuiltin(builtin, window)
	}
	return nil
}
func DesktopTrayMethod(tray *object.DesktopTray, property string) object.Object {
	methods := map[string]*object.Builtin{"add": DesktopTrayAddBuiltin(), "setTooltip": DesktopTrayTooltipBuiltin(), "close": DesktopTrayCloseBuiltin(), "isOpen": DesktopTrayOpenBuiltin()}
	if builtin := methods[property]; builtin != nil {
		return bindDesktopBuiltin(builtin, tray)
	}
	return nil
}
func DesktopProcessMethod(process *object.DesktopProcess, property string) object.Object {
	methods := map[string]*object.Builtin{"wait": DesktopProcessWaitBuiltin(), "kill": DesktopProcessKillBuiltin(), "running": DesktopProcessRunningBuiltin(), "id": DesktopProcessIDBuiltin()}
	if builtin := methods[property]; builtin != nil {
		return bindDesktopBuiltin(builtin, process)
	}
	return nil
}
func bindDesktopBuiltin(builtin *object.Builtin, receiver object.Object) object.Object {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		withReceiver := make([]object.Object, 0, len(args)+1)
		withReceiver = append(withReceiver, receiver)
		withReceiver = append(withReceiver, args...)
		return builtin.Fn(withReceiver...)
	}}
}
