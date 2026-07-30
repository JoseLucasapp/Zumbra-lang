// Package nativec lowers Zumbra MIR to portable C11. The generated program is
// standalone and links only the small native runtime plus the C standard
// library. Unsupported operations are rejected before invoking a C compiler.
package nativec

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"zumbra/builtinspec"
	"zumbra/hir"
	"zumbra/mir"
	"zumbra/types"
)

//go:embed runtime/zumbra_runtime.c runtime/zumbra_runtime.h runtime/zumbra_http.inc runtime/zumbra_z12.inc runtime/zumbra_sqlite.inc runtime/zumbra_desktop.inc runtime/zumbra_ui.inc
var runtimeFiles embed.FS

type Sources struct {
	Program []byte
	Runtime []byte
	Header  []byte
}

type Diagnostic struct {
	Instruction int
	Message     string
}

func (d Diagnostic) Error() string {
	if d.Instruction > 0 {
		return fmt.Sprintf("MIR instruction %d: %s", d.Instruction, d.Message)
	}
	return d.Message
}

var supportedBuiltins = map[string]bool{
	"show": true, "sizeOf": true, "toString": true, "addToArrayStart": true, "addToArrayEnd": true,
	"u8": true, "u16": true, "u32": true, "u64": true,
	"i8": true, "i16": true, "i32": true, "i64": true,
	"toInt": true, "toFloat": true, "toBool": true,
	"wrapAdd": true, "wrapSub": true, "wrapMul": true,
	"checkedAdd": true, "checkedSub": true, "checkedMul": true,
	"satAdd": true, "satSub": true, "satMul": true,
	"bytes": true, "arrayOf": true, "slice": true, "fill": true,
	"readBytes": true, "writeBytes": true,
	"readU16LE": true, "readU16BE": true, "readU32LE": true, "readU32BE": true,
	"readU64LE": true, "readU64BE": true,
	"writeU16LE": true, "writeU16BE": true, "writeU32LE": true, "writeU32BE": true,
	"writeU64LE": true, "writeU64BE": true,
	"copyBytes": true, "bytesEqual": true, "sha256": true,
	"join": true, "cancel": true, "taskDone": true, "taskCancelled": true, "joinTimeout": true, "sleepMs": true,
	"channel": true, "send": true, "receive": true, "receiveOk": true, "receiveTimeout": true, "closeChannel": true, "channelClosed": true, "channelLen": true, "channelCap": true,
	"mutex": true, "lock": true, "unlock": true, "rwMutex": true, "rLock": true, "rUnlock": true,
	"waitGroup": true, "wgAdd": true, "wgDone": true, "wgWait": true, "semaphore": true, "acquire": true, "release": true,
	"atomicInt": true, "atomicLoad": true, "atomicStore": true, "atomicAdd": true, "atomicSwap": true, "atomicCompareSwap": true,
	"tcpListen": true, "tcpConnect": true, "tcpConnectTimeout": true,
	"tlsListen": true, "tlsConnect": true, "tlsConnectTimeout": true,
	"listenerAccept": true, "listenerAcceptTimeout": true, "listenerClose": true, "listenerClosed": true, "listenerAddress": true, "listenerPort": true,
	"streamRead": true, "streamReadExact": true, "streamReadTimeout": true, "streamWrite": true, "streamWriteAll": true,
	"streamClose": true, "streamClosed": true, "streamShutdownRead": true, "streamShutdownWrite": true,
	"streamLocalAddress": true, "streamLocalPort": true, "streamRemoteAddress": true, "streamRemotePort": true,
	"streamSetReadTimeout": true, "streamSetWriteTimeout": true, "tcpSetKeepAlive": true,
	"dnsLookup": true, "dnsLookupTimeout": true,
	"udpBind": true, "udpSendTo": true, "udpReceiveFrom": true, "udpReceiveFromTimeout": true, "udpClose": true, "udpClosed": true, "udpAddress": true, "udpPort": true,
	"httpApp": true, "httpRoute": true, "httpUse": true, "httpStatic": true, "httpLimitBody": true, "httpCompression": true, "httpCors": true,
	"httpServe": true, "httpServeTLS": true, "httpShutdown": true, "httpServerPort": true, "httpServerAddress": true, "httpServerRunning": true,
	"httpText": true, "httpJson": true, "httpHtml": true, "httpRedirect": true, "httpFile": true, "httpHeader": true, "httpCookie": true,
	"httpStream": true, "httpSSE": true, "sseEvent": true, "httpRequest": true, "httpStatus": true, "httpBody": true, "httpBodyBytes": true, "httpBodyJSON": true, "httpHeaders": true,
	"jsonStringify": true, "jsonParse": true, "jwtSignHS256": true, "jwtVerifyHS256": true,
	"webSocketUpgrade": true, "webSocketConnect": true, "webSocketRead": true, "webSocketReadTimeout": true,
	"webSocketWriteText": true, "webSocketWriteBinary": true, "webSocketPing": true, "webSocketClose": true, "webSocketClosed": true,
	"sqliteOpen": true, "sqliteMemory": true, "sqliteExec": true, "sqliteQuery": true, "sqlitePrepare": true, "sqliteBegin": true, "sqliteClose": true, "sqliteIsOpen": true, "sqlitePath": true,
	"sqliteStatementExec": true, "sqliteStatementQuery": true, "sqliteStatementClose": true, "sqliteStatementOpen": true, "sqliteStatementSQL": true,
	"sqliteTransactionExec": true, "sqliteTransactionQuery": true, "sqliteTransactionPrepare": true, "sqliteCommit": true, "sqliteRollback": true, "sqliteTransactionActive": true,
}

