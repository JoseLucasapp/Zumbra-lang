package nativec

import (
	"fmt"
	"strings"
)

func (g *generator) emitFFIDeclarations() {
	if len(g.externs) == 0 {
		return
	}
	g.line("/* Z8 C FFI declarations and adapters. */")
	for _, external := range g.externs {
		parameters := make([]string, 0, len(external.Params))
		for _, parameter := range external.Params {
			parameters = append(parameters, parameter.Type.cDecl(sanitize(parameter.Name)))
		}
		if len(parameters) == 0 {
			parameters = append(parameters, "void")
		}
		g.line("extern %s %s(%s);", external.Return.cBase(), external.CName, strings.Join(parameters, ", "))
	}
	g.line("")
	for _, external := range g.externs {
		g.emitFFICallbacks(external)
		g.emitFFIWrapper(external)
	}
}

func (g *generator) emitFFICallbacks(external ffiInfo) {
	for index, parameter := range external.Params {
		if parameter.Type.Name != "callback" {
			continue
		}
		global := fmt.Sprintf("zffi_callback_%d_%d", external.ID, index)
		trampoline := fmt.Sprintf("zffi_trampoline_%d_%d", external.ID, index)
		g.line("static ZValue %s;", global)
		declarations := make([]string, 0, len(parameter.Type.CallbackParams))
		for paramIndex, callbackParam := range parameter.Type.CallbackParams {
			declarations = append(declarations, callbackParam.cDecl(fmt.Sprintf("arg%d", paramIndex)))
		}
		if len(declarations) == 0 {
			declarations = append(declarations, "void")
		}
		returned := "void"
		if parameter.Type.CallbackReturn != nil {
			returned = parameter.Type.CallbackReturn.cBase()
		}
		g.line("static %s %s(%s) {", returned, trampoline, strings.Join(declarations, ", "))
		g.indent++
		if len(parameter.Type.CallbackParams) > 0 {
			values := make([]string, 0, len(parameter.Type.CallbackParams))
			for paramIndex, callbackParam := range parameter.Type.CallbackParams {
				values = append(values, ffiToZ(callbackParam, fmt.Sprintf("arg%d", paramIndex)))
			}
			g.line("ZValue callback_args[] = {%s};", strings.Join(values, ", "))
			g.line("ZValue callback_result = z_call(%s, callback_args, %d);", global, len(values))
		} else {
			g.line("ZValue callback_result = z_call(%s, NULL, 0);", global)
		}
		if parameter.Type.CallbackReturn == nil || parameter.Type.CallbackReturn.Name == "void" {
			g.line("(void)callback_result;")
			g.line("return;")
		} else {
			g.line("return %s;", ffiFromZ(*parameter.Type.CallbackReturn, "callback_result"))
		}
		g.indent--
		g.line("}")
		g.line("")
	}
}

func (g *generator) emitFFIWrapper(external ffiInfo) {
	g.line("static ZValue zffi_%d(const ZValue *args, size_t argc) {", external.ID)
	g.indent++
	g.line("if (argc != %d) z_fatal(%s, argc);", len(external.Params), cString(fmt.Sprintf("extern %s expects %d arguments, got %%zu", external.Name, len(external.Params))))
	arguments := make([]string, 0, len(external.Params))
	for index, parameter := range external.Params {
		if parameter.Type.Name == "callback" {
			global := fmt.Sprintf("zffi_callback_%d_%d", external.ID, index)
			trampoline := fmt.Sprintf("zffi_trampoline_%d_%d", external.ID, index)
			g.line("if (args[%d].tag != ZV_FUNCTION && args[%d].tag != ZV_BOUND_METHOD) z_fatal(\"extern callback argument %d must be a Zumbra function\");", index, index, index+1)
			g.line("%s = args[%d];", global, index)
			arguments = append(arguments, trampoline)
		} else {
			arguments = append(arguments, ffiFromZ(parameter.Type, fmt.Sprintf("args[%d]", index)))
		}
	}
	call := fmt.Sprintf("%s(%s)", external.CName, strings.Join(arguments, ", "))
	if external.Return.Name == "void" {
		g.line("%s;", call)
		g.line("return z_null();")
	} else {
		g.line("%s ffi_result = %s;", external.Return.cBase(), call)
		g.line("return %s;", ffiToZ(external.Return, "ffi_result"))
	}
	g.indent--
	g.line("}")
	g.line("")
}
