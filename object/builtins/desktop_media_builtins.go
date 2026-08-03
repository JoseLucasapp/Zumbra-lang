package builtins

import "zumbra/object"

func desktopByteArray(value object.Object, name string) ([]byte, *object.Error) {
	buffer, ok := value.(*object.ByteArray)
	if !ok {
		return nil, NewError("%s expects ByteArray, got %s", name, value.Type())
	}
	return buffer.Data, nil
}

func desktopMediaBackendArg(app *object.DesktopApp, name string) (object.DesktopMediaBackend, *object.Error) {
	backend, ok := app.Backend.(object.DesktopMediaBackend)
	if !ok {
		return nil, NewError("%s is not supported by desktop backend %s", name, app.Backend.Name())
	}
	return backend, nil
}

func desktopFramebufferArg(window *object.DesktopWindow, name string) (object.DesktopFramebufferRuntime, *object.Error) {
	runtime, ok := window.Runtime.(object.DesktopFramebufferRuntime)
	if !ok {
		return nil, NewError("%s is not supported by this desktop window", name)
	}
	return runtime, nil
}

func DesktopWindowPresentRGBABuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("desktopWindowPresentRGBA expects window, pixels, width and height")
		}
		window, e := desktopWindowArg(args[0], "desktopWindowPresentRGBA")
		if e != nil {
			return e
		}
		pixels, e := desktopByteArray(args[1], "desktopWindowPresentRGBA pixels")
		if e != nil {
			return e
		}
		width, e := desktopInt(args[2], "desktopWindowPresentRGBA width")
		if e != nil {
			return e
		}
		height, e := desktopInt(args[3], "desktopWindowPresentRGBA height")
		if e != nil {
			return e
		}
		if width < 1 || height < 1 || int64(len(pixels)) != width*height*4 {
			return NewError("desktopWindowPresentRGBA expects exactly width*height*4 bytes")
		}
		framebuffer, e := desktopFramebufferArg(window, "desktopWindowPresentRGBA")
		if e != nil {
			return e
		}
		if err := framebuffer.PresentRGBA(pixels, width, height); err != nil {
			return desktopError("desktopWindowPresentRGBA", err)
		}
		return NewBoolean(true)
	}}
}

func DesktopWindowSetVSyncBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopWindowSetVSync expects window and enabled")
		}
		window, e := desktopWindowArg(args[0], "desktopWindowSetVSync")
		if e != nil {
			return e
		}
		enabled, e := desktopBool(args[1], "desktopWindowSetVSync enabled")
		if e != nil {
			return e
		}
		framebuffer, e := desktopFramebufferArg(window, "desktopWindowSetVSync")
		if e != nil {
			return e
		}
		if err := framebuffer.SetVSync(enabled); err != nil {
			return desktopError("desktopWindowSetVSync", err)
		}
		return window
	}}
}

func DesktopKeyDownBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("desktopKeyDown expects app and scancode")
		}
		app, e := desktopAppArg(args[0], "desktopKeyDown")
		if e != nil {
			return e
		}
		scancode, e := desktopInt(args[1], "desktopKeyDown scancode")
		if e != nil {
			return e
		}
		backend, e := desktopMediaBackendArg(app, "desktopKeyDown")
		if e != nil {
			return e
		}
		return NewBoolean(backend.KeyDown(scancode))
	}}
}

func DesktopGamepadButtonBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("desktopGamepadButton expects app, player and button")
		}
		app, e := desktopAppArg(args[0], "desktopGamepadButton")
		if e != nil {
			return e
		}
		player, e := desktopInt(args[1], "desktopGamepadButton player")
		if e != nil {
			return e
		}
		button, e := desktopInt(args[2], "desktopGamepadButton button")
		if e != nil {
			return e
		}
		backend, e := desktopMediaBackendArg(app, "desktopGamepadButton")
		if e != nil {
			return e
		}
		return NewBoolean(backend.GamepadButton(player, button))
	}}
}

func DesktopAudioQueueBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("desktopAudioQueue expects app, samples, volume and muted")
		}
		app, e := desktopAppArg(args[0], "desktopAudioQueue")
		if e != nil {
			return e
		}
		samples, e := desktopByteArray(args[1], "desktopAudioQueue samples")
		if e != nil {
			return e
		}
		volume, e := desktopInt(args[2], "desktopAudioQueue volume")
		if e != nil {
			return e
		}
		muted, e := desktopBool(args[3], "desktopAudioQueue muted")
		if e != nil {
			return e
		}
		backend, e := desktopMediaBackendArg(app, "desktopAudioQueue")
		if e != nil {
			return e
		}
		queued, err := backend.QueueAudio(samples, volume, muted)
		if err != nil {
			return desktopError("desktopAudioQueue", err)
		}
		return NewInteger(queued)
	}}
}

func DesktopAudioQueuedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("desktopAudioQueued expects app")
		}
		app, e := desktopAppArg(args[0], "desktopAudioQueued")
		if e != nil {
			return e
		}
		backend, e := desktopMediaBackendArg(app, "desktopAudioQueued")
		if e != nil {
			return e
		}
		return NewInteger(backend.AudioQueued())
	}}
}
