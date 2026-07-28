package evaluator

import (
	"fmt"
	"zumbra/object"
	"zumbra/object/builtins"
)

var builtinsList = make(map[string]*object.Builtin)

func init() {
	arrays := []string{
		"addToArrayStart", "addToArrayEnd", "allButFirst", "first", "indexOf", "last", "max", "min", "organize", "removeFromArray", "sizeOf", "sum",
	}

	dicts := []string{
		"addToDict", "deleteFromDict", "dictKeys", "dictValues", "getFromDict",
	}

	extras := []string{
		"date", "dotenvLoad", "dotenvGet", "hashCode",
	}

	http := []string{
		"get", "html", "registerRoute", "server", "serveFile", "serveStatic",
	}

	ioUtils := []string{
		"input", "show",
	}

	jwt := []string{
		"jwtCreateToken", "jwtVerifyToken",
	}

	messages := []string{
		"sendEmail", "sendWhatsapp",
	}

	mysql := []string{
		"mysqlConnection", "mysqlCreateTable", "mysqlDeleteFromTable", "mysqlDropTable", "mysqlGetFromTable", "mysqlInsertIntoTable", "mysqlShowTables", "mysqlShowTableColumns", "mysqlUpdateIntoTable",
	}

	numbersUtils := []string{
		"bhaskara", "randomFloat", "randomInteger",
	}

	parsers := []string{
		"json_parse", "toBool", "toFloat", "toInt", "toString",
	}

	memoryCollections := []string{
		"bytes", "arrayOf", "slice", "fill",
	}

	binaryIO := []string{
		"readBytes", "writeBytes",
		"readU16LE", "readU16BE", "readU32LE", "readU32BE", "readU64LE", "readU64BE",
		"writeU16LE", "writeU16BE", "writeU32LE", "writeU32BE", "writeU64LE", "writeU64BE",
		"copyBytes", "bytesEqual", "sha256",
	}

	concurrency := []string{
		"join", "cancel", "taskDone", "taskCancelled", "joinTimeout", "sleepMs",
		"channel", "send", "receive", "receiveOk", "receiveTimeout", "closeChannel", "channelClosed", "channelLen", "channelCap",
		"mutex", "lock", "unlock", "rwMutex", "rLock", "rUnlock",
		"waitGroup", "wgAdd", "wgDone", "wgWait", "semaphore", "acquire", "release",
		"atomicInt", "atomicLoad", "atomicStore", "atomicAdd", "atomicSwap", "atomicCompareSwap",
	}

	network := []string{
		"tcpListen", "tcpConnect", "tcpConnectTimeout", "tlsListen", "tlsConnect", "tlsConnectTimeout",
		"listenerAccept", "listenerAcceptTimeout", "listenerClose", "listenerClosed", "listenerAddress", "listenerPort",
		"streamRead", "streamReadExact", "streamReadTimeout", "streamWrite", "streamWriteAll", "streamClose", "streamClosed",
		"streamShutdownRead", "streamShutdownWrite", "streamLocalAddress", "streamLocalPort", "streamRemoteAddress", "streamRemotePort",
		"streamSetReadTimeout", "streamSetWriteTimeout", "tcpSetKeepAlive", "dnsLookup", "dnsLookupTimeout",
		"udpBind", "udpSendTo", "udpReceiveFrom", "udpReceiveFromTimeout", "udpClose", "udpClosed", "udpAddress", "udpPort",
	}

	sqlite := []string{
		"sqliteOpen", "sqliteMemory", "sqliteExec", "sqliteQuery", "sqlitePrepare", "sqliteBegin", "sqliteClose", "sqliteIsOpen", "sqlitePath",
		"sqliteStatementExec", "sqliteStatementQuery", "sqliteStatementClose", "sqliteStatementOpen", "sqliteStatementSQL",
		"sqliteTransactionExec", "sqliteTransactionQuery", "sqliteTransactionPrepare", "sqliteCommit", "sqliteRollback", "sqliteTransactionActive",
	}

	httpZ11 := []string{
		"httpApp", "httpRoute", "httpUse", "httpStatic", "httpLimitBody", "httpCompression", "httpCors",
		"httpServe", "httpServeTLS", "httpShutdown", "httpServerPort", "httpServerAddress", "httpServerRunning",
		"httpText", "httpJson", "httpHtml", "httpRedirect", "httpFile", "httpHeader", "httpCookie",
		"httpStream", "httpSSE", "sseEvent", "httpRequest", "httpStatus", "httpBody", "httpBodyBytes", "httpBodyJSON", "httpHeaders",
		"jsonStringify", "jsonParse", "jwtSignHS256", "jwtVerifyHS256",
		"webSocketUpgrade", "webSocketConnect", "webSocketRead", "webSocketReadTimeout", "webSocketWriteText", "webSocketWriteBinary",
		"webSocketPing", "webSocketClose", "webSocketClosed",
	}

	fixedIntegers := []string{
		"u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64",
		"wrapAdd", "wrapSub", "wrapMul",
		"checkedAdd", "checkedSub", "checkedMul",
		"satAdd", "satSub", "satMul",
	}

	stringUtils := []string{
		"capitalize", "removeWhiteSpaces", "replace", "toLowercase", "toUppercase",
	}

	allBuiltins := append(arrays, dicts...)
	allBuiltins = append(allBuiltins, http...)
	allBuiltins = append(allBuiltins, parsers...)
	allBuiltins = append(allBuiltins, fixedIntegers...)
	allBuiltins = append(allBuiltins, memoryCollections...)
	allBuiltins = append(allBuiltins, binaryIO...)
	allBuiltins = append(allBuiltins, concurrency...)
	allBuiltins = append(allBuiltins, network...)
	allBuiltins = append(allBuiltins, httpZ11...)
	allBuiltins = append(allBuiltins, sqlite...)
	allBuiltins = append(allBuiltins, stringUtils...)
	allBuiltins = append(allBuiltins, numbersUtils...)
	allBuiltins = append(allBuiltins, ioUtils...)
	allBuiltins = append(allBuiltins, messages...)
	allBuiltins = append(allBuiltins, extras...)
	allBuiltins = append(allBuiltins, mysql...)
	allBuiltins = append(allBuiltins, jwt...)

	for _, name := range allBuiltins {
		if builtin := builtins.GetBuiltinByName(name); builtin != nil {
			builtinsList[name] = builtin
		}
	}
	// Register every runtime builtin as a final pass so extension milestones do
	// not need to duplicate the canonical registry in the evaluator.
	for _, definition := range builtins.Builtins {
		builtinsList[definition.Name] = definition.Builtin
	}
	builtins.SetRouteInvoker(func(handler object.Object, args ...object.Object) (object.Object, error) {
		result := applyFunctionSync(handler, args)
		if errObj, ok := result.(*object.Error); ok {
			return result, fmt.Errorf("%s", errObj.Message)
		}
		return result, nil
	})
}