var z12NativeBuiltins = map[string]bool{
	"sqliteQueryOne":                true,
	"sqliteQueryStream":             true,
	"sqliteMigrate":                 true,
	"sqliteSchemaVersion":           true,
	"sqliteBackup":                  true,
	"sqliteRestore":                 true,
	"sqliteIntegrityCheck":          true,
	"sqliteStatementQueryStream":    true,
	"sqliteStatementParameterCount": true,
	"sqliteStatementColumns":        true,
	"sqliteTransactionQueryStream":  true,
	"sqliteSavepoint":               true,
	"sqliteRollbackTo":              true,
	"sqliteRelease":                 true,
	"sqlRowsNext":                   true,
	"sqlRowsColumns":                true,
	"sqlRowsClose":                  true,
	"sqlRowsOpen":                   true,
	"postgresOpen":                  true,
	"postgresConfigurePool":         true,
	"postgresPoolStats":             true,
	"postgresPing":                  true,
	"postgresClose":                 true,
	"postgresIsOpen":                true,
	"postgresExecDb":                true,
	"postgresQueryDb":               true,
	"postgresQueryOne":              true,
	"postgresQueryStream":           true,
	"postgresPrepare":               true,
	"postgresBegin":                 true,
	"postgresStatementExec":         true,
	"postgresStatementQuery":        true,
	"postgresStatementStream":       true,
	"postgresStatementClose":        true,
	"postgresStatementOpen":         true,
	"postgresStatementSQL":          true,
	"postgresTransactionExec":       true,
	"postgresTransactionQuery":      true,
	"postgresTransactionStream":     true,
	"postgresTransactionPrepare":    true,
	"postgresSavepoint":             true,
	"postgresRollbackTo":            true,
	"postgresRelease":               true,
	"postgresCommit":                true,
	"postgresRollback":              true,
	"postgresTransactionActive":     true,
	"redisOpen":                     true,
	"redisPing":                     true,
	"redisClose":                    true,
	"redisIsOpen":                   true,
	"redisSetClient":                true,
	"redisGetClient":                true,
	"redisDelete":                   true,
	"redisExists":                   true,
	"redisExpire":                   true,
	"redisTTL":                      true,
	"redisIncrement":                true,
	"redisPipeline":                 true,
	"redisPoolStats":                true,
	"configLoad":                    true,
	"configFrom":                    true,
	"configEnv":                     true,
	"configMerge":                   true,
	"configRequired":                true,
	"configString":                  true,
	"configInt":                     true,
	"configFloat":                   true,
	"configBool":                    true,
	"configSecret":                  true,
	"configRedacted":                true,
	"logger":                        true,
	"loggerWith":                    true,
	"loggerSetLevel":                true,
	"loggerLog":                     true,
	"loggerClose":                   true,
	"metrics":                       true,
	"metricsCounter":                true,
	"metricsGauge":                  true,
	"metricsHistogram":              true,
	"metricsSnapshot":               true,
	"metricsReset":                  true,
	"traceStart":                    true,
	"traceChild":                    true,
	"traceSet":                      true,
	"traceEvent":                    true,
	"traceFinish":                   true,
	"traceActive":                   true,
	"sessionSQLite":                 true,
	"sessionRedis":                  true,
	"sessionCreate":                 true,
	"sessionGet":                    true,
	"sessionSet":                    true,
	"sessionDelete":                 true,
	"sessionRotate":                 true,
	"sessionTouch":                  true,
	"sessionCleanup":                true,
	"sessionClose":                  true,
	"rateLimiter":                   true,
	"rateAllow":                     true,
	"rateReset":                     true,
	"fileExists":                    true,
	"jsonReadFile":                  true,
	"jsonWriteFile":                 true,
	"jsonReadResult":                true,
	"jsonWriteResult":               true,
	"csvReadFile":                   true,
	"csvWriteFile":                  true,
	"csvReadResult":                 true,
	"csvWriteResult":                true,
	"binaryEncode":                  true,
	"binaryDecode":                  true,
	"binaryWriteFile":               true,
	"binaryReadFile":                true,
}

var desktopNativeBuiltins = map[string]bool{
	"desktopApp": true, "desktopBackend": true, "desktopWindow": true, "desktopOn": true, "desktopShortcut": true,
	"desktopPoll": true, "desktopRun": true, "desktopQuit": true, "desktopRunning": true, "desktopClose": true, "desktopEmit": true,
	"desktopSetClipboard": true, "desktopClipboard": true, "desktopPickFile": true, "desktopPickFolder": true, "desktopNotify": true,
	"desktopPaths": true, "desktopOpenExternal": true, "desktopTray": true, "desktopTrayAdd": true, "desktopTrayTooltip": true,
	"desktopTrayClose": true, "desktopTrayOpen": true, "desktopSpawn": true, "desktopProcessWait": true, "desktopProcessKill": true,
	"desktopProcessRunning": true, "desktopProcessId": true, "desktopWindowShow": true, "desktopWindowHide": true,
	"desktopWindowClose": true, "desktopWindowOpen": true, "desktopWindowId": true, "desktopWindowTitle": true,
	"desktopWindowSetTitle": true, "desktopWindowSize": true, "desktopWindowPixelSize": true, "desktopWindowSetSize": true,
	"desktopWindowPosition": true, "desktopWindowSetPosition": true, "desktopWindowFullscreen": true,
	"desktopWindowSetFullscreen": true, "desktopWindowMaximize": true, "desktopWindowMinimize": true,
	"desktopWindowRestore": true, "desktopWindowFocus": true, "desktopWindowDisplayScale": true,
	"desktopWindowPixelDensity": true, "desktopWindowSetIcon": true,
}

var uiNativeBuiltins = map[string]bool{
	"uiTheme": true, "uiState": true, "uiStateGet": true, "uiStateSet": true, "uiStateSubscribe": true, "uiBind": true,
	"uiNode": true, "uiRow": true, "uiColumn": true, "uiContainer": true, "uiText": true, "uiButton": true,
	"uiInput": true, "uiTextarea": true, "uiSelect": true, "uiCheckbox": true, "uiRadio": true, "uiTable": true,
	"uiList": true, "uiTree": true, "uiTabs": true, "uiMenu": true, "uiModal": true, "uiTooltip": true,
	"uiProgress": true, "uiImage": true, "uiCanvas": true, "uiSpacer": true, "uiCustom": true,
	"uiMount": true, "uiUnmount": true, "uiRender": true, "uiSnapshot": true, "uiSetTheme": true, "uiDispatch": true,
	"uiSet": true, "uiGet": true, "uiAdd": true, "uiRemove": true, "uiFind": true, "uiFocus": true,
	"uiFocusNext": true, "uiAccessibility": true, "uiCanvasCommand": true,
}

func init() {
	for name := range z12NativeBuiltins {
		supportedBuiltins[name] = true
	}
	for name := range desktopNativeBuiltins {
		supportedBuiltins[name] = true
	}
	for name := range uiNativeBuiltins {
		supportedBuiltins[name] = true
	}
	for name := range assetNativeBuiltins {
		supportedBuiltins[name] = true
	}
}

type structInfo struct {
	ID      int
	Name    string
	Fields  []string
	Methods map[string]int
}

type enumMember struct {
	Name    string
	Ordinal int
}

type enumInfo struct {
	ID      int
	Name    string
	Members []enumMember
}

type generator struct {
	module *mir.Module
	out    bytes.Buffer
	indent int

	structs        []structInfo
	structByName   map[string]int
	enums          []enumInfo
	enumByName     map[string]int
	functions      map[*mir.Function]int
	functionByName map[string]int
	externs        []ffiInfo
	externByName   map[string]int
	globals        map[string]string

	scopes []map[string]string
	errs   []Diagnostic
}

