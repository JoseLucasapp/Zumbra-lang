package builtins

import (
	"fmt"
	"zumbra/binarydata"
	"zumbra/numeric"
	"zumbra/object"
)

var Builtins = []struct {
	Name    string
	Builtin *object.Builtin
}{
	{
		"addToArrayStart", AddToArrayStartBuiltin(),
	},
	{
		"addToArrayEnd", AddToArrayEndBuiltin(),
	},
	{
		"addToDict", AddToDictBuiltin(),
	},
	{
		"allButFirst", AllButFirstBuiltin(),
	},
	{
		"bhaskara", BhaskaraBuiltin(),
	},
	{
		"capitalize", CapitalizeBuiltin(),
	},
	{
		"date", DateBuiltin(),
	},
	{
		"deleteFromDict", DeleteFromDictBuiltin(),
	},
	{
		"dictKeys", DictKeysBuiltin(),
	},
	{
		"dictValues", DictValuesBuiltin(),
	},
	{
		"dotenvLoad", loadEnvBuiltin(),
	},
	{
		"dotenvGet", getEnvBuiltin(),
	},
	{
		"first", ArrayFirstBuiltin(),
	},
	{
		"get", GetBuiltin(),
	},
	{
		"getFromDict", GetFromDictBuiltin(),
	},
	{
		"hashCode", HashCodeBuiltin(),
	},
	{
		"html", HtmlHandlerBuiltin(),
	},
	{
		"indexOf", IndexOfBuiltin(),
	},
	{
		"input", InputBuiltin(),
	},
	{
		"jsonParse", JSONParseZ11Builtin(),
	},
	{
		"jwtCreateToken", createTokenBuiltin(),
	},
	{
		"jwtVerifyToken", verifyTokenBuiltin(),
	},
	{
		"last", ArrayLastBuiltin(),
	},
	{
		"max", MaxBuiltin(),
	},
	{
		"min", MinBuiltin(),
	},
	{
		"mysqlConnection", MySqlConnectionBuiltin(),
	},
	{
		"mysqlCreateTable", mysqlCreateTableBuiltin(),
	},
	{
		"mysqlDeleteFromTable", mysqlDeleteFromTableBuiltin(),
	},
	{
		"mysqlDropTable", mysqlDeleteTableBuiltin(),
	},
	{
		"mysqlGetFromTable", mysqlGetFromTableBuiltin(),
	},
	{
		"mysqlInsertIntoTable", mysqlInsertIntoTableBuiltin(),
	},
	{
		"mysqlShowTables", mysqlShowTablesBuiltin(),
	},
	{
		"mysqlShowTableColumns", mysqlShowTableColumnsBuiltin(),
	},
	{
		"mysqlUpdateIntoTable", mysqlUpdateIntoTableBuiltin(),
	},
	{
		"organize", OrganizeBuiltins(),
	},
	{
		"randomFloat", GenerateRandomFloatBuiltin(),
	},
	{
		"randomInteger", GenerateRandomIntegerBuiltin(),
	},
	{
		"registerRoute", RegisterRoutesBuiltin(),
	},
	{
		"removeFromArray", RemoveFromArrayBuiltin(),
	},
	{
		"removeWhiteSpaces", RemoveWhiteSpacesBuiltin(),
	},
	{
		"replace", ReplaceBuiltin(),
	},
	{
		"sendEmail", SendEmailBuiltin(),
	},
	{
		"sendWhatsapp", SendWhatsappBuiltin(),
	},
	{
		"server", CreateServerBuiltin(),
	},
	{
		"serveFile", ServeFileBuiltin(),
	},
	{
		"serveStatic", ServerStaticBuiltin(),
	},
	{
		"show", ShowBuiltin(),
	},
	{
		"sizeOf", SizeOfBuiltin(),
	},
	{
		"sum", SumBuiltin(),
	},
	{
		"toBool", ToBoolParserBuiltin(),
	},
	{
		"toFloat", ToFloatParserBuiltin(),
	},
	{
		"toInt", ToIntParserBuiltin(),
	},
	{
		"toLowercase", LowercaseBuiltin(),
	},
	{
		"toString", ToStringParserBuiltin(),
	},
	{
		"toUppercase", UppercaseBuiltin(),
	},

	// files
	{
		"createCsv", CreateCsvBuiltin(),
	},
	{
		"createDoc", CreateDocBuiltin(),
	},
	{
		"createFile", CreateFileBuiltin(),
	},
	{
		"createPdf", CreatePdfBuiltin(),
	},
	{
		"createTxt", CreateTxtBuiltin(),
	},

	// rest
	{
		"restDelete", RestDeleteBuiltin(),
	},
	{
		"restGet", RestGetBuiltin(),
	},
	{
		"restPatch", RestPatchBuiltin(),
	},
	{
		"restPost", RestPostBuiltin(),
	},
	{
		"restPut", RestPutBuiltin(),
	},

	// utils
	{
		"switchCase", SwitchCaseBuiltin(),
	},

	{
		"postgresConnection", PostgresConnectionBuiltin(),
	},
	{
		"postgresExec", PostgresExecBuiltin(),
	},
	{
		"postgresQuery", PostgresQueryBuiltin(),
	},
	{
		"redisConnection", RedisConnectionBuiltin(),
	},
	{
		"redisSet", RedisSetBuiltin(),
	},
	{
		"redisGet", RedisGetBuiltin(),
	},
	{
		"redisDel", RedisDelBuiltin(),
	},

	// Z12.1 SQLite embedded persistence
	{"sqliteOpen", SQLiteOpenBuiltin()},
	{"sqliteMemory", SQLiteMemoryBuiltin()},
	{"sqliteExec", SQLiteExecBuiltin()},
	{"sqliteQuery", SQLiteQueryBuiltin()},
	{"sqlitePrepare", SQLitePrepareBuiltin()},
	{"sqliteBegin", SQLiteBeginBuiltin()},
	{"sqliteClose", SQLiteCloseBuiltin()},
	{"sqliteIsOpen", SQLiteIsOpenBuiltin()},
	{"sqlitePath", SQLitePathBuiltin()},
	{"sqliteStatementExec", SQLiteStatementExecBuiltin()},
	{"sqliteStatementQuery", SQLiteStatementQueryBuiltin()},
	{"sqliteStatementClose", SQLiteStatementCloseBuiltin()},
	{"sqliteStatementOpen", SQLiteStatementOpenBuiltin()},
	{"sqliteStatementSQL", SQLiteStatementSQLBuiltin()},
	{"sqliteTransactionExec", SQLiteTransactionExecBuiltin()},
	{"sqliteTransactionQuery", SQLiteTransactionQueryBuiltin()},
	{"sqliteTransactionPrepare", SQLiteTransactionPrepareBuiltin()},
	{"sqliteCommit", SQLiteCommitBuiltin()},
	{"sqliteRollback", SQLiteRollbackBuiltin()},
	{"sqliteTransactionActive", SQLiteTransactionActiveBuiltin()},
	{
		"supabaseConnection", SupabaseConnectionBuiltin(),
	},
	{
		"supabaseSelect", SupabaseSelectBuiltin(),
	},
	{
		"supabaseInsert", SupabaseInsertBuiltin(),
	},
	{
		"supabaseQuery", SupabaseQueryBuiltin(),
	},
	{
		"supabaseUpdate", SupabaseUpdateBuiltin(),
	},
	{
		"supabaseDelete", SupabaseDeleteBuiltin(),
	},
	{
		"supabaseUpsert", SupabaseUpsertBuiltin(),
	},
	{
		"supabaseRpc", SupabaseRpcBuiltin(),
	},
	{
		"supabaseCount", SupabaseCountBuiltin(),
	},
	{
		"supabaseSingle", SupabaseSingleBuiltin(),
	},
	{
		"supabaseStorageUpload", SupabaseStorageUploadBuiltin(),
	},
	{
		"supabaseStorageDelete", SupabaseStorageDeleteBuiltin(),
	},
	{
		"supabaseStoragePublicUrl", SupabaseStoragePublicUrlBuiltin(),
	},
	{
		"supabaseStorageSignedUrl", SupabaseStorageSignedUrlBuiltin(),
	},
	{
		"supabaseStorageDownload", SupabaseStorageDownloadBuiltin(),
	},
	{
		"supabaseAuthSignUp", SupabaseAuthSignUpBuiltin(),
	},
	{
		"supabaseAuthSignIn", SupabaseAuthSignInBuiltin(),
	},
	{"error", &ErrorBuiltin},
	{"panic", &PanicBuiltin},

	// fixed-width integers
	{
		"u8", FixedIntegerConversionBuiltin(object.FixedU8),
	},
	{
		"u16", FixedIntegerConversionBuiltin(object.FixedU16),
	},
	{
		"u32", FixedIntegerConversionBuiltin(object.FixedU32),
	},
	{
		"u64", FixedIntegerConversionBuiltin(object.FixedU64),
	},
	{
		"i8", FixedIntegerConversionBuiltin(object.FixedI8),
	},
	{
		"i16", FixedIntegerConversionBuiltin(object.FixedI16),
	},
	{
		"i32", FixedIntegerConversionBuiltin(object.FixedI32),
	},
	{
		"i64", FixedIntegerConversionBuiltin(object.FixedI64),
	},
	{
		"wrapAdd", FixedArithmeticBuiltin(numeric.Wrapping, "+", "wrapAdd"),
	},
	{
		"wrapSub", FixedArithmeticBuiltin(numeric.Wrapping, "-", "wrapSub"),
	},
	{
		"wrapMul", FixedArithmeticBuiltin(numeric.Wrapping, "*", "wrapMul"),
	},
	{
		"checkedAdd", FixedArithmeticBuiltin(numeric.Checked, "+", "checkedAdd"),
	},
	{
		"checkedSub", FixedArithmeticBuiltin(numeric.Checked, "-", "checkedSub"),
	},
	{
		"checkedMul", FixedArithmeticBuiltin(numeric.Checked, "*", "checkedMul"),
	},
	{
		"satAdd", FixedArithmeticBuiltin(numeric.Saturating, "+", "satAdd"),
	},
	{
		"satSub", FixedArithmeticBuiltin(numeric.Saturating, "-", "satSub"),
	},
	{
		"satMul", FixedArithmeticBuiltin(numeric.Saturating, "*", "satMul"),
	},

	// compact memory collections
	{
		"bytes", BytesBuiltin(),
	},
	{
		"arrayOf", ArrayOfBuiltin(),
	},
	{
		"slice", SliceBuiltin(),
	},
	{
		"fill", FillBuiltin(),
	},

	// concurrency and synchronization
	{"join", JoinTaskBuiltin()},
	{"cancel", CancelTaskBuiltin()},
	{"taskDone", TaskDoneBuiltin()},
	{"taskCancelled", TaskCancelledBuiltin()},
	{"joinTimeout", JoinTimeoutBuiltin()},
	{"sleepMs", SleepMsBuiltin()},
	{"channel", ChannelBuiltin()},
	{"send", SendBuiltin()},
	{"receive", ReceiveBuiltin()},
	{"receiveOk", ReceiveOkBuiltin()},
	{"receiveTimeout", ReceiveTimeoutBuiltin()},
	{"closeChannel", CloseChannelBuiltin()},
	{"channelClosed", ChannelClosedBuiltin()},
	{"channelLen", ChannelLenBuiltin()},
	{"channelCap", ChannelCapBuiltin()},
	{"mutex", MutexBuiltin()},
	{"lock", LockBuiltin()},
	{"unlock", UnlockBuiltin()},
	{"rwMutex", RWMutexBuiltin()},
	{"rLock", RLockBuiltin()},
	{"rUnlock", RUnlockBuiltin()},
	{"waitGroup", WaitGroupBuiltin()},
	{"wgAdd", WGAddBuiltin()},
	{"wgDone", WGDoneBuiltin()},
	{"wgWait", WGWaitBuiltin()},
	{"semaphore", SemaphoreBuiltin()},
	{"acquire", AcquireBuiltin()},
	{"release", ReleaseBuiltin()},
	{"atomicInt", AtomicIntBuiltin()},
	{"atomicLoad", AtomicLoadBuiltin()},
	{"atomicStore", AtomicStoreBuiltin()},
	{"atomicAdd", AtomicAddBuiltin()},
	{"atomicSwap", AtomicSwapBuiltin()},
	{"atomicCompareSwap", AtomicCompareSwapBuiltin()},

	// TCP, TLS, UDP, DNS and byte streams
	{"tcpListen", TCPListenBuiltin()},
	{"tcpConnect", TCPConnectBuiltin(false)},
	{"tcpConnectTimeout", TCPConnectBuiltin(true)},
	{"tlsListen", TLSListenBuiltin()},
	{"tlsConnect", TLSConnectBuiltin(false)},
	{"tlsConnectTimeout", TLSConnectBuiltin(true)},
	{"listenerAccept", ListenerAcceptBuiltin(false)},
	{"listenerAcceptTimeout", ListenerAcceptBuiltin(true)},
	{"listenerClose", ListenerCloseBuiltin()},
	{"listenerClosed", ListenerClosedBuiltin()},
	{"listenerAddress", ListenerAddressBuiltin(false)},
	{"listenerPort", ListenerAddressBuiltin(true)},
	{"streamRead", StreamReadBuiltin(false, false)},
	{"streamReadExact", StreamReadBuiltin(true, false)},
	{"streamReadTimeout", StreamReadBuiltin(false, true)},
	{"streamWrite", StreamWriteBuiltin(false)},
	{"streamWriteAll", StreamWriteBuiltin(true)},
	{"streamClose", StreamCloseBuiltin()},
	{"streamClosed", StreamClosedBuiltin()},
	{"streamShutdownRead", StreamShutdownBuiltin(false)},
	{"streamShutdownWrite", StreamShutdownBuiltin(true)},
	{"streamLocalAddress", StreamAddressBuiltin(false, false)},
	{"streamLocalPort", StreamAddressBuiltin(false, true)},
	{"streamRemoteAddress", StreamAddressBuiltin(true, false)},
	{"streamRemotePort", StreamAddressBuiltin(true, true)},
	{"streamSetReadTimeout", StreamSetTimeoutBuiltin(true)},
	{"streamSetWriteTimeout", StreamSetTimeoutBuiltin(false)},
	{"tcpSetKeepAlive", TCPSetKeepAliveBuiltin()},
	{"dnsLookup", DNSLookupBuiltin(false)},
	{"dnsLookupTimeout", DNSLookupBuiltin(true)},
	{"udpBind", UDPBindBuiltin()},
	{"udpSendTo", UDPSendToBuiltin()},
	{"udpReceiveFrom", UDPReceiveFromBuiltin(false)},
	{"udpReceiveFromTimeout", UDPReceiveFromBuiltin(true)},
	{"udpClose", UDPCloseBuiltin()},
	{"udpClosed", UDPClosedBuiltin()},
	{"udpAddress", UDPAddressBuiltin(false)},
	{"udpPort", UDPAddressBuiltin(true)},

	// HTTP/1.1, APIs, streaming, SSE, WebSockets and JWT
	{"httpApp", HTTPAppBuiltin()},
	{"httpRoute", HTTPRouteBuiltin()},
	{"httpUse", HTTPUseBuiltin()},
	{"httpStatic", HTTPStaticBuiltin()},
	{"httpLimitBody", HTTPLimitBodyBuiltin()},
	{"httpCompression", HTTPCompressionBuiltin()},
	{"httpCors", HTTPCorsBuiltin()},
	{"httpServe", HTTPServeBuiltin()},
	{"httpServeTLS", HTTPServeTLSBuiltin()},
	{"httpShutdown", HTTPShutdownBuiltin()},
	{"httpServerPort", HTTPServerPortBuiltin()},
	{"httpServerAddress", HTTPServerAddressBuiltin()},
	{"httpServerRunning", HTTPServerRunningBuiltin()},
	{"httpText", HTTPResponseBuiltin("text")},
	{"httpJson", HTTPResponseBuiltin("json")},
	{"httpHtml", HTTPResponseBuiltin("html")},
	{"httpRedirect", HTTPResponseBuiltin("redirect")},
	{"httpFile", HTTPFileResponseBuiltin()},
	{"httpHeader", HTTPHeaderBuiltin()},
	{"httpCookie", HTTPCookieBuiltin()},
	{"httpStream", HTTPStreamBuiltin(false)},
	{"httpSSE", HTTPStreamBuiltin(true)},
	{"sseEvent", SSEEventBuiltin()},
	{"httpRequest", HTTPRequestBuiltin()},
	{"httpStatus", HTTPClientStatusBuiltin()},
	{"httpBody", HTTPClientBodyBuiltin(false)},
	{"httpBodyBytes", HTTPClientBodyBuiltin(true)},
	{"httpBodyJSON", HTTPClientJSONBuiltin()},
	{"httpHeaders", HTTPClientHeadersBuiltin()},
	{"jsonStringify", JSONStringifyBuiltin()},
	{"jwtSignHS256", JWTSignHS256Builtin()},
	{"jwtVerifyHS256", JWTVerifyHS256Builtin()},
	{"webSocketUpgrade", WebSocketUpgradeBuiltin()},
	{"webSocketConnect", WebSocketConnectBuiltin()},
	{"webSocketRead", WebSocketReadBuiltin(false)},
	{"webSocketReadTimeout", WebSocketReadBuiltin(true)},
	{"webSocketWriteText", WebSocketWriteBuiltin(0x1)},
	{"webSocketWriteBinary", WebSocketWriteBuiltin(0x2)},
	{"webSocketPing", WebSocketWriteBuiltin(0x9)},
	{"webSocketClose", WebSocketCloseBuiltin()},
	{"webSocketClosed", WebSocketClosedBuiltin()},

	// binary files, endian access and hashing
	{
		"readBytes", ReadBytesBuiltin(),
	},
	{
		"writeBytes", WriteBytesBuiltin(),
	},
	{
		"readU16LE", ReadUnsignedBuiltin(2, binarydata.LittleEndian, "readU16LE"),
	},
	{
		"readU16BE", ReadUnsignedBuiltin(2, binarydata.BigEndian, "readU16BE"),
	},
	{
		"readU32LE", ReadUnsignedBuiltin(4, binarydata.LittleEndian, "readU32LE"),
	},
	{
		"readU32BE", ReadUnsignedBuiltin(4, binarydata.BigEndian, "readU32BE"),
	},
	{
		"readU64LE", ReadUnsignedBuiltin(8, binarydata.LittleEndian, "readU64LE"),
	},
	{
		"readU64BE", ReadUnsignedBuiltin(8, binarydata.BigEndian, "readU64BE"),
	},
	{
		"writeU16LE", WriteUnsignedBuiltin(2, binarydata.LittleEndian, "writeU16LE"),
	},
	{
		"writeU16BE", WriteUnsignedBuiltin(2, binarydata.BigEndian, "writeU16BE"),
	},
	{
		"writeU32LE", WriteUnsignedBuiltin(4, binarydata.LittleEndian, "writeU32LE"),
	},
	{
		"writeU32BE", WriteUnsignedBuiltin(4, binarydata.BigEndian, "writeU32BE"),
	},
	{
		"writeU64LE", WriteUnsignedBuiltin(8, binarydata.LittleEndian, "writeU64LE"),
	},
	{
		"writeU64BE", WriteUnsignedBuiltin(8, binarydata.BigEndian, "writeU64BE"),
	},
	{
		"copyBytes", CopyBytesBuiltin(),
	},
	{
		"bytesEqual", BytesEqualBuiltin(),
	},
	{
		"sha256", SHA256Builtin(),
	},

	// Z13 desktop runtime
	{"desktopApp", DesktopAppBuiltin()},
	{"desktopBackend", DesktopBackendBuiltin()},
	{"desktopWindow", DesktopWindowBuiltin()},
	{"desktopOn", DesktopOnBuiltin()},
	{"desktopShortcut", DesktopShortcutBuiltin()},
	{"desktopPoll", DesktopPollBuiltin()},
	{"desktopRun", DesktopRunBuiltin()},
	{"desktopQuit", DesktopQuitBuiltin()},
	{"desktopRunning", DesktopRunningBuiltin()},
	{"desktopClose", DesktopCloseBuiltin()},
	{"desktopEmit", DesktopEmitBuiltin()},
	{"desktopSetClipboard", DesktopSetClipboardBuiltin()},
	{"desktopClipboard", DesktopClipboardBuiltin()},
	{"desktopPickFile", DesktopPickFileBuiltin()},
	{"desktopPickFolder", DesktopPickFolderBuiltin()},
	{"desktopNotify", DesktopNotifyBuiltin()},
	{"desktopPaths", DesktopPathsBuiltin()},
	{"desktopOpenExternal", DesktopOpenExternalBuiltin()},
	{"desktopTray", DesktopTrayBuiltin()},
	{"desktopTrayAdd", DesktopTrayAddBuiltin()},
	{"desktopTrayTooltip", DesktopTrayTooltipBuiltin()},
	{"desktopTrayClose", DesktopTrayCloseBuiltin()},
	{"desktopTrayOpen", DesktopTrayOpenBuiltin()},
	{"desktopSpawn", DesktopSpawnBuiltin()},
	{"desktopProcessWait", DesktopProcessWaitBuiltin()},
	{"desktopProcessKill", DesktopProcessKillBuiltin()},
	{"desktopProcessRunning", DesktopProcessRunningBuiltin()},
	{"desktopProcessId", DesktopProcessIDBuiltin()},
	{"desktopWindowShow", DesktopWindowShowBuiltin()},
	{"desktopWindowHide", DesktopWindowHideBuiltin()},
	{"desktopWindowClose", DesktopWindowCloseBuiltin()},
	{"desktopWindowOpen", DesktopWindowOpenBuiltin()},
	{"desktopWindowId", DesktopWindowIDBuiltin()},
	{"desktopWindowTitle", DesktopWindowTitleBuiltin()},
	{"desktopWindowSetTitle", DesktopWindowSetTitleBuiltin()},
	{"desktopWindowSize", DesktopWindowSizeBuiltin()},
	{"desktopWindowPixelSize", DesktopWindowPixelSizeBuiltin()},
	{"desktopWindowSetSize", DesktopWindowSetSizeBuiltin()},
	{"desktopWindowPosition", DesktopWindowPositionBuiltin()},
	{"desktopWindowSetPosition", DesktopWindowSetPositionBuiltin()},
	{"desktopWindowFullscreen", DesktopWindowFullscreenBuiltin()},
	{"desktopWindowSetFullscreen", DesktopWindowSetFullscreenBuiltin()},
	{"desktopWindowMaximize", DesktopWindowMaximizeBuiltin()},
	{"desktopWindowMinimize", DesktopWindowMinimizeBuiltin()},
	{"desktopWindowRestore", DesktopWindowRestoreBuiltin()},
	{"desktopWindowFocus", DesktopWindowFocusBuiltin()},
	{"desktopWindowDisplayScale", DesktopWindowDisplayScaleBuiltin()},
	{"desktopWindowPixelDensity", DesktopWindowPixelDensityBuiltin()},
	{"desktopWindowSetIcon", DesktopWindowSetIconBuiltin()},

	// Z14 retained-mode GUI toolkit
	{"uiTheme", UIThemeBuiltin()},
	{"uiState", UIStateBuiltin()},
	{"uiStateGet", UIStateGetBuiltin()},
	{"uiStateSet", UIStateSetBuiltin()},
	{"uiStateSubscribe", UIStateSubscribeBuiltin()},
	{"uiBind", UIBindBuiltin()},
	{"uiNode", UINodeBuiltin()},
	{"uiRow", uiNodeBuiltin("row")},
	{"uiColumn", uiNodeBuiltin("column")},
	{"uiContainer", uiNodeBuiltin("container")},
	{"uiText", uiNodeBuiltin("text")},
	{"uiButton", uiNodeBuiltin("button")},
	{"uiInput", uiNodeBuiltin("input")},
	{"uiTextarea", uiNodeBuiltin("textarea")},
	{"uiSelect", uiNodeBuiltin("select")},
	{"uiCheckbox", uiNodeBuiltin("checkbox")},
	{"uiRadio", uiNodeBuiltin("radio")},
	{"uiTable", uiNodeBuiltin("table")},
	{"uiList", uiNodeBuiltin("list")},
	{"uiTree", uiNodeBuiltin("tree")},
	{"uiTabs", uiNodeBuiltin("tabs")},
	{"uiMenu", uiNodeBuiltin("menu")},
	{"uiModal", uiNodeBuiltin("modal")},
	{"uiTooltip", uiNodeBuiltin("tooltip")},
	{"uiProgress", uiNodeBuiltin("progress")},
	{"uiImage", uiNodeBuiltin("image")},
	{"uiCanvas", uiNodeBuiltin("canvas")},
	{"uiSpacer", uiNodeBuiltin("spacer")},
	{"uiCustom", uiNodeBuiltin("custom")},
	{"uiMount", UIMountBuiltin()},
	{"uiUnmount", UIUnmountBuiltin()},
	{"uiRender", UIRenderBuiltin()},
	{"uiSnapshot", UISnapshotBuiltin()},
	{"uiSetTheme", UISetThemeBuiltin()},
	{"uiDispatch", UIDispatchBuiltin()},
	{"uiSet", UISetBuiltin()},
	{"uiGet", UIGetBuiltin()},
	{"uiAdd", UIAddBuiltin()},
	{"uiRemove", UIRemoveBuiltin()},
	{"uiFind", UIFindBuiltin()},
	{"uiFocus", UIFocusBuiltin()},
	{"uiFocusNext", UIFocusNextBuiltin()},
	{"uiAccessibility", UIAccessibilityBuiltin()},
	{"uiCanvasCommand", UICanvasCommandBuiltin()},

	// Z15 embedded application assets
	{"assetExists", AssetExistsBuiltin()},
	{"assetText", AssetTextBuiltin()},
	{"assetBytes", AssetBytesBuiltin()},
	{"assetList", AssetListBuiltin()},
}

func NewBoolean(value bool) *object.Boolean {
	return &object.Boolean{Value: value}
}

func NewFloat(value float64) *object.Float {
	return &object.Float{Value: value}
}

func NewString(value string) *object.String {
	return &object.String{Value: value}
}

func NewInteger(value int64) *object.Integer {
	return &object.Integer{Value: value}
}

func NewError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func GetBuiltinByName(name string) *object.Builtin {
	for _, builtin := range Builtins {
		if builtin.Name == name {
			return builtin.Builtin
		}
	}
	return nil
}
