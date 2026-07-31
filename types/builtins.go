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

	// Z17 systems programming and explicit memory.
	case "alloc", "calloc":
		return FuncOf([]*Type{Simple(String), Simple(Int)}, PointerOf(Simple(Unknown))), true
	case "nullPointer":
		return FuncOf([]*Type{Simple(String)}, PointerOf(Simple(Unknown))), true
	case "realloc":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Int)}, PointerOf(Simple(Unknown))), true
	case "free":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, Simple(Bool)), true
	case "addressOf":
		return FuncOf([]*Type{Simple(Unknown), Simple(Int)}, PointerOf(Simple(Unknown))), true
	case "pointerFromAddress":
		return FuncOf([]*Type{Simple(String), Simple(U64), Simple(Int), Simple(Bool)}, PointerOf(Simple(Unknown))), true
	case "dereference", "pointerRead", "volatileRead":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Int)}, Simple(Unknown)), true
	case "pointerWrite", "volatileWrite":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Int), Simple(Unknown)}, PointerOf(Simple(Unknown))), true
	case "atomicPointerLoad":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, Simple(Unknown)), true
	case "atomicPointerStore":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Unknown)}, Simple(Null)), true
	case "atomicPointerSwap", "atomicPointerAdd":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Unknown)}, Simple(Unknown)), true
	case "atomicPointerCompareSwap":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Unknown), Simple(Unknown)}, Simple(Bool)), true
	case "pointerOffset":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Int)}, PointerOf(Simple(Unknown))), true
	case "pointerLength", "pointerByteLength":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, Simple(Int)), true
	case "pointerType":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, Simple(String)), true
	case "pointerAddress":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, Simple(U64)), true
	case "pointerEqual":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), PointerOf(Simple(Unknown))}, Simple(Bool)), true
	case "pointerCompare":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), PointerOf(Simple(Unknown))}, Simple(Int)), true
	case "pointerIsAligned":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Int)}, Simple(Bool)), true
	case "pointerIsNull", "pointerIsValid", "pointerOwned", "pointerBorrowed", "pointerMutable":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, Simple(Bool)), true
	case "borrowPointer", "borrowPointerMut", "movePointer":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, PointerOf(Simple(Unknown))), true
	case "releaseBorrow":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, Simple(Bool)), true
	case "pointerCopy":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Int), PointerOf(Simple(Unknown)), Simple(Int), Simple(Int)}, PointerOf(Simple(Unknown))), true
	case "pointerFill":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(Unknown)}, PointerOf(Simple(Unknown))), true
	case "sizeOfType", "alignOfType":
		return FuncOf([]*Type{Simple(String)}, Simple(Int)), true
	case "byteSizeOf":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Int)), true
	case "nativeStructLayout":
		return FuncOf([]*Type{ArrayOf(DictOf(Simple(String), Simple(Unknown)))}, DictOf(Simple(String), Simple(Unknown))), true
	case "arenaCreate":
		return FuncOf(nil, Simple(MemoryArena)), true
	case "arenaAlloc":
		return FuncOf([]*Type{Simple(MemoryArena), Simple(String), Simple(Int)}, PointerOf(Simple(Unknown))), true
	case "arenaReset", "arenaFree":
		return FuncOf([]*Type{Simple(MemoryArena)}, Simple(Bool)), true
	case "arenaStats":
		return FuncOf([]*Type{Simple(MemoryArena)}, DictOf(Simple(String), Simple(Unknown))), true
	case "memoryStats", "memoryValidate":
		return FuncOf(nil, DictOf(Simple(String), Simple(Unknown))), true
	case "memoryLeaks":
		return FuncOf(nil, ArrayOf(DictOf(Simple(String), Simple(Unknown)))), true
	case "memoryResetStats":
		return FuncOf(nil, Simple(Bool)), true
	case "mmapOpen":
		return FuncOf([]*Type{Simple(String), Simple(String), Simple(Int)}, Simple(MappedMemory)), true
	case "mmapPointer":
		return FuncOf([]*Type{Simple(MappedMemory)}, PointerOf(Simple(U8))), true
	case "mmapFlush", "mmapClose":
		return FuncOf([]*Type{Simple(MappedMemory)}, Simple(Bool)), true
	case "mmapSize":
		return FuncOf([]*Type{Simple(MappedMemory)}, Simple(Int)), true
	case "sharedMemoryOpen":
		return FuncOf([]*Type{Simple(String), Simple(Int), Simple(Bool)}, Simple(SharedMemory)), true
	case "sharedMemoryPointer":
		return FuncOf([]*Type{Simple(SharedMemory)}, PointerOf(Simple(U8))), true
	case "sharedMemoryClose", "sharedMemoryUnlink":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Bool)), true
	case "memoryFence":
		return FuncOf([]*Type{Simple(String)}, Simple(Null)), true
	case "memoryProtect":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(String)}, Simple(Bool)), true
	case "memoryLock", "memoryUnlock":
		return FuncOf([]*Type{PointerOf(Simple(Unknown))}, Simple(Bool)), true
	case "dynamicOpen":
		return FuncOf([]*Type{Simple(String)}, Simple(DynamicLibrary)), true
	case "dynamicSymbol":
		return FuncOf([]*Type{Simple(DynamicLibrary), Simple(String)}, PointerOf(Simple(Unknown))), true
	case "dynamicCall":
		return FuncOf([]*Type{PointerOf(Simple(Unknown)), Simple(String), ArrayOf(Simple(Unknown))}, Simple(Unknown)), true
	case "dynamicClose", "dynamicIsOpen":
		return FuncOf([]*Type{Simple(DynamicLibrary)}, Simple(Bool)), true
	case "dynamicError":
		return FuncOf([]*Type{Simple(DynamicLibrary)}, Simple(String)), true
	case "systemInfo":
		return FuncOf(nil, DictOf(Simple(String), Simple(Unknown))), true
	case "pageSize", "cpuCount":
		return FuncOf(nil, Simple(Int)), true
	case "rawSyscall":
		return FuncOf([]*Type{Simple(Int), ArrayOf(Simple(Int))}, DictOf(Simple(String), Simple(Unknown))), true
	case "profileNowNs":
		return FuncOf(nil, Simple(U64)), true
	case "profileElapsedNs":
		return FuncOf([]*Type{Simple(U64)}, Simple(U64)), true

	case "assetExists":
		return FuncOf([]*Type{Simple(String)}, Simple(Bool)), true
	case "assetText":
		return FuncOf([]*Type{Simple(String)}, Simple(String)), true
	case "assetBytes":
		return FuncOf([]*Type{Simple(String)}, ByteArrayOf()), true
	case "assetList":
		return FuncOf(nil, ArrayOf(Simple(String))), true

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

	case "sqliteOpen":
		return FuncOf([]*Type{Simple(String)}, Simple(SQLiteDatabase)), true
	case "sqliteMemory":
		return FuncOf(nil, Simple(SQLiteDatabase)), true
	case "sqliteExec":
		return FuncOf([]*Type{Simple(SQLiteDatabase), Simple(String), Simple(SQLParameters)}, DictOf(Simple(String), Simple(Int))), true
	case "sqliteQuery":
		return FuncOf([]*Type{Simple(SQLiteDatabase), Simple(String), Simple(SQLParameters)}, ArrayOf(DictOf(Simple(String), Simple(Unknown)))), true
	case "sqlitePrepare":
		return FuncOf([]*Type{Simple(SQLiteDatabase), Simple(String)}, Simple(SQLiteStatement)), true
	case "sqliteBegin":
		return FuncOf([]*Type{Simple(SQLiteDatabase)}, Simple(SQLiteTransaction)), true
	case "sqliteClose", "sqliteIsOpen":
		return FuncOf([]*Type{Simple(SQLiteDatabase)}, Simple(Bool)), true
	case "sqlitePath":
		return FuncOf([]*Type{Simple(SQLiteDatabase)}, Simple(String)), true
	case "sqliteStatementExec":
		return FuncOf([]*Type{Simple(SQLiteStatement), Simple(SQLParameters)}, DictOf(Simple(String), Simple(Int))), true
	case "sqliteStatementQuery":
		return FuncOf([]*Type{Simple(SQLiteStatement), Simple(SQLParameters)}, ArrayOf(DictOf(Simple(String), Simple(Unknown)))), true
	case "sqliteStatementClose", "sqliteStatementOpen":
		return FuncOf([]*Type{Simple(SQLiteStatement)}, Simple(Bool)), true
	case "sqliteStatementSQL":
		return FuncOf([]*Type{Simple(SQLiteStatement)}, Simple(String)), true
	case "sqliteTransactionExec":
		return FuncOf([]*Type{Simple(SQLiteTransaction), Simple(String), Simple(SQLParameters)}, DictOf(Simple(String), Simple(Int))), true
	case "sqliteTransactionQuery":
		return FuncOf([]*Type{Simple(SQLiteTransaction), Simple(String), Simple(SQLParameters)}, ArrayOf(DictOf(Simple(String), Simple(Unknown)))), true
	case "sqliteTransactionPrepare":
		return FuncOf([]*Type{Simple(SQLiteTransaction), Simple(String)}, Simple(SQLiteStatement)), true
	case "sqliteCommit", "sqliteRollback", "sqliteTransactionActive":
		return FuncOf([]*Type{Simple(SQLiteTransaction)}, Simple(Bool)), true

	case "sqliteQueryOne":
		return FuncOf([]*Type{Simple(SQLiteDatabase), Simple(String), Simple(SQLParameters)}, DictOf(Simple(String), Simple(Unknown))), true
	case "sqliteQueryStream":
		return FuncOf([]*Type{Simple(SQLiteDatabase), Simple(String), Simple(SQLParameters)}, Simple(SQLRows)), true
	case "sqliteMigrate":
		return FuncOf([]*Type{Simple(SQLiteDatabase), ArrayOf(DictOf(Simple(String), Simple(Unknown)))}, Simple(Int)), true
	case "sqliteSchemaVersion":
		return FuncOf([]*Type{Simple(SQLiteDatabase)}, Simple(Int)), true
	case "sqliteBackup", "sqliteRestore":
		return FuncOf([]*Type{Simple(SQLiteDatabase), Simple(String)}, DictOf(Simple(String), Simple(Unknown))), true
	case "sqliteIntegrityCheck":
		return FuncOf([]*Type{Simple(SQLiteDatabase)}, DictOf(Simple(String), Simple(Unknown))), true
	case "sqliteStatementQueryStream":
		return FuncOf([]*Type{Simple(SQLiteStatement), Simple(SQLParameters)}, Simple(SQLRows)), true
	case "sqliteStatementParameterCount":
		return FuncOf([]*Type{Simple(SQLiteStatement)}, Simple(Int)), true
	case "sqliteStatementColumns":
		return FuncOf([]*Type{Simple(SQLiteStatement)}, ArrayOf(Simple(String))), true
	case "sqliteTransactionQueryStream":
		return FuncOf([]*Type{Simple(SQLiteTransaction), Simple(String), Simple(SQLParameters)}, Simple(SQLRows)), true
	case "sqliteSavepoint", "sqliteRollbackTo", "sqliteRelease":
		return FuncOf([]*Type{Simple(SQLiteTransaction), Simple(String)}, Simple(Bool)), true
	case "sqlRowsNext":
		return FuncOf([]*Type{Simple(SQLRows)}, ArrayOf(Simple(Unknown))), true
	case "sqlRowsColumns":
		return FuncOf([]*Type{Simple(SQLRows)}, ArrayOf(Simple(String))), true
	case "sqlRowsClose", "sqlRowsOpen":
		return FuncOf([]*Type{Simple(SQLRows)}, Simple(Bool)), true

	case "postgresOpen":
		return FuncOf([]*Type{Simple(String), DictOf(Simple(String), Simple(Unknown))}, Simple(PostgresDatabase)), true
	case "postgresConfigurePool":
		return FuncOf([]*Type{Simple(PostgresDatabase), Simple(Int), Simple(Int), Simple(Int), Simple(Int)}, Simple(PostgresDatabase)), true
	case "postgresPoolStats":
		return FuncOf([]*Type{Simple(PostgresDatabase)}, DictOf(Simple(String), Simple(Int))), true
	case "postgresPing", "postgresClose", "postgresIsOpen":
		return FuncOf([]*Type{Simple(PostgresDatabase)}, Simple(Bool)), true
	case "postgresExecDb":
		return FuncOf([]*Type{Simple(PostgresDatabase), Simple(String), Simple(SQLParameters)}, DictOf(Simple(String), Simple(Int))), true
	case "postgresQueryDb":
		return FuncOf([]*Type{Simple(PostgresDatabase), Simple(String), Simple(SQLParameters)}, ArrayOf(DictOf(Simple(String), Simple(Unknown)))), true
	case "postgresQueryOne":
		return FuncOf([]*Type{Simple(PostgresDatabase), Simple(String), Simple(SQLParameters)}, Simple(Unknown)), true
	case "postgresQueryStream":
		return FuncOf([]*Type{Simple(PostgresDatabase), Simple(String), Simple(SQLParameters)}, Simple(SQLRows)), true
	case "postgresPrepare":
		return FuncOf([]*Type{Simple(PostgresDatabase), Simple(String)}, Simple(PostgresStatement)), true
	case "postgresBegin":
		return FuncOf([]*Type{Simple(PostgresDatabase)}, Simple(PostgresTransaction)), true
	case "postgresStatementExec":
		return FuncOf([]*Type{Simple(PostgresStatement), Simple(SQLParameters)}, DictOf(Simple(String), Simple(Int))), true
	case "postgresStatementQuery":
		return FuncOf([]*Type{Simple(PostgresStatement), Simple(SQLParameters)}, ArrayOf(DictOf(Simple(String), Simple(Unknown)))), true
	case "postgresStatementStream":
		return FuncOf([]*Type{Simple(PostgresStatement), Simple(SQLParameters)}, Simple(SQLRows)), true
	case "postgresStatementClose", "postgresStatementOpen":
		return FuncOf([]*Type{Simple(PostgresStatement)}, Simple(Bool)), true
	case "postgresStatementSQL":
		return FuncOf([]*Type{Simple(PostgresStatement)}, Simple(String)), true
	case "postgresTransactionExec":
		return FuncOf([]*Type{Simple(PostgresTransaction), Simple(String), Simple(SQLParameters)}, DictOf(Simple(String), Simple(Int))), true
	case "postgresTransactionQuery":
		return FuncOf([]*Type{Simple(PostgresTransaction), Simple(String), Simple(SQLParameters)}, ArrayOf(DictOf(Simple(String), Simple(Unknown)))), true
	case "postgresTransactionStream":
		return FuncOf([]*Type{Simple(PostgresTransaction), Simple(String), Simple(SQLParameters)}, Simple(SQLRows)), true
	case "postgresTransactionPrepare":
		return FuncOf([]*Type{Simple(PostgresTransaction), Simple(String)}, Simple(PostgresStatement)), true
	case "postgresSavepoint", "postgresRollbackTo", "postgresRelease":
		return FuncOf([]*Type{Simple(PostgresTransaction), Simple(String)}, Simple(Bool)), true
	case "postgresCommit", "postgresRollback", "postgresTransactionActive":
		return FuncOf([]*Type{Simple(PostgresTransaction)}, Simple(Bool)), true

	case "redisOpen":
		return FuncOf([]*Type{Simple(String), Simple(Int), Simple(String), Simple(Int), Simple(Int)}, Simple(RedisClient)), true
	case "redisPing", "redisClose", "redisIsOpen":
		return FuncOf([]*Type{Simple(RedisClient)}, Simple(Bool)), true
	case "redisSetClient":
		return FuncOf([]*Type{Simple(RedisClient), Simple(String), Simple(Unknown), Simple(Int)}, Simple(Bool)), true
	case "redisGetClient":
		return FuncOf([]*Type{Simple(RedisClient), Simple(String)}, Simple(Unknown)), true
	case "redisDelete", "redisExists":
		return FuncOf([]*Type{Simple(RedisClient), Simple(String)}, Simple(Int)), true
	case "redisExpire":
		return FuncOf([]*Type{Simple(RedisClient), Simple(String), Simple(Int)}, Simple(Bool)), true
	case "redisTTL":
		return FuncOf([]*Type{Simple(RedisClient), Simple(String)}, Simple(Int)), true
	case "redisIncrement":
		return FuncOf([]*Type{Simple(RedisClient), Simple(String), Simple(Int)}, Simple(Int)), true
	case "redisPipeline":
		return FuncOf([]*Type{Simple(RedisClient), ArrayOf(DictOf(Simple(String), Simple(Unknown)))}, ArrayOf(Simple(Unknown))), true
	case "redisPoolStats":
		return FuncOf([]*Type{Simple(RedisClient)}, DictOf(Simple(String), Simple(Int))), true

	case "configLoad":
		return FuncOf([]*Type{Simple(String)}, Simple(Config)), true
	case "configFrom":
		return FuncOf([]*Type{DictOf(Simple(String), Simple(Unknown))}, Simple(Config)), true
	case "configEnv":
		return FuncOf([]*Type{Simple(String)}, Simple(Config)), true
	case "configMerge":
		return FuncOf([]*Type{Simple(Config), Simple(Config)}, Simple(Config)), true
	case "configRequired":
		return FuncOf([]*Type{Simple(Config), Simple(String), Simple(Unknown)}, Simple(Unknown)), true
	case "configString":
		return FuncOf([]*Type{Simple(Config), Simple(String), Simple(Unknown)}, Simple(String)), true
	case "configInt":
		return FuncOf([]*Type{Simple(Config), Simple(String), Simple(Unknown)}, Simple(Int)), true
	case "configFloat":
		return FuncOf([]*Type{Simple(Config), Simple(String), Simple(Unknown)}, Simple(Float)), true
	case "configBool":
		return FuncOf([]*Type{Simple(Config), Simple(String), Simple(Unknown)}, Simple(Bool)), true
	case "configSecret":
		return FuncOf([]*Type{Simple(Config), Simple(String)}, Simple(Config)), true
	case "configRedacted":
		return FuncOf([]*Type{Simple(Config)}, DictOf(Simple(String), Simple(Unknown))), true

	case "logger":
		return FuncOf([]*Type{Simple(String), Simple(String), Simple(String)}, Simple(Logger)), true
	case "loggerWith":
		return FuncOf([]*Type{Simple(Logger), DictOf(Simple(String), Simple(Unknown))}, Simple(Logger)), true
	case "loggerSetLevel":
		return FuncOf([]*Type{Simple(Logger), Simple(String)}, Simple(Logger)), true
	case "loggerLog":
		return FuncOf([]*Type{Simple(Logger), Simple(String), Simple(String), DictOf(Simple(String), Simple(Unknown))}, Simple(Bool)), true
	case "loggerClose":
		return FuncOf([]*Type{Simple(Logger)}, Simple(Bool)), true
	case "metrics":
		return FuncOf(nil, Simple(MetricsRegistry)), true
	case "metricsCounter", "metricsGauge", "metricsHistogram":
		return FuncOf([]*Type{Simple(MetricsRegistry), Simple(String), Simple(Unknown), DictOf(Simple(String), Simple(String))}, Simple(Bool)), true
	case "metricsSnapshot":
		return FuncOf([]*Type{Simple(MetricsRegistry)}, DictOf(Simple(String), Simple(Unknown))), true
	case "metricsReset":
		return FuncOf([]*Type{Simple(MetricsRegistry)}, Simple(Bool)), true
	case "traceStart":
		return FuncOf([]*Type{Simple(String), DictOf(Simple(String), Simple(Unknown))}, Simple(TraceSpan)), true
	case "traceChild":
		return FuncOf([]*Type{Simple(TraceSpan), Simple(String), DictOf(Simple(String), Simple(Unknown))}, Simple(TraceSpan)), true
	case "traceSet":
		return FuncOf([]*Type{Simple(TraceSpan), Simple(String), Simple(Unknown)}, Simple(TraceSpan)), true
	case "traceEvent":
		return FuncOf([]*Type{Simple(TraceSpan), Simple(String), DictOf(Simple(String), Simple(Unknown))}, Simple(TraceSpan)), true
	case "traceFinish":
		return FuncOf([]*Type{Simple(TraceSpan), Simple(String)}, DictOf(Simple(String), Simple(Unknown))), true
	case "traceActive":
		return FuncOf([]*Type{Simple(TraceSpan)}, Simple(Bool)), true
	case "sessionSQLite":
		return FuncOf([]*Type{Simple(String)}, Simple(SessionStore)), true
	case "sessionRedis":
		return FuncOf([]*Type{Simple(RedisClient), Simple(String)}, Simple(SessionStore)), true
	case "sessionCreate":
		return FuncOf([]*Type{Simple(SessionStore), DictOf(Simple(String), Simple(Unknown)), Simple(Int)}, Simple(String)), true
	case "sessionGet":
		return FuncOf([]*Type{Simple(SessionStore), Simple(String)}, Simple(Unknown)), true
	case "sessionSet":
		return FuncOf([]*Type{Simple(SessionStore), Simple(String), DictOf(Simple(String), Simple(Unknown)), Simple(Int)}, Simple(Bool)), true
	case "sessionDelete":
		return FuncOf([]*Type{Simple(SessionStore), Simple(String)}, Simple(Bool)), true
	case "sessionRotate":
		return FuncOf([]*Type{Simple(SessionStore), Simple(String), Simple(Int)}, Simple(String)), true
	case "sessionTouch":
		return FuncOf([]*Type{Simple(SessionStore), Simple(String), Simple(Int)}, Simple(Bool)), true
	case "sessionCleanup":
		return FuncOf([]*Type{Simple(SessionStore)}, Simple(Int)), true
	case "sessionClose":
		return FuncOf([]*Type{Simple(SessionStore)}, Simple(Bool)), true
	case "rateLimiter":
		return FuncOf([]*Type{Simple(Int), Simple(Int)}, Simple(RateLimiter)), true
	case "rateAllow":
		return FuncOf([]*Type{Simple(RateLimiter), Simple(String)}, DictOf(Simple(String), Simple(Unknown))), true
	case "rateReset":
		return FuncOf([]*Type{Simple(RateLimiter), Simple(String)}, Simple(Bool)), true
	case "fileExists":
		return FuncOf([]*Type{Simple(String)}, Simple(Bool)), true
	case "jsonReadFile":
		return FuncOf([]*Type{Simple(String)}, Simple(Unknown)), true
	case "jsonWriteFile":
		return FuncOf([]*Type{Simple(String), Simple(Unknown), Simple(Bool)}, Simple(Int)), true
	case "jsonReadResult":
		return FuncOf([]*Type{Simple(String)}, DictOf(Simple(String), Simple(Unknown))), true
	case "jsonWriteResult":
		return FuncOf([]*Type{Simple(String), Simple(Unknown), Simple(Bool)}, DictOf(Simple(String), Simple(Unknown))), true
	case "csvReadFile":
		return FuncOf([]*Type{Simple(String)}, ArrayOf(ArrayOf(Simple(String)))), true
	case "csvWriteFile":
		return FuncOf([]*Type{Simple(String), ArrayOf(ArrayOf(Simple(Unknown)))}, Simple(Int)), true
	case "csvReadResult":
		return FuncOf([]*Type{Simple(String)}, DictOf(Simple(String), Simple(Unknown))), true
	case "csvWriteResult":
		return FuncOf([]*Type{Simple(String), ArrayOf(ArrayOf(Simple(Unknown)))}, DictOf(Simple(String), Simple(Unknown))), true
	case "binaryEncode":
		return FuncOf([]*Type{Simple(Unknown)}, ByteArrayOf()), true
	case "binaryDecode":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Unknown)), true
	case "binaryWriteFile":
		return FuncOf([]*Type{Simple(String), Simple(Unknown)}, Simple(Int)), true
	case "binaryReadFile":
		return FuncOf([]*Type{Simple(String)}, Simple(Unknown)), true

	case "desktopApp":
		return FuncOf([]*Type{DictOf(Simple(String), Simple(Unknown))}, Simple(DesktopApp)), true
	case "desktopBackend":
		return FuncOf([]*Type{Simple(DesktopApp)}, Simple(String)), true
	case "desktopWindow":
		return FuncOf([]*Type{Simple(DesktopApp), DictOf(Simple(String), Simple(Unknown))}, Simple(DesktopWindow)), true
	case "desktopOn":
		handler := FuncOf([]*Type{DictOf(Simple(String), Simple(Unknown))}, Simple(Unknown))
		return FuncOf([]*Type{Simple(DesktopApp), Simple(String), handler}, Simple(DesktopApp)), true
	case "desktopShortcut":
		handler := FuncOf([]*Type{DictOf(Simple(String), Simple(Unknown))}, Simple(Unknown))
		return FuncOf([]*Type{Simple(DesktopApp), Simple(String), handler}, Simple(DesktopApp)), true
	case "desktopPoll":
		return FuncOf([]*Type{Simple(DesktopApp), Simple(Int)}, Simple(Unknown)), true
	case "desktopRun":
		return FuncOf([]*Type{Simple(DesktopApp)}, Simple(Bool)), true
	case "desktopQuit", "desktopRunning", "desktopClose":
		return FuncOf([]*Type{Simple(DesktopApp)}, Simple(Bool)), true
	case "desktopEmit":
		return FuncOf([]*Type{Simple(DesktopApp), DictOf(Simple(String), Simple(Unknown))}, Simple(Bool)), true
	case "desktopSetClipboard":
		return FuncOf([]*Type{Simple(DesktopApp), Simple(String)}, Simple(Bool)), true
	case "desktopClipboard":
		return FuncOf([]*Type{Simple(DesktopApp)}, Simple(String)), true
	case "desktopPickFile":
		return FuncOf([]*Type{Simple(DesktopApp), DictOf(Simple(String), Simple(Unknown))}, ArrayOf(Simple(String))), true
	case "desktopPickFolder":
		return FuncOf([]*Type{Simple(DesktopApp), DictOf(Simple(String), Simple(Unknown))}, Simple(Unknown)), true
	case "desktopNotify":
		return FuncOf([]*Type{Simple(DesktopApp), DictOf(Simple(String), Simple(Unknown))}, Simple(Bool)), true
	case "desktopPaths":
		return FuncOf([]*Type{Simple(DesktopApp)}, DictOf(Simple(String), Simple(String))), true
	case "desktopOpenExternal":
		return FuncOf([]*Type{Simple(DesktopApp), Simple(String)}, Simple(Bool)), true
	case "desktopTray":
		return FuncOf([]*Type{Simple(DesktopApp), DictOf(Simple(String), Simple(Unknown))}, Simple(DesktopTray)), true
	case "desktopTrayAdd":
		handler := FuncOf([]*Type{DictOf(Simple(String), Simple(Unknown))}, Simple(Unknown))
		return FuncOf([]*Type{Simple(DesktopTray), Simple(String), Simple(String), handler}, Simple(DesktopTray)), true
	case "desktopTrayTooltip":
		return FuncOf([]*Type{Simple(DesktopTray), Simple(String)}, Simple(DesktopTray)), true
	case "desktopTrayClose", "desktopTrayOpen":
		return FuncOf([]*Type{Simple(DesktopTray)}, Simple(Bool)), true
	case "desktopSpawn":
		return FuncOf([]*Type{Simple(String), DictOf(Simple(String), Simple(Unknown))}, Simple(DesktopProcess)), true
	case "desktopProcessWait", "desktopProcessId":
		return FuncOf([]*Type{Simple(DesktopProcess)}, Simple(Int)), true
	case "desktopProcessKill", "desktopProcessRunning":
		return FuncOf([]*Type{Simple(DesktopProcess)}, Simple(Bool)), true
	case "desktopWindowShow", "desktopWindowHide", "desktopWindowClose", "desktopWindowMaximize", "desktopWindowMinimize", "desktopWindowRestore", "desktopWindowFocus":
		return FuncOf([]*Type{Simple(DesktopWindow)}, Simple(DesktopWindow)), true
	case "desktopWindowOpen", "desktopWindowFullscreen":
		return FuncOf([]*Type{Simple(DesktopWindow)}, Simple(Bool)), true
	case "desktopWindowId":
		return FuncOf([]*Type{Simple(DesktopWindow)}, Simple(Int)), true
	case "desktopWindowTitle":
		return FuncOf([]*Type{Simple(DesktopWindow)}, Simple(String)), true
	case "desktopWindowSetTitle", "desktopWindowSetIcon":
		return FuncOf([]*Type{Simple(DesktopWindow), Simple(String)}, Simple(DesktopWindow)), true
	case "desktopWindowSize", "desktopWindowPixelSize", "desktopWindowPosition":
		return FuncOf([]*Type{Simple(DesktopWindow)}, DictOf(Simple(String), Simple(Int))), true
	case "desktopWindowSetSize", "desktopWindowSetPosition":
		return FuncOf([]*Type{Simple(DesktopWindow), Simple(Int), Simple(Int)}, Simple(DesktopWindow)), true
	case "desktopWindowSetFullscreen":
		return FuncOf([]*Type{Simple(DesktopWindow), Simple(Bool)}, Simple(DesktopWindow)), true
	case "desktopWindowDisplayScale", "desktopWindowPixelDensity":
		return FuncOf([]*Type{Simple(DesktopWindow)}, Simple(Float)), true

	case "uiTheme":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(UITheme)), true
	case "uiState":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(UIState)), true
	case "uiStateGet":
		return FuncOf([]*Type{Simple(UIState)}, Simple(Unknown)), true
	case "uiStateSet":
		return FuncOf([]*Type{Simple(UIState), Simple(Unknown)}, Simple(Unknown)), true
	case "uiStateSubscribe":
		handler := FuncOf([]*Type{Simple(Unknown)}, Simple(Unknown))
		return FuncOf([]*Type{Simple(UIState), handler}, Simple(UIState)), true
	case "uiBind":
		return FuncOf([]*Type{Simple(UINode), Simple(String), Simple(UIState)}, Simple(UINode)), true
	case "uiNode":
		return FuncOf([]*Type{Simple(String), DictOf(Simple(String), Simple(Unknown)), ArrayOf(Simple(UINode))}, Simple(UINode)), true
	case "uiRow", "uiColumn", "uiContainer", "uiText", "uiButton", "uiInput", "uiTextarea", "uiSelect", "uiCheckbox", "uiRadio", "uiTable", "uiList", "uiTree", "uiTabs", "uiMenu", "uiModal", "uiTooltip", "uiProgress", "uiImage", "uiCanvas", "uiSpacer", "uiCustom":
		return FuncOf([]*Type{DictOf(Simple(String), Simple(Unknown)), ArrayOf(Simple(UINode))}, Simple(UINode)), true
	case "uiMount":
		return FuncOf([]*Type{Simple(DesktopApp), Simple(DesktopWindow), Simple(UINode), DictOf(Simple(String), Simple(Unknown))}, Simple(UIContext)), true
	case "uiUnmount":
		return FuncOf([]*Type{Simple(UIContext)}, Simple(Bool)), true
	case "uiRender":
		return FuncOf([]*Type{Simple(UIContext)}, Simple(Bool)), true
	case "uiSnapshot":
		return FuncOf([]*Type{Simple(UIContext)}, DictOf(Simple(String), Simple(Unknown))), true
	case "uiSetTheme":
		return FuncOf([]*Type{Simple(UIContext), Simple(UITheme)}, Simple(UIContext)), true
	case "uiDispatch":
		return FuncOf([]*Type{Simple(UIContext), DictOf(Simple(String), Simple(Unknown))}, Simple(Bool)), true
	case "uiSet":
		return FuncOf([]*Type{Simple(UINode), Simple(String), Simple(Unknown)}, Simple(UINode)), true
	case "uiGet":
		return FuncOf([]*Type{Simple(UINode), Simple(String)}, Simple(Unknown)), true
	case "uiAdd":
		return FuncOf([]*Type{Simple(UINode), Simple(UINode)}, Simple(UINode)), true
	case "uiRemove":
		return FuncOf([]*Type{Simple(UINode), Simple(String)}, Simple(Bool)), true
	case "uiFind":
		return FuncOf([]*Type{Simple(UIContext), Simple(String)}, Simple(Unknown)), true
	case "uiFocus":
		return FuncOf([]*Type{Simple(UIContext), Simple(UINode)}, Simple(Bool)), true
	case "uiFocusNext":
		return FuncOf([]*Type{Simple(UIContext), Simple(Bool)}, Simple(Unknown)), true
	case "uiAccessibility":
		return FuncOf([]*Type{Simple(UIContext)}, ArrayOf(DictOf(Simple(String), Simple(Unknown)))), true
	case "uiCanvasCommand":
		return FuncOf([]*Type{Simple(UINode), Simple(String), DictOf(Simple(String), Simple(Unknown))}, Simple(UINode)), true

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