func Generate(module *mir.Module) (*Sources, []Diagnostic) {
	if module == nil {
		return nil, []Diagnostic{{Message: "cannot generate native code from a nil MIR module"}}
	}
	g := &generator{
		module:         module,
		structByName:   map[string]int{},
		enumByName:     map[string]int{},
		functions:      map[*mir.Function]int{},
		functionByName: map[string]int{},
		externByName:   map[string]int{},
		globals:        map[string]string{},
	}
	g.collectMetadata()
	g.validateModule()
	if len(g.errs) != 0 {
		return nil, g.errs
	}
	g.emitProgram()
	if len(g.errs) != 0 {
		return nil, g.errs
	}
	runtimeSource, err := runtimeFiles.ReadFile("runtime/zumbra_runtime.c")
	if err != nil {
		return nil, []Diagnostic{{Message: "could not load embedded native runtime: " + err.Error()}}
	}
	prefix := ""
	if UsesAssets(module) {
		prefix += "#define ZUMBRA_ENABLE_ASSETS 1\n"
	}
	if UsesDesktop(module) || UsesUI(module) {
		prefix += "#define ZUMBRA_ENABLE_DESKTOP 1\n"
	}
	if UsesUI(module) {
		prefix += "#define ZUMBRA_ENABLE_UI 1\n"
	}
	if UsesNetwork(module) || UsesHTTP(module) {
		prefix += "#define ZUMBRA_ENABLE_NETWORK 1\n"
	}
	if UsesTLS(module) || UsesHTTP(module) {
		prefix += "#define ZUMBRA_ENABLE_TLS 1\n"
	}
	if UsesHTTP(module) {
		prefix += "#define ZUMBRA_ENABLE_HTTP 1\n"
	}
	if UsesZ12(module) {
		prefix += "#define ZUMBRA_ENABLE_Z12 1\n"
	}
	if UsesPostgres(module) {
		prefix += "#define ZUMBRA_ENABLE_POSTGRES 1\n"
	}
	if UsesRedis(module) {
		prefix += "#define ZUMBRA_ENABLE_REDIS 1\n"
	}
	if UsesSQLite(module) {
		prefix += "#define ZUMBRA_ENABLE_SQLITE 1\n"
	}
	if UsesDesktop(module) || UsesUI(module) {
		desktopRuntime, readErr := runtimeFiles.ReadFile("runtime/zumbra_desktop.inc")
		if readErr != nil {
			return nil, []Diagnostic{{Message: "could not load embedded desktop runtime: " + readErr.Error()}}
		}
		runtimeSource = append(runtimeSource, '\n')
		runtimeSource = append(runtimeSource, desktopRuntime...)
	}
	if UsesUI(module) {
		uiRuntime, readErr := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
		if readErr != nil {
			return nil, []Diagnostic{{Message: "could not load embedded UI runtime: " + readErr.Error()}}
		}
		runtimeSource = append(runtimeSource, '\n')
		runtimeSource = append(runtimeSource, uiRuntime...)
	}
	if UsesHTTP(module) {
		httpRuntime, readErr := runtimeFiles.ReadFile("runtime/zumbra_http.inc")
		if readErr != nil {
			return nil, []Diagnostic{{Message: "could not load embedded HTTP runtime: " + readErr.Error()}}
		}
		runtimeSource = append(runtimeSource, '\n')
		runtimeSource = append(runtimeSource, httpRuntime...)
	}
	if UsesZ12(module) {
		z12Runtime, readErr := runtimeFiles.ReadFile("runtime/zumbra_z12.inc")
		if readErr != nil {
			return nil, []Diagnostic{{Message: "could not load embedded Z12 runtime: " + readErr.Error()}}
		}
		runtimeSource = append(runtimeSource, '\n')
		runtimeSource = append(runtimeSource, z12Runtime...)
	}
	if UsesSQLite(module) {
		sqliteRuntime, readErr := runtimeFiles.ReadFile("runtime/zumbra_sqlite.inc")
		if readErr != nil {
			return nil, []Diagnostic{{Message: "could not load embedded SQLite runtime: " + readErr.Error()}}
		}
		runtimeSource = append(runtimeSource, '\n')
		runtimeSource = append(runtimeSource, sqliteRuntime...)
	}
	if prefix != "" {
		runtimeSource = append([]byte(prefix), runtimeSource...)
	}
	header, err := runtimeFiles.ReadFile("runtime/zumbra_runtime.h")
	if err != nil {
		return nil, []Diagnostic{{Message: "could not load embedded native runtime header: " + err.Error()}}
	}
	return &Sources{Program: g.out.Bytes(), Runtime: runtimeSource, Header: header}, nil
}

func (g *generator) collectMetadata() {
	for _, declaration := range g.module.Declarations {
		switch declaration.Op {
		case mir.OpExtern:
			info, err := externFromInstruction(len(g.externs), declaration)
			if err != nil {
				g.errs = append(g.errs, Diagnostic{Instruction: declaration.ID, Message: err.Error()})
				continue
			}
			g.externByName[info.Name] = info.ID
			g.externs = append(g.externs, info)
		case mir.OpStruct:
			info := structInfo{ID: len(g.structs), Name: declaration.Name, Methods: map[string]int{}}
			for _, region := range declaration.Regions {
				if len(region.Instructions) == 1 && region.Instructions[0].Op == mir.OpStructField {
					info.Fields = append(info.Fields, region.Instructions[0].Name)
				}
			}
			g.structByName[info.Name] = info.ID
			g.structs = append(g.structs, info)
		case mir.OpEnum:
			info := enumInfo{ID: len(g.enums), Name: declaration.Name}
			for name, raw := range declaration.Meta {
				ordinal, _ := strconv.Atoi(raw)
				info.Members = append(info.Members, enumMember{Name: name, Ordinal: ordinal})
			}
			sort.Slice(info.Members, func(i, j int) bool { return info.Members[i].Ordinal < info.Members[j].Ordinal })
			g.enumByName[info.Name] = info.ID
			g.enums = append(g.enums, info)
		}
	}
	for index, function := range g.module.Functions {
		g.functions[function] = index
		if function.Owner == "" && function.Name != "" && !strings.HasPrefix(function.Name, "lambda.") {
			g.functionByName[function.Name] = index
		}
	}
	for _, function := range g.module.Functions {
		if function.Owner == "" {
			continue
		}
		structID, ok := g.structByName[function.Owner]
		if !ok {
			g.errs = append(g.errs, Diagnostic{Message: fmt.Sprintf("method %s.%s references unknown struct", function.Owner, function.Name)})
			continue
		}
		g.structs[structID].Methods[function.Name] = g.functions[function]
	}
	for _, instruction := range g.module.Entry.Instructions {
		if instruction.Op == mir.OpDeclare {
			g.globals[instruction.Name] = "zg_" + sanitize(instruction.Name)
		}
	}
}

func (g *generator) validateModule() {
	for _, declaration := range g.module.Declarations {
		switch declaration.Op {
		case mir.OpImport:
			g.errs = append(g.errs, Diagnostic{Instruction: declaration.ID, Message: "unresolved import reached the Z8 native backend"})
		case mir.OpExtern:
			if declaration.Meta["abi"] != "C" {
				g.errs = append(g.errs, Diagnostic{Instruction: declaration.ID, Message: "only extern C is supported"})
			}
		}
	}
	for _, function := range g.module.Functions {
		g.validateRegion(function.Body)
	}
	g.validateRegion(g.module.Entry)
}

func (g *generator) validateRegion(region *mir.Region) {
	if region == nil {
		return
	}
	for _, instruction := range region.Instructions {
		switch instruction.Op {
		case mir.OpTry, mir.OpHandler:
			g.errs = append(g.errs, Diagnostic{Instruction: instruction.ID, Message: fmt.Sprintf("%s is not supported by the first native backend", instruction.Op)})
		case mir.OpUnknown:
			g.errs = append(g.errs, Diagnostic{Instruction: instruction.ID, Message: "unknown MIR operation cannot be compiled natively"})
		case mir.OpImport:
			g.errs = append(g.errs, Diagnostic{Instruction: instruction.ID, Message: "imports require Z8"})
		}
		for _, child := range instruction.Regions {
			g.validateRegion(child)
		}
	}
}

