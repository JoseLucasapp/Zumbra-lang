package builtinspec

func init() {
	Names = append(Names,
		"desktopApp", "desktopBackend", "desktopWindow", "desktopOn", "desktopShortcut",
		"desktopPoll", "desktopRun", "desktopQuit", "desktopRunning", "desktopClose", "desktopEmit",
		"desktopSetClipboard", "desktopClipboard", "desktopPickFile", "desktopPickFolder", "desktopNotify",
		"desktopPaths", "desktopOpenExternal", "desktopTray", "desktopTrayAdd", "desktopTrayTooltip",
		"desktopTrayClose", "desktopTrayOpen", "desktopSpawn", "desktopProcessWait", "desktopProcessKill",
		"desktopProcessRunning", "desktopProcessId", "desktopWindowShow", "desktopWindowHide",
		"desktopWindowClose", "desktopWindowOpen", "desktopWindowId", "desktopWindowTitle",
		"desktopWindowSetTitle", "desktopWindowSize", "desktopWindowPixelSize", "desktopWindowSetSize",
		"desktopWindowPosition", "desktopWindowSetPosition", "desktopWindowFullscreen",
		"desktopWindowSetFullscreen", "desktopWindowMaximize", "desktopWindowMinimize",
		"desktopWindowRestore", "desktopWindowFocus", "desktopWindowDisplayScale",
		"desktopWindowPixelDensity", "desktopWindowSetIcon",
		"desktopWindowPresentRGBA", "desktopWindowSetVSync", "desktopKeyDown",
		"desktopGamepadButton", "desktopAudioQueue", "desktopAudioQueued",
	)
}
