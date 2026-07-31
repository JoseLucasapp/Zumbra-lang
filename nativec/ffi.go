package nativec

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"zumbra/mir"
)

type ffiType struct {
	Name           string
	CallbackParams []ffiType
	CallbackReturn *ffiType
}

type ffiParam struct {
	Name string
	Type ffiType
}

type ffiInfo struct {
	ID          int
	Name        string
	CName       string
	ABI         string
	Link        string
	Params      []ffiParam
	Return      ffiType
	Instruction int
}

func externFromInstruction(id int, instruction *mir.Instruction) (ffiInfo, error) {
	info := ffiInfo{ID: id, Name: instruction.Name, CName: instruction.Meta["c_name"], ABI: instruction.Meta["abi"], Link: instruction.Meta["link"], Instruction: instruction.ID}
	if info.CName == "" {
		info.CName = info.Name
	}
	if !cIdentifier(info.CName) {
		return info, fmt.Errorf("extern C symbol %q is not a valid C identifier", info.CName)
	}
	count, err := strconv.Atoi(instruction.Meta["param_count"])
	if err != nil && instruction.Meta["param_count"] != "" {
		return info, fmt.Errorf("extern %s has invalid parameter metadata", info.Name)
	}
	for index := 0; index < count; index++ {
		raw := instruction.Meta[fmt.Sprintf("param.%d.type", index)]
		parsed, parseErr := parseFFIType(raw)
		if parseErr != nil {
			return info, fmt.Errorf("extern %s parameter %d: %w", info.Name, index+1, parseErr)
		}
		name := instruction.Meta[fmt.Sprintf("param.%d.name", index)]
		if name == "" {
			name = fmt.Sprintf("arg%d", index)
		}
		info.Params = append(info.Params, ffiParam{Name: name, Type: parsed})
	}
	info.Return, err = parseFFIType(instruction.Meta["return"])
	if err != nil {
		return info, fmt.Errorf("extern %s return: %w", info.Name, err)
	}
	return info, nil
}

func parseFFIType(raw string) (ffiType, error) {
	parser := &ffiTypeParser{input: strings.TrimSpace(raw)}
	result, err := parser.parse()
	if err != nil {
		return ffiType{}, err
	}
	parser.space()
	if parser.index != len(parser.input) {
		return ffiType{}, fmt.Errorf("unexpected FFI type suffix %q", parser.input[parser.index:])
	}
	if err := validateFFIType(result, false); err != nil {
		return ffiType{}, err
	}
	return result, nil
}

type ffiTypeParser struct {
	input string
	index int
}

func (p *ffiTypeParser) space() {
	for p.index < len(p.input) && unicode.IsSpace(rune(p.input[p.index])) {
		p.index++
	}
}
func (p *ffiTypeParser) parseName() string {
	p.space()
	start := p.index
	for p.index < len(p.input) {
		r := rune(p.input[p.index])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			break
		}
		p.index++
	}
	return p.input[start:p.index]
}
func (p *ffiTypeParser) consume(text string) bool {
	p.space()
	if strings.HasPrefix(p.input[p.index:], text) {
		p.index += len(text)
		return true
	}
	return false
}
func (p *ffiTypeParser) parse() (ffiType, error) {
	name := p.parseName()
	if name == "" {
		return ffiType{}, fmt.Errorf("missing FFI type")
	}
	result := ffiType{Name: name}
	if name != "callback" {
		return result, nil
	}
	if !p.consume("(") {
		return ffiType{}, fmt.Errorf("callback requires parameter list")
	}
	if !p.consume(")") {
		for {
			parameter, err := p.parse()
			if err != nil {
				return ffiType{}, err
			}
			result.CallbackParams = append(result.CallbackParams, parameter)
			if p.consume(")") {
				break
			}
			if !p.consume(",") {
				return ffiType{}, fmt.Errorf("callback parameters must be comma-separated")
			}
		}
	}
	if !p.consume("->") {
		return ffiType{}, fmt.Errorf("callback requires return type")
	}
	returned, err := p.parse()
	if err != nil {
		return ffiType{}, err
	}
	result.CallbackReturn = &returned
	return result, nil
}