func (g *generator) emitProgram() {
	g.line("/* Generated from Zumbra MIR. Do not edit manually. */")
	g.line("#include \"zumbra_runtime.h\"")
	g.line("#include <stdio.h>")
	g.line("#include <string.h>")
	g.line("_Static_assert(ZUMBRA_NATIVE_ABI_VERSION == 6u, \"unsupported Zumbra native ABI\");")
	g.line("")
	g.emitFFIDeclarations()
	for _, name := range sortedKeys(g.globals) {
		g.line("static ZValue %s;", g.globals[name])
	}
	if len(g.globals) != 0 {
		g.line("")
	}
	for index := range g.module.Functions {
		g.line("static ZValue zf_%d(const ZValue *args, size_t argc);", index)
	}
	if len(g.module.Functions) != 0 {
		g.line("")
	}
	g.emitMetadataFunctions()
	for index, function := range g.module.Functions {
		g.emitFunction(index, function)
	}
	g.emitDispatch()
	g.emitMain()
}

func (g *generator) emitMetadataFunctions() {
	g.line("const char *z_struct_type_name(int type_id) {")
	g.indent++
	g.line("switch (type_id) {")
	g.indent++
	for _, info := range g.structs {
		g.line("case %d: return %s;", info.ID, cString(info.Name))
	}
	g.line("default: return \"<unknown-struct>\";")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")

	g.line("const char *z_enum_type_name(int type_id) {")
	g.indent++
	g.line("switch (type_id) {")
	g.indent++
	for _, info := range g.enums {
		g.line("case %d: return %s;", info.ID, cString(info.Name))
	}
	g.line("default: return \"<unknown-enum>\";")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")

	g.line("const char *z_enum_member_name(int type_id, int ordinal) {")
	g.indent++
	g.line("switch (type_id) {")
	g.indent++
	for _, info := range g.enums {
		g.line("case %d:", info.ID)
		g.indent++
		g.line("switch (ordinal) {")
		g.indent++
		for _, member := range info.Members {
			g.line("case %d: return %s;", member.Ordinal, cString(member.Name))
		}
		g.line("default: return \"<unknown-member>\";")
		g.indent--
		g.line("}")
		g.indent--
	}
	g.line("default: return \"<unknown-member>\";")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")

	g.line("int z_enum_member_ordinal(int type_id, const char *name) {")
	g.indent++
	g.line("switch (type_id) {")
	g.indent++
	for _, info := range g.enums {
		g.line("case %d:", info.ID)
		g.indent++
		for _, member := range info.Members {
			g.line("if (strcmp(name, %s) == 0) return %d;", cString(member.Name), member.Ordinal)
		}
		g.line("return -1;")
		g.indent--
	}
	g.line("default: return -1;")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")

	g.line("int z_struct_field_index(int type_id, const char *name) {")
	g.indent++
	g.line("switch (type_id) {")
	g.indent++
	for _, info := range g.structs {
		g.line("case %d:", info.ID)
		g.indent++
		for index, field := range info.Fields {
			g.line("if (strcmp(name, %s) == 0) return %d;", cString(field), index)
		}
		g.line("return -1;")
		g.indent--
	}
	g.line("default: return -1;")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")

	g.line("int z_struct_method_id(int type_id, const char *name) {")
	g.indent++
	g.line("switch (type_id) {")
	g.indent++
	for _, info := range g.structs {
		g.line("case %d:", info.ID)
		g.indent++
		for _, method := range sortedIntMapKeys(info.Methods) {
			g.line("if (strcmp(name, %s) == 0) return %d;", cString(method), info.Methods[method])
		}
		g.line("return -1;")
		g.indent--
	}
	g.line("default: return -1;")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")

	g.line("ZValue z_construct_struct(int type_id, const ZValue *args, size_t argc) {")
	g.indent++
	g.line("switch (type_id) {")
	g.indent++
	for _, info := range g.structs {
		g.line("case %d: {", info.ID)
		g.indent++
		g.line("ZStruct *instance = (ZStruct *)z_alloc(sizeof(ZStruct));")
		g.line("instance->type_id = %d;", info.ID)
		g.line("instance->field_count = %d;", len(info.Fields))
		if len(info.Fields) == 0 {
			g.line("instance->fields = NULL;")
		} else {
			g.line("instance->fields = (ZValue *)z_alloc(sizeof(ZValue) * %d);", len(info.Fields))
			g.line("if (argc == 1 && args[0].tag == ZV_DICT) {")
			g.indent++
			g.line("for (size_t i = 0; i < %d; i++) instance->fields[i] = z_null();", len(info.Fields))
			g.line("for (size_t i = 0; i < args[0].as.dict->len; i++) {")
			g.indent++
			g.line("if (args[0].as.dict->keys[i].tag != ZV_STRING) z_fatal(\"named struct field must be a string\");")
			g.line("int field = z_struct_field_index(%d, args[0].as.dict->keys[i].as.s);", info.ID)
			g.line("if (field < 0) z_fatal(\"unknown field %%s for %s\", args[0].as.dict->keys[i].as.s);", info.Name)
			g.line("instance->fields[field] = args[0].as.dict->values[i];")
			g.indent--
			g.line("}")
			g.line("for (size_t i = 0; i < %d; i++) if (instance->fields[i].tag == ZV_NULL) z_fatal(\"missing named field for %s\");", len(info.Fields), info.Name)
			g.indent--
			g.line("} else {")
			g.indent++
			g.line("if (argc != %d) z_fatal(\"%s expects %d fields, got %%zu\", argc);", len(info.Fields), info.Name, len(info.Fields))
			g.line("memcpy(instance->fields, args, sizeof(ZValue) * %d);", len(info.Fields))
			g.indent--
			g.line("}")
		}
		g.line("ZValue result = z_null();")
		g.line("result.tag = ZV_STRUCT;")
		g.line("result.kind = ZK_STRUCT;")
		g.line("result.as.structure = instance;")
		g.line("return result;")
		g.indent--
		g.line("}")
	}
	g.line("default: z_fatal(\"unknown struct type id %%d\", type_id); return z_null();")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")
}

func (g *generator) emitFunction(index int, function *mir.Function) {
	g.line("static ZValue zf_%d(const ZValue *args, size_t argc) {", index)
	g.indent++
	g.pushScope()
	g.line("if (argc != %d) z_fatal(%s, argc);", len(function.Parameters), cString(fmt.Sprintf("%s expects %d arguments, got %%zu", qualifiedFunctionName(function), len(function.Parameters))))
	for parameterIndex, parameter := range function.Parameters {
		binding := fmt.Sprintf("zp_%s_%d", sanitize(parameter), parameterIndex)
		g.line("ZValue %s = args[%d];", binding, parameterIndex)
		g.bind(parameter, binding)
	}
	g.emitRegion(function.Body, false, "")
	if function.Body != nil && function.Body.Result != 0 {
		g.line("return %s;", valueName(function.Body.Result))
	} else {
		g.line("return z_null();")
	}
	g.popScope()
	g.indent--
	g.line("}")
	g.line("")
}

func (g *generator) emitDispatch() {
	g.line("ZValue z_dispatch_function(int function_id, const ZValue *args, size_t argc) {")
	g.indent++
	g.line("switch (function_id) {")
	g.indent++
	for index := range g.module.Functions {
		g.line("case %d: return zf_%d(args, argc);", index, index)
	}
	for index := range g.externs {
		g.line("case %d: return zffi_%d(args, argc);", len(g.module.Functions)+index, index)
	}
	g.line("default: z_fatal(\"unknown function id %%d\", function_id); return z_null();")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")
	g.line("bool z_function_is_async(int function_id) {")
	g.indent++
	g.line("switch (function_id) {")
	g.indent++
	for index, function := range g.module.Functions {
		if function.Async {
			g.line("case %d: return true;", index)
		}
	}
	g.line("default: return false;")
	g.indent--
	g.line("}")
	g.indent--
	g.line("}")
	g.line("")

}

