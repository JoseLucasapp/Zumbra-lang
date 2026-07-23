package types

func builtinType(name string) (*Type, bool) {
	switch name {
	case "show":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Null)), true

	case "input":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(String)), true

	case "sizeOf":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Int)), true

	case "toInt":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Int)), true

	case "toFloat":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Float)), true

	case "toString":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(String)), true

	case "toBool":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Bool)), true

	case "u8":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(U8)), true
	case "u16":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(U16)), true
	case "u32":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(U32)), true
	case "u64":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(U64)), true
	case "i8":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(I8)), true
	case "i16":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(I16)), true
	case "i32":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(I32)), true
	case "i64":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(I64)), true

	case "wrapAdd", "wrapSub", "wrapMul",
		"checkedAdd", "checkedSub", "checkedMul",
		"satAdd", "satSub", "satMul":
		return FuncOf([]*Type{Simple(Unknown), Simple(Unknown)}, Simple(Unknown)), true

	case "bytes":
		return FuncOf([]*Type{Simple(Int)}, ByteArrayOf()), true

	case "arrayOf":
		return FuncOf([]*Type{Simple(String), Simple(Int)}, TypedArrayOf(Simple(Unknown))), true

	case "slice":
		return FuncOf([]*Type{Simple(Unknown), Simple(Int), Simple(Int)}, SliceOf(Simple(Unknown))), true

	case "fill":
		return FuncOf([]*Type{Simple(Unknown), Simple(Unknown)}, Simple(Unknown)), true

	case "readBytes":
		return FuncOf([]*Type{Simple(String)}, ByteArrayOf()), true
	case "writeBytes":
		return FuncOf([]*Type{Simple(String), Simple(Unknown)}, Simple(Int)), true
	case "readU16LE", "readU16BE":
		return FuncOf([]*Type{Simple(Unknown), Simple(Int)}, Simple(U16)), true
	case "readU32LE", "readU32BE":
		return FuncOf([]*Type{Simple(Unknown), Simple(Int)}, Simple(U32)), true
	case "readU64LE", "readU64BE":
		return FuncOf([]*Type{Simple(Unknown), Simple(Int)}, Simple(U64)), true
	case "writeU16LE", "writeU16BE", "writeU32LE", "writeU32BE", "writeU64LE", "writeU64BE":
		return FuncOf([]*Type{Simple(Unknown), Simple(Int), Simple(Unknown)}, Simple(Unknown)), true
	case "copyBytes":
		return FuncOf([]*Type{Simple(Unknown), Simple(Int), Simple(Unknown), Simple(Int), Simple(Int)}, Simple(Unknown)), true
	case "bytesEqual":
		return FuncOf([]*Type{Simple(Unknown), Simple(Unknown)}, Simple(Bool)), true
	case "sha256":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(String)), true

	case "join":
		return FuncOf([]*Type{TaskOf(Simple(Unknown))}, Simple(Unknown)), true
	case "cancel", "taskDone", "taskCancelled":
		return FuncOf([]*Type{TaskOf(Simple(Unknown))}, Simple(Bool)), true
	case "joinTimeout":
		return FuncOf([]*Type{TaskOf(Simple(Unknown)), Simple(Int)}, ArrayOf(Simple(Unknown))), true
	case "sleepMs":
		return FuncOf([]*Type{Simple(Int)}, Simple(Null)), true
	case "channel":
		return FuncOf([]*Type{Simple(Int)}, ChannelOf(Simple(Unknown))), true
	case "send":
		return FuncOf([]*Type{ChannelOf(Simple(Unknown)), Simple(Unknown)}, Simple(Null)), true
	case "receive":
		return FuncOf([]*Type{ChannelOf(Simple(Unknown))}, Simple(Unknown)), true
	case "receiveOk", "receiveTimeout":
		return FuncOf([]*Type{ChannelOf(Simple(Unknown))}, ArrayOf(Simple(Unknown))), true
	case "closeChannel", "channelClosed":
		return FuncOf([]*Type{ChannelOf(Simple(Unknown))}, Simple(Bool)), true
	case "channelLen", "channelCap":
		return FuncOf([]*Type{ChannelOf(Simple(Unknown))}, Simple(Int)), true
	case "mutex":
		return FuncOf(nil, Simple(Mutex)), true
	case "rwMutex":
		return FuncOf(nil, Simple(RWMutex)), true
	case "waitGroup":
		return FuncOf(nil, Simple(WaitGroup)), true
	case "semaphore":
		return FuncOf([]*Type{Simple(Int)}, Simple(Semaphore)), true
	case "atomicInt":
		return FuncOf([]*Type{Simple(Int)}, Simple(AtomicInt)), true
	case "atomicLoad":
		return FuncOf([]*Type{Simple(AtomicInt)}, Simple(Int)), true
	case "atomicStore":
		return FuncOf([]*Type{Simple(AtomicInt), Simple(Int)}, Simple(Null)), true
	case "atomicAdd", "atomicSwap":
		return FuncOf([]*Type{Simple(AtomicInt), Simple(Int)}, Simple(Int)), true
	case "atomicCompareSwap":
		return FuncOf([]*Type{Simple(AtomicInt), Simple(Int), Simple(Int)}, Simple(Bool)), true
	case "lock", "unlock", "rLock", "rUnlock", "wgAdd", "wgDone", "wgWait", "acquire", "release":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Null)), true

	case "first":
		return FuncOf([]*Type{ArrayOf(Simple(Unknown))}, Simple(Unknown)), true

	case "last":
		return FuncOf([]*Type{ArrayOf(Simple(Unknown))}, Simple(Unknown)), true

	default:
		return nil, false
	}
}
