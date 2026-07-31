package transpiler

import (
	"fmt"
	"zumbra/mir"
	"zumbra/pipeline"
)

// ZumbraTranspilerPipeline ensures the Go backend receives an already parsed,
// semantically checked and typed source unit. Z7 will replace the textual
// backend internals with direct MIR consumption.
func ZumbraTranspilerPipeline(result *pipeline.Result) (string, error) {
	if result == nil || result.Program == nil || result.MIR == nil {
		return "", fmt.Errorf("transpiler received an incomplete pipeline result")
	}
	for _, declaration := range result.MIR.Declarations {
		if declaration.Op == mir.OpExtern {
			return "", fmt.Errorf("extern C is available only in the native backend")
		}
	}
	for _, function := range result.MIR.Functions {
		if function != nil && function.Async {
			return "", fmt.Errorf("Z9 concurrency is supported by the VM and native backend; the legacy Go transpiler does not schedule async functions")
		}
		if function != nil && mirRegionHasConcurrency(function.Body) {
			return "", fmt.Errorf("Z9 concurrency is supported by the VM and native backend; the legacy Go transpiler does not support spawn/await")
		}
	}
	if mirRegionHasConcurrency(result.MIR.Entry) {
		return "", fmt.Errorf("Z9 concurrency is supported by the VM and native backend; the legacy Go transpiler does not support spawn/await")
	}
	for _, function := range result.MIR.Functions {
		if function != nil && mirRegionHasNetwork(function.Body) {
			return "", fmt.Errorf("Z10 networking is supported by the VM and native backend; the legacy Go transpiler does not provide socket streams")
		}
	}
	if mirRegionHasNetwork(result.MIR.Entry) {
		return "", fmt.Errorf("Z10 networking is supported by the VM and native backend; the legacy Go transpiler does not provide socket streams")
	}
	for _, function := range result.MIR.Functions {
		if function != nil && mirRegionHasHTTP(function.Body) {
			return "", fmt.Errorf("Z11 HTTP and WebSockets are supported by the VM and native backend; the legacy Go transpiler does not provide the HTTP runtime")
		}
	}
	if mirRegionHasHTTP(result.MIR.Entry) {
		return "", fmt.Errorf("Z11 HTTP and WebSockets are supported by the VM and native backend; the legacy Go transpiler does not provide the HTTP runtime")
	}
	return ZumbraTranspiler(result.Program.String())
}

var networkBuiltinNames = map[string]bool{
	"tcpListen": true, "tcpConnect": true, "tcpConnectTimeout": true,
	"tlsListen": true, "tlsConnect": true, "tlsConnectTimeout": true,
	"listenerAccept": true, "listenerAcceptTimeout": true, "listenerClose": true, "listenerClosed": true,
	"listenerAddress": true, "listenerPort": true,
	"streamRead": true, "streamReadExact": true, "streamReadTimeout": true, "streamWrite": true, "streamWriteAll": true,
	"streamClose": true, "streamClosed": true, "streamShutdownRead": true, "streamShutdownWrite": true,
	"streamLocalAddress": true, "streamLocalPort": true, "streamRemoteAddress": true, "streamRemotePort": true,
	"streamSetReadTimeout": true, "streamSetWriteTimeout": true, "tcpSetKeepAlive": true,
	"dnsLookup": true, "dnsLookupTimeout": true,
	"udpBind": true, "udpSendTo": true, "udpReceiveFrom": true, "udpReceiveFromTimeout": true,
	"udpClose": true, "udpClosed": true, "udpAddress": true, "udpPort": true,
}

func mirRegionHasNetwork(region *mir.Region) bool {
	if region == nil {
		return false
	}
	for _, instruction := range region.Instructions {
		if instruction == nil {
			continue
		}
		if instruction.Op == mir.OpLoad && networkBuiltinNames[instruction.Name] {
			return true
		}
		for _, child := range instruction.Regions {
			if mirRegionHasNetwork(child) {
				return true
			}
		}
	}
	return false
}

func mirRegionHasConcurrency(region *mir.Region) bool {
	if region == nil {
		return false
	}
	for _, instruction := range region.Instructions {
		if instruction == nil {
			continue
		}
		if instruction.Op == mir.OpSpawn || instruction.Op == mir.OpAwait {
			return true
		}
		for _, child := range instruction.Regions {
			if mirRegionHasConcurrency(child) {
				return true
			}
		}
	}
	return false
}

var httpBuiltinNames = map[string]bool{
	"httpApp": true, "httpRoute": true, "httpUse": true, "httpStatic": true, "httpLimitBody": true,
	"httpCompression": true, "httpCors": true, "httpServe": true, "httpServeTLS": true, "httpShutdown": true,
	"httpServerPort": true, "httpServerAddress": true, "httpServerRunning": true,
	"httpText": true, "httpJson": true, "httpHtml": true, "httpRedirect": true, "httpFile": true,
	"httpHeader": true, "httpCookie": true, "httpStream": true, "httpSSE": true, "sseEvent": true,
	"httpRequest": true, "httpStatus": true, "httpBody": true, "httpBodyBytes": true, "httpBodyJSON": true,
	"httpHeaders": true, "jsonStringify": true, "jsonParse": true, "jwtSignHS256": true, "jwtVerifyHS256": true,
	"webSocketUpgrade": true, "webSocketConnect": true, "webSocketRead": true, "webSocketReadTimeout": true,
	"webSocketWriteText": true, "webSocketWriteBinary": true, "webSocketPing": true,
	"webSocketClose": true, "webSocketClosed": true,
}

func mirRegionHasHTTP(region *mir.Region) bool {
	if region == nil {
		return false
	}
	for _, instruction := range region.Instructions {
		if instruction == nil {
			continue
		}
		if instruction.Op == mir.OpLoad && httpBuiltinNames[instruction.Name] {
			return true
		}
		for _, child := range instruction.Regions {
			if mirRegionHasHTTP(child) {
				return true
			}
		}
	}
	return false
}