func (g *generator) emitMain() {
	g.line("int main(void) {")
	g.indent++
	g.line("z_runtime_init();")
	for _, name := range sortedKeys(g.globals) {
		g.line("%s = z_null();", g.globals[name])
	}
	g.pushScope()
	g.emitRegion(g.module.Entry, true, "")
	g.popScope()
	g.line("z_runtime_shutdown();")
	g.line("return 0;")
	g.indent--
	g.line("}")
}

func (g *generator) emitRegion(region *mir.Region, rootEntry bool, resultTarget string) {
	if region == nil {
		return
	}
	if !rootEntry {
		g.pushScope()
	}
	for _, instruction := range region.Instructions {
		g.emitInstruction(instruction, rootEntry)
	}
	if resultTarget != "" && region.Result != 0 {
		g.line("%s = %s;", resultTarget, valueName(region.Result))
	}
	if !rootEntry {
		g.popScope()
	}
}

func (g *generator) emitInstruction(instruction *mir.Instruction, rootEntry bool) {
	if instruction == nil {
		return
	}
	result := valueName(instruction.Result)
	switch instruction.Op {
	case mir.OpConst:
		g.line("ZValue %s = %s;", result, constExpression(instruction))
	case mir.OpLoad:
		expression, ok := g.resolveLoad(instruction.Name)
		if !ok {
			message := fmt.Sprintf("native backend cannot resolve symbol %q; captured local closures are not supported yet", instruction.Name)
			if builtinspec.Contains(instruction.Name) {
				message = fmt.Sprintf("builtin %q is not available in the Z7 native runtime; use the VM or wait for its native module", instruction.Name)
			}
			g.errs = append(g.errs, Diagnostic{Instruction: instruction.ID, Message: message})
			expression = "z_null()"
		}
		g.line("ZValue %s = %s;", result, expression)
	case mir.OpDeclare:
		value := argName(instruction.Args, 0)
		if rootEntry {
			binding := g.globals[instruction.Name]
			g.line("%s = %s;", binding, value)
			g.bind(instruction.Name, binding)
		} else {
			binding := fmt.Sprintf("zl_%s_%d", sanitize(instruction.Name), instruction.ID)
			g.line("ZValue %s = %s;", binding, value)
			g.bind(instruction.Name, binding)
		}
	case mir.OpStore:
		binding, ok := g.resolveBinding(instruction.Name)
		if !ok {
			g.errs = append(g.errs, Diagnostic{Instruction: instruction.ID, Message: fmt.Sprintf("assignment target %q is not in scope", instruction.Name)})
			return
		}
		g.line("%s = %s;", binding, argName(instruction.Args, 0))
	case mir.OpUnary:
		g.line("ZValue %s = z_unary(%s, %s, %s);", result, cString(instruction.Operator), argName(instruction.Args, 0), cKind(instruction.Type))
	case mir.OpBinary:
		g.line("ZValue %s = z_binary(%s, %s, %s, %s);", result, cString(instruction.Operator), argName(instruction.Args, 0), argName(instruction.Args, 1), cKind(instruction.Type))
	case mir.OpFunctionRef:
		functionID := g.functionIDForRef(instruction)
		if functionID < 0 {
			g.errs = append(g.errs, Diagnostic{Instruction: instruction.ID, Message: fmt.Sprintf("function reference %q was not found", instruction.Name)})
			g.line("ZValue %s = z_null();", result)
		} else {
			g.line("ZValue %s = z_function(%d);", result, functionID)
		}
	case mir.OpSpawn:
		g.emitValueArray("zs", instruction.ID, instruction.Args[1:])
		g.line("ZValue %s = z_spawn(%s, %s, %d);", result, argName(instruction.Args, 0), arrayNameOrNull("zs", instruction.ID, len(instruction.Args)-1), max(0, len(instruction.Args)-1))
	case mir.OpAwait:
		g.line("ZValue %s = z_task_await(%s);", result, argName(instruction.Args, 0))
	case mir.OpCall:
		g.emitValueArray("za", instruction.ID, instruction.Args[1:])
		g.line("ZValue %s = z_call(%s, %s, %d);", result, argName(instruction.Args, 0), arrayNameOrNull("za", instruction.ID, len(instruction.Args)-1), max(0, len(instruction.Args)-1))
	case mir.OpArray:
		g.emitValueArray("za", instruction.ID, instruction.Args)
		g.line("ZValue %s = z_array_from(%s, %d);", result, arrayNameOrNull("za", instruction.ID, len(instruction.Args)), len(instruction.Args))
	case mir.OpPair:
		g.line("ZValue %s = z_pair(%s, %s);", result, argName(instruction.Args, 0), argName(instruction.Args, 1))
	case mir.OpDict:
		g.emitValueArray("zd", instruction.ID, instruction.Args)
		g.line("ZValue %s = z_dict_from(%s, %d);", result, arrayNameOrNull("zd", instruction.ID, len(instruction.Args)), len(instruction.Args))
	case mir.OpIndex:
		g.line("ZValue %s = z_index(%s, %s);", result, argName(instruction.Args, 0), argName(instruction.Args, 1))
	case mir.OpSetIndex:
		g.line("z_set_index(%s, %s, %s);", argName(instruction.Args, 0), argName(instruction.Args, 1), argName(instruction.Args, 2))
	case mir.OpField:
		g.line("ZValue %s = z_get_field(%s, %s);", result, argName(instruction.Args, 0), cString(instruction.Name))
	case mir.OpSetField:
		g.line("z_set_field(%s, %s, %s);", argName(instruction.Args, 0), cString(instruction.Name), argName(instruction.Args, 1))
	case mir.OpIf:
		g.line("ZValue %s = z_null();", result)
		g.line("if (z_truthy(%s)) {", argName(instruction.Args, 0))
		g.indent++
		if len(instruction.Regions) > 0 {
			g.emitRegion(instruction.Regions[0], false, result)
		}
		g.indent--
		if len(instruction.Regions) > 1 {
			g.line("} else {")
			g.indent++
			g.emitRegion(instruction.Regions[1], false, result)
			g.indent--
		}
		g.line("}")
		// Some control-flow expressions are used only for side effects. Keeping an
		// explicit read prevents -Wunused-but-set-variable under release -Werror,
		// while preserving the value for expressions that are consumed later.
		g.line("(void)%s;", result)
	case mir.OpMatch:
		g.emitMatch(instruction)
	case mir.OpWhile:
		g.line("while (true) {")
		g.indent++
		if len(instruction.Regions) > 0 {
			g.line("ZValue z_condition_%d = z_null();", instruction.ID)
			g.emitRegion(instruction.Regions[0], false, fmt.Sprintf("z_condition_%d", instruction.ID))
			g.line("if (!z_truthy(z_condition_%d)) break;", instruction.ID)
		}
		if len(instruction.Regions) > 1 {
			g.emitRegion(instruction.Regions[1], false, "")
		}
		g.indent--
		g.line("}")
	case mir.OpFor:
		g.emitFor(instruction)
	case mir.OpForEach:
		g.emitForEach(instruction)
	case mir.OpForRange:
		g.emitForRange(instruction)
	case mir.OpForever:
		g.line("while (true) {")
		g.indent++
		if len(instruction.Regions) > 0 {
			g.emitRegion(instruction.Regions[0], false, "")
		}
		g.indent--
		g.line("}")
	case mir.OpReturn:
		if len(instruction.Args) == 0 {
			g.line("return z_null();")
		} else {
			g.line("return %s;", argName(instruction.Args, 0))
		}
	case mir.OpBreak:
		g.line("break;")
	case mir.OpContinue:
		g.line("continue;")
	case mir.OpDrop:
		if len(instruction.Args) > 0 {
			g.line("(void)%s;", argName(instruction.Args, 0))
		}
	case mir.OpUnsafe:
		if len(instruction.Regions) > 0 {
			g.emitRegion(instruction.Regions[0], false, "")
		}
	case mir.OpTypeAlias, mir.OpStruct, mir.OpStructField, mir.OpEnum, mir.OpImport, mir.OpExtern:
		// Declarations are emitted through metadata tables.
	default:
		g.errs = append(g.errs, Diagnostic{Instruction: instruction.ID, Message: fmt.Sprintf("native code emission for %s is not implemented", instruction.Op)})
	}
}

