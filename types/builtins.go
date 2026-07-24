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

	case "tcpListen":
		return FuncOf([]*Type{Simple(String), Simple(Int)}, Simple(NetListener)), true
	case "tcpConnect":
		return FuncOf([]*Type{Simple(String), Simple(Int)}, Simple(NetStream)), true
	case "tcpConnectTimeout":
		return FuncOf([]*Type{Simple(String), Simple(Int), Simple(Int)}, Simple(NetStream)), true
	case "tlsListen":
		return FuncOf([]*Type{Simple(String), Simple(Int), Simple(String), Simple(String)}, Simple(NetListener)), true
	case "tlsConnect":
		return FuncOf([]*Type{Simple(String), Simple(Int), Simple(String), Simple(Bool)}, Simple(NetStream)), true
	case "tlsConnectTimeout":
		return FuncOf([]*Type{Simple(String), Simple(Int), Simple(String), Simple(Bool), Simple(Int)}, Simple(NetStream)), true
	case "listenerAccept":
		return FuncOf([]*Type{Simple(NetListener)}, Simple(NetStream)), true
	case "listenerAcceptTimeout":
		return FuncOf([]*Type{Simple(NetListener), Simple(Int)}, ArrayOf(Simple(Unknown))), true
	case "listenerClose", "listenerClosed":
		return FuncOf([]*Type{Simple(NetListener)}, Simple(Bool)), true
	case "listenerAddress":
		return FuncOf([]*Type{Simple(NetListener)}, Simple(String)), true
	case "listenerPort":
		return FuncOf([]*Type{Simple(NetListener)}, Simple(Int)), true
	case "streamRead", "streamReadExact":
		return FuncOf([]*Type{Simple(NetStream), Simple(Int)}, ByteArrayOf()), true
	case "streamReadTimeout":
		return FuncOf([]*Type{Simple(NetStream), Simple(Int), Simple(Int)}, ArrayOf(Simple(Unknown))), true
	case "streamWrite", "streamWriteAll":
		return FuncOf([]*Type{Simple(NetStream), Simple(Unknown)}, Simple(Int)), true
	case "streamClose", "streamClosed", "streamShutdownRead", "streamShutdownWrite":
		return FuncOf([]*Type{Simple(NetStream)}, Simple(Bool)), true
	case "streamLocalAddress", "streamRemoteAddress":
		return FuncOf([]*Type{Simple(NetStream)}, Simple(String)), true
	case "streamLocalPort", "streamRemotePort":
		return FuncOf([]*Type{Simple(NetStream)}, Simple(Int)), true
	case "streamSetReadTimeout", "streamSetWriteTimeout":
		return FuncOf([]*Type{Simple(NetStream), Simple(Int)}, Simple(Null)), true
	case "tcpSetKeepAlive":
		return FuncOf([]*Type{Simple(NetStream), Simple(Bool), Simple(Int)}, Simple(Null)), true
	case "dnsLookup":
		return FuncOf([]*Type{Simple(String)}, ArrayOf(Simple(String))), true
	case "dnsLookupTimeout":
		return FuncOf([]*Type{Simple(String), Simple(Int)}, ArrayOf(Simple(Unknown))), true
	case "udpBind":
		return FuncOf([]*Type{Simple(String), Simple(Int)}, Simple(UDPSocket)), true
	case "udpSendTo":
		return FuncOf([]*Type{Simple(UDPSocket), Simple(String), Simple(Int), Simple(Unknown)}, Simple(Int)), true
	case "udpReceiveFrom":
		return FuncOf([]*Type{Simple(UDPSocket), Simple(Int)}, ArrayOf(Simple(Unknown))), true
	case "udpReceiveFromTimeout":
		return FuncOf([]*Type{Simple(UDPSocket), Simple(Int), Simple(Int)}, ArrayOf(Simple(Unknown))), true
	case "udpClose", "udpClosed":
		return FuncOf([]*Type{Simple(UDPSocket)}, Simple(Bool)), true
	case "udpAddress":
		return FuncOf([]*Type{Simple(UDPSocket)}, Simple(String)), true
	case "udpPort":
		return FuncOf([]*Type{Simple(UDPSocket)}, Simple(Int)), true

	case "httpApp":
		return FuncOf(nil, Simple(HttpApp)), true
	case "httpRoute":
		handler := FuncOf([]*Type{Simple(HttpRequest), Simple(HttpResponse)}, Simple(Unknown))
		return FuncOf([]*Type{Simple(HttpApp), Simple(String), Simple(String), handler}, Simple(HttpApp)), true
	case "httpUse":
		middleware := FuncOf([]*Type{Simple(HttpRequest), Simple(HttpResponse)}, Simple(Unknown))
		return FuncOf([]*Type{Simple(HttpApp), middleware}, Simple(HttpApp)), true
	case "httpStatic":
		return FuncOf([]*Type{Simple(HttpApp), Simple(String), Simple(String)}, Simple(HttpApp)), true
	case "httpLimitBody":
		return FuncOf([]*Type{Simple(HttpApp), Simple(Int)}, Simple(HttpApp)), true
	case "httpCompression":
		return FuncOf([]*Type{Simple(HttpApp), Simple(Bool)}, Simple(HttpApp)), true
	case "httpCors":
		return FuncOf([]*Type{Simple(HttpApp), ArrayOf(Simple(String)), ArrayOf(Simple(String)), ArrayOf(Simple(String)), Simple(Bool), Simple(Int)}, Simple(HttpApp)), true
	case "httpServe":
		return FuncOf([]*Type{Simple(HttpApp), Simple(String), Simple(Int)}, Simple(HttpServer)), true
	case "httpServeTLS":
		return FuncOf([]*Type{Simple(HttpApp), Simple(String), Simple(Int), Simple(String), Simple(String)}, Simple(HttpServer)), true
	case "httpShutdown":
		return FuncOf([]*Type{Simple(HttpServer), Simple(Int)}, Simple(Bool)), true
	case "httpServerPort":
		return FuncOf([]*Type{Simple(HttpServer)}, Simple(Int)), true
	case "httpServerAddress":
		return FuncOf([]*Type{Simple(HttpServer)}, Simple(String)), true
	case "httpServerRunning":
		return FuncOf([]*Type{Simple(HttpServer)}, Simple(Bool)), true
	case "httpText", "httpJson", "httpHtml", "httpRedirect", "httpFile":
		return FuncOf([]*Type{Simple(Int), Simple(Unknown)}, Simple(HttpResponse)), true
	case "httpHeader":
		return FuncOf([]*Type{Simple(HttpResponse), Simple(String), Simple(String)}, Simple(HttpResponse)), true
	case "httpCookie":
		return FuncOf([]*Type{Simple(HttpResponse), Simple(String), Simple(String), DictOf(Simple(String), Simple(Unknown))}, Simple(HttpResponse)), true
	case "httpStream":
		return FuncOf([]*Type{Simple(Int), Simple(String), ChannelOf(Simple(Unknown))}, Simple(HttpResponse)), true
	case "httpSSE":
		return FuncOf([]*Type{Simple(Int), ChannelOf(Simple(String))}, Simple(HttpResponse)), true
	case "sseEvent":
		return FuncOf([]*Type{Simple(String), Simple(String), Simple(String), Simple(Int)}, Simple(String)), true
	case "httpRequest":
		return FuncOf([]*Type{Simple(String), Simple(String), DictOf(Simple(String), Simple(String)), Simple(Unknown), Simple(Int)}, Simple(HttpClientResponse)), true
	case "httpStatus":
		return FuncOf([]*Type{Simple(HttpClientResponse)}, Simple(Int)), true
	case "httpBody":
		return FuncOf([]*Type{Simple(HttpClientResponse)}, Simple(String)), true
	case "httpBodyBytes":
		return FuncOf([]*Type{Simple(HttpClientResponse)}, ByteArrayOf()), true
	case "httpBodyJSON":
		return FuncOf([]*Type{Simple(HttpClientResponse)}, Simple(Unknown)), true
	case "httpHeaders":
		return FuncOf([]*Type{Simple(HttpClientResponse)}, DictOf(Simple(String), Simple(String))), true
	case "jsonStringify":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(String)), true
	case "jsonParse":
		return FuncOf([]*Type{Simple(String)}, Simple(Unknown)), true
	case "jwtSignHS256":
		return FuncOf([]*Type{DictOf(Simple(String), Simple(Unknown)), Simple(String), Simple(Int)}, Simple(String)), true
	case "jwtVerifyHS256":
		return FuncOf([]*Type{Simple(String), Simple(String)}, ArrayOf(Simple(Unknown))), true
	case "webSocketUpgrade":
		return FuncOf([]*Type{Simple(HttpRequest)}, Simple(WebSocket)), true
	case "webSocketConnect":
		return FuncOf([]*Type{Simple(String), DictOf(Simple(String), Simple(String)), Simple(Int)}, Simple(WebSocket)), true
	case "webSocketRead":
		return FuncOf([]*Type{Simple(WebSocket)}, ArrayOf(Simple(Unknown))), true
	case "webSocketReadTimeout":
		return FuncOf([]*Type{Simple(WebSocket), Simple(Int)}, ArrayOf(Simple(Unknown))), true
	case "webSocketWriteText", "webSocketPing":
		return FuncOf([]*Type{Simple(WebSocket), Simple(String)}, Simple(Int)), true
	case "webSocketWriteBinary":
		return FuncOf([]*Type{Simple(WebSocket), Simple(Unknown)}, Simple(Int)), true
	case "webSocketClose":
		return FuncOf([]*Type{Simple(WebSocket), Simple(Int), Simple(String)}, Simple(Bool)), true
	case "webSocketClosed":
		return FuncOf([]*Type{Simple(WebSocket)}, Simple(Bool)), true

	case "first":
		return FuncOf([]*Type{ArrayOf(Simple(Unknown))}, Simple(Unknown)), true

	case "last":
		return FuncOf([]*Type{ArrayOf(Simple(Unknown))}, Simple(Unknown)), true

	default:
		return nil, false
	}
}