func validateFFIType(value ffiType, allowVoid bool) error {
	switch value.Name {
	case "void":
		if !allowVoid {
			return fmt.Errorf("void is only valid as a return type")
		}
	case "int", "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "usize", "float", "bool", "string", "cstring", "ptr":
	case "callback":
		if value.CallbackReturn == nil {
			return fmt.Errorf("callback return type is missing")
		}
		for _, parameter := range value.CallbackParams {
			if err := validateFFIType(parameter, false); err != nil {
				return err
			}
		}
		if err := validateFFIType(*value.CallbackReturn, true); err != nil {
			return err
		}
		if value.CallbackReturn.Name == "callback" {
			return fmt.Errorf("callbacks cannot return callbacks")
		}
	default:
		return fmt.Errorf("unsupported C FFI type %q", value.Name)
	}
	return nil
}

func cIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if r == '_' || unicode.IsLetter(r) || (index > 0 && unicode.IsDigit(r)) {
			continue
		}
		return false
	}
	return true
}

func (t ffiType) cBase() string {
	switch t.Name {
	case "void":
		return "void"
	case "int":
		return "int"
	case "i64":
		return "int64_t"
	case "i8":
		return "int8_t"
	case "i16":
		return "int16_t"
	case "i32":
		return "int32_t"
	case "u8":
		return "uint8_t"
	case "u16":
		return "uint16_t"
	case "u32":
		return "uint32_t"
	case "u64":
		return "uint64_t"
	case "usize":
		return "size_t"
	case "float":
		return "double"
	case "bool":
		return "bool"
	case "string", "cstring":
		return "const char *"
	case "ptr":
		return "void *"
	default:
		return "void"
	}
}

func (t ffiType) cDecl(name string) string {
	if t.Name != "callback" {
		return t.cBase() + " " + name
	}
	returned := "void"
	if t.CallbackReturn != nil {
		returned = t.CallbackReturn.cBase()
	}
	parameters := make([]string, 0, len(t.CallbackParams))
	for index, param := range t.CallbackParams {
		parameters = append(parameters, param.cDecl(fmt.Sprintf("arg%d", index)))
	}
	if len(parameters) == 0 {
		parameters = append(parameters, "void")
	}
	return fmt.Sprintf("%s (*%s)(%s)", returned, name, strings.Join(parameters, ", "))
}

func ffiFromZ(value ffiType, expression string) string {
	switch value.Name {
	case "int", "i8", "i16", "i32", "i64":
		return fmt.Sprintf("(%s)z_as_i64(%s)", value.cBase(), expression)
	case "u8", "u16", "u32", "u64", "usize":
		return fmt.Sprintf("(%s)z_as_u64(%s)", value.cBase(), expression)
	case "float":
		return "z_as_f64(" + expression + ")"
	case "bool":
		return "z_as_bool(" + expression + ")"
	case "string", "cstring":
		return "z_as_cstring(" + expression + ")"
	case "ptr":
		return "z_as_pointer(" + expression + ")"
	default:
		return expression
	}
}

func ffiToZ(value ffiType, expression string) string {
	switch value.Name {
	case "void":
		return "z_null()"
	case "int":
		return "z_int((int64_t)(" + expression + "))"
	case "i8":
		return "z_signed((int64_t)(" + expression + "), ZK_I8)"
	case "i16":
		return "z_signed((int64_t)(" + expression + "), ZK_I16)"
	case "i32":
		return "z_signed((int64_t)(" + expression + "), ZK_I32)"
	case "i64":
		return "z_signed((int64_t)(" + expression + "), ZK_I64)"
	case "u8":
		return "z_uint((uint64_t)(" + expression + "), ZK_U8)"
	case "u16":
		return "z_uint((uint64_t)(" + expression + "), ZK_U16)"
	case "u32":
		return "z_uint((uint64_t)(" + expression + "), ZK_U32)"
	case "u64", "usize":
		return "z_uint((uint64_t)(" + expression + "), ZK_U64)"
	case "float":
		return "z_float((double)(" + expression + "))"
	case "bool":
		return "z_bool((bool)(" + expression + "))"
	case "string", "cstring":
		return "z_string((" + expression + ") == NULL ? \"\" : (" + expression + "))"
	case "ptr":
		return "z_pointer((void *)(" + expression + "))"
	default:
		return "z_null()"
	}
}