func (g *generator) emitMatch(instruction *mir.Instruction) {
	result := valueName(instruction.Result)
	g.line("ZValue %s = z_null();", result)
	g.line("bool zm_%d = false;", instruction.ID)
	for _, caseRegion := range instruction.Regions {
		if caseRegion == nil || len(caseRegion.Instructions) == 0 {
			continue
		}
		caseInstruction := caseRegion.Instructions[len(caseRegion.Instructions)-1]
		g.line("{")
		g.indent++
		g.pushScope()
		for _, pre := range caseRegion.Instructions[:len(caseRegion.Instructions)-1] {
			g.emitInstruction(pre, false)
		}
		if caseInstruction.Name == "else" {
			g.line("if (!zm_%d) {", instruction.ID)
		} else {
			g.line("if (!zm_%d && z_equal(%s, %s)) {", instruction.ID, argName(instruction.Args, 0), argName(caseInstruction.Args, 0))
		}
		g.indent++
		g.line("zm_%d = true;", instruction.ID)
		if len(caseInstruction.Regions) > 0 {
			g.emitRegion(caseInstruction.Regions[0], false, result)
		}
		g.indent--
		g.line("}")
		g.popScope()
		g.indent--
		g.line("}")
	}
	g.line("(void)%s;", result)
}

func (g *generator) emitFor(instruction *mir.Instruction) {
	g.line("{")
	g.indent++
	g.pushScope()
	if len(instruction.Regions) > 0 {
		g.emitRegion(instruction.Regions[0], false, "")
	}
	g.line("while (true) {")
	g.indent++
	if len(instruction.Regions) > 1 {
		g.line("ZValue z_condition_%d = z_bool(true);", instruction.ID)
		g.emitRegion(instruction.Regions[1], false, fmt.Sprintf("z_condition_%d", instruction.ID))
		g.line("if (!z_truthy(z_condition_%d)) break;", instruction.ID)
	}
	if len(instruction.Regions) > 3 {
		g.emitRegion(instruction.Regions[3], false, "")
	}
	if len(instruction.Regions) > 2 {
		g.emitRegion(instruction.Regions[2], false, "")
	}
	g.indent--
	g.line("}")
	g.popScope()
	g.indent--
	g.line("}")
}

func (g *generator) emitForEach(instruction *mir.Instruction) {
	collection := argName(instruction.Args, 0)
	parts := strings.Split(instruction.Name, ",")
	g.line("{")
	g.indent++
	g.pushScope()
	g.line("size_t z_count_%d = z_size_of(%s);", instruction.ID, collection)
	g.line("for (size_t z_i_%d = 0; z_i_%d < z_count_%d; z_i_%d++) {", instruction.ID, instruction.ID, instruction.ID, instruction.ID)
	g.indent++
	if instruction.Meta["source_kind"] == "map" && len(parts) == 2 {
		g.line("if (%s.tag != ZV_DICT) z_fatal(\"map iteration expects a dictionary\");", collection)
		keyBinding := fmt.Sprintf("zl_%s_%d_key", sanitize(parts[0]), instruction.ID)
		valueBinding := fmt.Sprintf("zl_%s_%d_value", sanitize(parts[1]), instruction.ID)
		g.line("ZValue %s = %s.as.dict->keys[z_i_%d];", keyBinding, collection, instruction.ID)
		g.line("ZValue %s = %s.as.dict->values[z_i_%d];", valueBinding, collection, instruction.ID)
		g.bind(parts[0], keyBinding)
		g.bind(parts[1], valueBinding)
	} else {
		name := instruction.Name
		binding := fmt.Sprintf("zl_%s_%d", sanitize(name), instruction.ID)
		g.line("ZValue %s = z_index(%s, z_int((int64_t)z_i_%d));", binding, collection, instruction.ID)
		g.bind(name, binding)
	}
	bodyIndex := len(instruction.Regions) - 1
	if len(instruction.Regions) == 2 {
		g.line("ZValue z_where_%d = z_bool(false);", instruction.ID)
		g.emitRegion(instruction.Regions[0], false, fmt.Sprintf("z_where_%d", instruction.ID))
		g.line("if (!z_truthy(z_where_%d)) continue;", instruction.ID)
	}
	if bodyIndex >= 0 {
		g.emitRegion(instruction.Regions[bodyIndex], false, "")
	}
	g.indent--
	g.line("}")
	g.popScope()
	g.indent--
	g.line("}")
}

func (g *generator) emitForRange(instruction *mir.Instruction) {
	g.line("{")
	g.indent++
	g.pushScope()
	binding := fmt.Sprintf("zl_%s_%d", sanitize(instruction.Name), instruction.ID)
	g.line("for (int64_t z_i_%d = z_convert(%s, ZK_INT).as.i, z_end_%d = z_convert(%s, ZK_INT).as.i; z_i_%d <= z_end_%d; z_i_%d++) {", instruction.ID, argName(instruction.Args, 0), instruction.ID, argName(instruction.Args, 1), instruction.ID, instruction.ID, instruction.ID)
	g.indent++
	g.line("ZValue %s = z_int(z_i_%d);", binding, instruction.ID)
	g.bind(instruction.Name, binding)
	bodyIndex := len(instruction.Regions) - 1
	if len(instruction.Regions) == 2 {
		g.line("ZValue z_where_%d = z_bool(false);", instruction.ID)
		g.emitRegion(instruction.Regions[0], false, fmt.Sprintf("z_where_%d", instruction.ID))
		g.line("if (!z_truthy(z_where_%d)) continue;", instruction.ID)
	}
	if bodyIndex >= 0 {
		g.emitRegion(instruction.Regions[bodyIndex], false, "")
	}
	g.indent--
	g.line("}")
	g.popScope()
	g.indent--
	g.line("}")
}

func (g *generator) emitValueArray(prefix string, instructionID int, values []mir.ValueID) {
	if len(values) == 0 {
		return
	}
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = valueName(value)
	}
	g.line("ZValue %s_%d[] = {%s};", prefix, instructionID, strings.Join(parts, ", "))
}

