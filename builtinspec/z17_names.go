package builtinspec

func init() {
	Names = append(Names,
		"alloc", "calloc", "nullPointer", "realloc", "free",
		"addressOf", "pointerFromAddress", "dereference", "pointerRead", "pointerWrite", "pointerOffset",
		"pointerLength", "pointerByteLength", "pointerType", "pointerAddress", "pointerEqual", "pointerCompare", "pointerIsAligned",
		"pointerIsNull", "pointerIsValid", "pointerOwned", "pointerBorrowed", "pointerMutable",
		"borrowPointer", "borrowPointerMut", "releaseBorrow", "movePointer", "pointerCopy", "pointerFill",
		"sizeOfType", "alignOfType", "byteSizeOf", "nativeStructLayout",
		"arenaCreate", "arenaAlloc", "arenaReset", "arenaFree", "arenaStats",
		"memoryStats", "memoryLeaks", "memoryValidate", "memoryResetStats",
		"mmapOpen", "mmapPointer", "mmapFlush", "mmapClose", "mmapSize",
		"sharedMemoryOpen", "sharedMemoryPointer", "sharedMemoryClose", "sharedMemoryUnlink",
		"volatileRead", "volatileWrite", "memoryFence",
		"atomicPointerLoad", "atomicPointerStore", "atomicPointerAdd", "atomicPointerSwap", "atomicPointerCompareSwap",
		"memoryProtect", "memoryLock", "memoryUnlock",
		"dynamicOpen", "dynamicSymbol", "dynamicCall", "dynamicClose", "dynamicIsOpen", "dynamicError",
		"systemInfo", "pageSize", "cpuCount", "rawSyscall", "profileNowNs", "profileElapsedNs",
	)
}