func (g *generator) resolveLoad(name string) (string, bool) {
	if binding, ok := g.resolveBinding(name); ok {
		return binding, true
	}
	if id, ok := g.structByName[name]; ok {
		return fmt.Sprintf("z_struct_type(%d)", id), true
	}
	if id, ok := g.enumByName[name]; ok {
		return fmt.Sprintf("z_enum_type(%d)", id), true
	}
	if id, ok := g.functionByName[name]; ok {
		return fmt.Sprintf("z_function(%d)", id), true
	}
	if id, ok := g.externByName[name]; ok {
		return fmt.Sprintf("z_function(%d)", len(g.module.Functions)+id), true
	}
	if supportedBuiltins[name] {
		return fmt.Sprintf("z_builtin(%s)", cString(name)), true
	}
	if name == "null" {
		return "z_null()", true
	}
	return "", false
}

func (g *generator) functionIDForRef(instruction *mir.Instruction) int {
	for function, id := range g.functions {
		if function.Name == instruction.Name {
			return id
		}
	}
	return -1
}

func (g *generator) pushScope() { g.scopes = append(g.scopes, map[string]string{}) }
func (g *generator) popScope() {
	if len(g.scopes) != 0 {
		g.scopes = g.scopes[:len(g.scopes)-1]
	}
}
func (g *generator) bind(name, value string) {
	if len(g.scopes) == 0 {
		g.pushScope()
	}
	g.scopes[len(g.scopes)-1][name] = value
}
func (g *generator) resolveBinding(name string) (string, bool) {
	for index := len(g.scopes) - 1; index >= 0; index-- {
		if value, ok := g.scopes[index][name]; ok {
			return value, true
		}
	}
	value, ok := g.globals[name]
	return value, ok
}

func (g *generator) line(format string, args ...any) {
	g.out.WriteString(strings.Repeat("    ", g.indent))
	g.out.WriteString(fmt.Sprintf(format, args...))
	g.out.WriteByte('\n')
}

func constExpression(instruction *mir.Instruction) string {
	kind := types.Unknown
	if instruction.Type != nil {
		kind = instruction.Type.Kind
	}
	// The literal's source category is more precise than an unknown contextual
	// type. This commonly occurs inside loops over dynamically shaped rows.
	// Preserve string/bool/float emission rather than falling through to int.
	if kind == types.Unknown && instruction.Meta != nil {
		switch instruction.Meta["literal_kind"] {
		case string(hir.StringKind):
			kind = types.String
		case string(hir.BooleanKind):
			kind = types.Bool
		case string(hir.FloatKind):
			kind = types.Float
		case string(hir.IntegerKind):
			kind = types.Int
		}
	}
	switch kind {
	case types.String:
		return "z_string(" + cString(instruction.Literal) + ")"
	case types.Bool:
		return "z_bool(" + instruction.Literal + ")"
	case types.Float:
		return "z_float(" + instruction.Literal + ")"
	case types.U8, types.U16, types.U32, types.U64:
		return fmt.Sprintf("z_uint(UINT64_C(%s), %s)", stripFixedSuffix(instruction.Literal), cKind(instruction.Type))
	case types.I8, types.I16, types.I32, types.I64:
		return fmt.Sprintf("z_signed(INT64_C(%s), %s)", stripFixedSuffix(instruction.Literal), cKind(instruction.Type))
	default:
		return fmt.Sprintf("z_int(INT64_C(%s))", stripFixedSuffix(instruction.Literal))
	}
}

func cKind(value *types.Type) string {
	if value == nil {
		return "ZK_UNKNOWN"
	}
	switch value.Kind {
	case types.Int:
		return "ZK_INT"
	case types.U8:
		return "ZK_U8"
	case types.U16:
		return "ZK_U16"
	case types.U32:
		return "ZK_U32"
	case types.U64:
		return "ZK_U64"
	case types.I8:
		return "ZK_I8"
	case types.I16:
		return "ZK_I16"
	case types.I32:
		return "ZK_I32"
	case types.I64:
		return "ZK_I64"
	case types.Float:
		return "ZK_FLOAT"
	case types.Bool:
		return "ZK_BOOL"
	case types.String:
		return "ZK_STRING"
	case types.Null:
		return "ZK_NULL"
	case types.Array:
		return "ZK_ARRAY"
	case types.ByteArray:
		return "ZK_BYTE_ARRAY"
	case types.TypedArray:
		return "ZK_TYPED_ARRAY"
	case types.Slice:
		return "ZK_SLICE"
	case types.Dict:
		return "ZK_DICT"
	case types.Struct:
		return "ZK_STRUCT"
	case types.Enum:
		return "ZK_ENUM"
	case types.Func:
		return "ZK_FUNCTION"
	case types.Pointer:
		return "ZK_POINTER"
	case types.Task:
		return "ZK_TASK"
	case types.Channel:
		return "ZK_CHANNEL"
	case types.Mutex:
		return "ZK_MUTEX"
	case types.RWMutex:
		return "ZK_RW_MUTEX"
	case types.WaitGroup:
		return "ZK_WAIT_GROUP"
	case types.Semaphore:
		return "ZK_SEMAPHORE"
	case types.AtomicInt:
		return "ZK_ATOMIC_INT"
	case types.SQLiteDatabase:
		return "ZK_SQLITE_DATABASE"
	case types.SQLiteStatement:
		return "ZK_SQLITE_STATEMENT"
	case types.SQLiteTransaction:
		return "ZK_SQLITE_TRANSACTION"
	case types.SQLRows:
		return "ZK_SQL_ROWS"
	case types.PostgresDatabase:
		return "ZK_POSTGRES_DATABASE"
	case types.PostgresStatement:
		return "ZK_POSTGRES_STATEMENT"
	case types.PostgresTransaction:
		return "ZK_POSTGRES_TRANSACTION"
	case types.RedisClient:
		return "ZK_REDIS_CLIENT"
	case types.Config:
		return "ZK_CONFIG"
	case types.Logger:
		return "ZK_LOGGER"
	case types.MetricsRegistry:
		return "ZK_METRICS"
	case types.TraceSpan:
		return "ZK_TRACE_SPAN"
	case types.SessionStore:
		return "ZK_SESSION_STORE"
	case types.RateLimiter:
		return "ZK_RATE_LIMITER"
	default:
		return "ZK_UNKNOWN"
	}
}

func stripFixedSuffix(value string) string {
	for _, suffix := range []string{"u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64"} {
		if strings.HasSuffix(value, suffix) {
			return strings.TrimSuffix(value, suffix)
		}
	}
	return value
}
func valueName(id mir.ValueID) string { return fmt.Sprintf("zv_%d", id) }
func argName(values []mir.ValueID, index int) string {
	if index < 0 || index >= len(values) {
		return "z_null()"
	}
	return valueName(values[index])
}
func arrayNameOrNull(prefix string, id, count int) string {
	if count == 0 {
		return "NULL"
	}
	return fmt.Sprintf("%s_%d", prefix, id)
}
func cString(value string) string { return strconv.Quote(value) }
func sanitize(value string) string {
	if value == "" {
		return "anonymous"
	}
	var out strings.Builder
	for index, r := range value {
		if unicode.IsLetter(r) || r == '_' || (index > 0 && unicode.IsDigit(r)) {
			out.WriteRune(r)
		} else {
			out.WriteRune('_')
		}
	}
	return out.String()
}
func qualifiedFunctionName(function *mir.Function) string {
	if function.Owner != "" {
		return function.Owner + "." + function.Name
	}
	return function.Name
}
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func sortedIntMapKeys(values map[string]int) []string { return sortedKeys(values) }
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var networkBuiltins = map[string]bool{
	"tcpListen": true, "tcpConnect": true, "tcpConnectTimeout": true,
	"tlsListen": true, "tlsConnect": true, "tlsConnectTimeout": true,
	"listenerAccept": true, "listenerAcceptTimeout": true, "listenerClose": true, "listenerClosed": true, "listenerAddress": true, "listenerPort": true,
	"streamRead": true, "streamReadExact": true, "streamReadTimeout": true, "streamWrite": true, "streamWriteAll": true,
	"streamClose": true, "streamClosed": true, "streamShutdownRead": true, "streamShutdownWrite": true,
	"streamLocalAddress": true, "streamLocalPort": true, "streamRemoteAddress": true, "streamRemotePort": true,
	"streamSetReadTimeout": true, "streamSetWriteTimeout": true, "tcpSetKeepAlive": true,
	"dnsLookup": true, "dnsLookupTimeout": true,
	"udpBind": true, "udpSendTo": true, "udpReceiveFrom": true, "udpReceiveFromTimeout": true,
	"udpClose": true, "udpClosed": true, "udpAddress": true, "udpPort": true,
}

var tlsBuiltins = map[string]bool{
	"tlsListen": true, "tlsConnect": true, "tlsConnectTimeout": true,
}

// UsesNetwork reports whether the MIR references any Z10 networking builtin.
func UsesNetwork(module *mir.Module) bool { return moduleUsesAnyBuiltin(module, networkBuiltins) }

var httpBuiltins = map[string]bool{
	"httpApp": true, "httpRoute": true, "httpUse": true, "httpStatic": true, "httpLimitBody": true, "httpCompression": true, "httpCors": true,
	"httpServe": true, "httpServeTLS": true, "httpShutdown": true, "httpServerPort": true, "httpServerAddress": true, "httpServerRunning": true,
	"httpText": true, "httpJson": true, "httpHtml": true, "httpRedirect": true, "httpFile": true, "httpHeader": true, "httpCookie": true,
	"httpStream": true, "httpSSE": true, "sseEvent": true, "httpRequest": true, "httpStatus": true, "httpBody": true, "httpBodyBytes": true, "httpBodyJSON": true, "httpHeaders": true,
	"jsonStringify": true, "jsonParse": true, "jwtSignHS256": true, "jwtVerifyHS256": true,
	"webSocketUpgrade": true, "webSocketConnect": true, "webSocketRead": true, "webSocketReadTimeout": true, "webSocketWriteText": true,
	"webSocketWriteBinary": true, "webSocketPing": true, "webSocketClose": true, "webSocketClosed": true,
}

// UsesDesktop reports whether the MIR references Z13 desktop APIs.
var assetNativeBuiltins = map[string]bool{"assetExists": true, "assetText": true, "assetBytes": true, "assetList": true}

// UsesAssets reports whether the MIR references Z15 embedded-asset APIs.
func UsesAssets(module *mir.Module) bool { return moduleUsesAnyBuiltin(module, assetNativeBuiltins) }

func UsesDesktop(module *mir.Module) bool { return moduleUsesAnyBuiltin(module, desktopNativeBuiltins) }
func UsesUI(module *mir.Module) bool      { return moduleUsesAnyBuiltin(module, uiNativeBuiltins) }

// UsesHTTP reports whether the MIR references Z11 HTTP, SSE, JWT or WebSocket APIs.
func UsesHTTP(module *mir.Module) bool { return moduleUsesAnyBuiltin(module, httpBuiltins) }

// UsesTLS reports whether the MIR references any TLS builtin and therefore
// requires OpenSSL during native compilation.
func UsesTLS(module *mir.Module) bool { return moduleUsesAnyBuiltin(module, tlsBuiltins) }

var sqliteBuiltins = map[string]bool{
	"sqliteOpen": true, "sqliteMemory": true, "sqliteExec": true, "sqliteQuery": true, "sqlitePrepare": true, "sqliteBegin": true, "sqliteClose": true, "sqliteIsOpen": true, "sqlitePath": true,
	"sqliteStatementExec": true, "sqliteStatementQuery": true, "sqliteStatementClose": true, "sqliteStatementOpen": true, "sqliteStatementSQL": true,
	"sqliteTransactionExec": true, "sqliteTransactionQuery": true, "sqliteTransactionPrepare": true, "sqliteCommit": true, "sqliteRollback": true, "sqliteTransactionActive": true,
	"sqliteQueryOne": true, "sqliteQueryStream": true, "sqliteMigrate": true, "sqliteSchemaVersion": true,
	"sqliteStatementQueryStream": true, "sqliteStatementParameterCount": true, "sqliteStatementColumns": true,
	"sqliteTransactionQueryStream": true, "sqliteSavepoint": true, "sqliteRollbackTo": true, "sqliteRelease": true,
	"sessionSQLite": true,
}

var postgresBuiltins = map[string]bool{}
var redisBuiltins = map[string]bool{}

func init() {
	for name := range z12NativeBuiltins {
		if strings.HasPrefix(name, "postgres") {
			postgresBuiltins[name] = true
		}
		if strings.HasPrefix(name, "redis") {
			redisBuiltins[name] = true
		}
	}
}

// UsesSQLite reports whether the MIR references SQLite or persistent SQLite sessions.
func UsesSQLite(module *mir.Module) bool   { return moduleUsesAnyBuiltin(module, sqliteBuiltins) }
func UsesPostgres(module *mir.Module) bool { return moduleUsesAnyBuiltin(module, postgresBuiltins) }
func UsesRedis(module *mir.Module) bool    { return moduleUsesAnyBuiltin(module, redisBuiltins) }
func UsesZ12(module *mir.Module) bool {
	return UsesSQLite(module) || moduleUsesAnyBuiltin(module, z12NativeBuiltins)
}

func moduleUsesAnyBuiltin(module *mir.Module, names map[string]bool) bool {
	if module == nil {
		return false
	}
	for _, declaration := range module.Declarations {
		if instructionUsesBuiltin(declaration, names) {
			return true
		}
	}
	if regionUsesBuiltin(module.Entry, names) {
		return true
	}
	for _, function := range module.Functions {
		if function != nil && regionUsesBuiltin(function.Body, names) {
			return true
		}
	}
	return false
}

func regionUsesBuiltin(region *mir.Region, names map[string]bool) bool {
	if region == nil {
		return false
	}
	for _, instruction := range region.Instructions {
		if instructionUsesBuiltin(instruction, names) {
			return true
		}
	}
	return false
}

func instructionUsesBuiltin(instruction *mir.Instruction, names map[string]bool) bool {
	if instruction == nil {
		return false
	}
	if instruction.Op == mir.OpLoad && names[instruction.Name] {
		return true
	}
	for _, region := range instruction.Regions {
		if regionUsesBuiltin(region, names) {
			return true
		}
	}
	return false
}
