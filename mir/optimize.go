package mir

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"zumbra/types"
)

type constant struct {
	kind types.Kind
	text string
}

// Optimize performs deterministic, semantics-preserving starter passes:
// constant folding, constant-condition annotation and unreachable-code removal.
func Optimize(module *Module) error {
	if module == nil {
		return fmt.Errorf("cannot optimize nil MIR module")
	}
	for _, fn := range module.Functions {
		optimizeRegion(fn.Body, nil, nil)
		eliminateDeadValues(fn.Body)
	}
	optimizeRegion(module.Entry, nil, nil)
	eliminateDeadValues(module.Entry)
	module.Optimized = true
	return Verify(module)
}

func optimizeRegion(region *Region, inherited map[ValueID]constant, inheritedLoads map[ValueID]string) {
	if region == nil {
		return
	}
	constants := map[ValueID]constant{}
	for key, value := range inherited {
		constants[key] = value
	}
	loads := map[ValueID]string{}
	for key, value := range inheritedLoads {
		loads[key] = value
	}
	optimized := make([]*Instruction, 0, len(region.Instructions))
	terminated := false
	terminalResult := ValueID(0)
	for _, inst := range region.Instructions {
		if terminated {
			continue
		}
		for _, nested := range inst.Regions {
			optimizeRegion(nested, constants, loads)
		}
		if folded, ok := foldInstruction(inst, constants, loads); ok {
			inst.Op = OpConst
			inst.Operator = ""
			inst.Args = nil
			inst.Literal = folded.text
			inst.Type = types.Simple(folded.kind)
			constants[inst.Result] = folded
		} else if inst.Op == OpConst && inst.Result != 0 && inst.Type != nil {
			constants[inst.Result] = constant{kind: inst.Type.Kind, text: inst.Literal}
		}
		if inst.Op == OpLoad && inst.Result != 0 {
			loads[inst.Result] = inst.Name
		}
		if inst.Op == OpIf && len(inst.Args) == 1 {
			if value, ok := constants[inst.Args[0]]; ok && value.kind == types.Bool {
				inst.Meta["constant_condition"] = value.text
			}
		}
		optimized = append(optimized, inst)
		switch inst.Op {
		case OpReturn:
			terminated = true
			if len(inst.Args) == 1 {
				terminalResult = inst.Args[0]
			}
		case OpBreak, OpContinue:
			terminated = true
		}
	}
	region.Instructions = optimized
	if terminalResult != 0 {
		region.Result = terminalResult
	}
}

func foldInstruction(inst *Instruction, constants map[ValueID]constant, loads map[ValueID]string) (constant, bool) {
	if inst == nil || inst.Result == 0 {
		return constant{}, false
	}
	switch inst.Op {
	case OpUnary:
		if len(inst.Args) != 1 {
			return constant{}, false
		}
		value, ok := constants[inst.Args[0]]
		if !ok {
			return constant{}, false
		}
		return foldUnary(inst.Operator, value)
	case OpBinary:
		if len(inst.Args) != 2 {
			return constant{}, false
		}
		left, lok := constants[inst.Args[0]]
		right, rok := constants[inst.Args[1]]
		if !lok || !rok {
			return constant{}, false
		}
		return foldBinary(inst.Operator, left, right, inst.Type)
	case OpCall:
		return foldSystemLayoutCall(inst, constants, loads)
	}
	return constant{}, false
}

func foldSystemLayoutCall(inst *Instruction, constants map[ValueID]constant, loads map[ValueID]string) (constant, bool) {
	if inst == nil || len(inst.Args) != 2 {
		return constant{}, false
	}
	callee, ok := loads[inst.Args[0]]
	if !ok || (callee != "sizeOfType" && callee != "alignOfType") {
		return constant{}, false
	}
	typeName, ok := constants[inst.Args[1]]
	if !ok || typeName.kind != types.String {
		return constant{}, false
	}
	size, alignment, ok := nativeSystemLayout(typeName.text)
	if !ok {
		return constant{}, false
	}
	value := size
	if callee == "alignOfType" {
		value = alignment
	}
	return constant{kind: types.Int, text: strconv.Itoa(value)}, true
}

func nativeSystemLayout(name string) (size int, alignment int, ok bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "u8", "i8", "bool":
		return 1, 1, true
	case "u16", "i16":
		return 2, 2, true
	case "u32", "i32":
		return 4, 4, true
	case "u64", "i64", "int", "float":
		return 8, 8, true
	}
	return 0, 0, false
}

func foldUnary(operator string, value constant) (constant, bool) {
	switch operator {
	case "!":
		if value.kind != types.Bool {
			return constant{}, false
		}
		parsed, err := strconv.ParseBool(value.text)
		if err != nil {
			return constant{}, false
		}
		return constant{kind: types.Bool, text: strconv.FormatBool(!parsed)}, true
	case "-", "bnot":
		parsed, ok := parseInteger(value.text)
		if !ok {
			return constant{}, false
		}
		if operator == "-" {
			parsed = -parsed
		} else {
			parsed = ^parsed
		}
		return constant{kind: value.kind, text: formatInteger(parsed, value.kind)}, true
	}
	return constant{}, false
}

func foldBinary(operator string, left, right constant, resultType *types.Type) (constant, bool) {
	if left.kind == types.String && right.kind == types.String && operator == "+" {
		return constant{kind: types.String, text: left.text + right.text}, true
	}
	if left.kind == types.Bool && right.kind == types.Bool {
		lv, le := strconv.ParseBool(left.text)
		rv, re := strconv.ParseBool(right.text)
		if le != nil || re != nil {
			return constant{}, false
		}
		switch operator {
		case "and":
			return constant{kind: types.Bool, text: strconv.FormatBool(lv && rv)}, true
		case "or":
			return constant{kind: types.Bool, text: strconv.FormatBool(lv || rv)}, true
		case "==":
			return constant{kind: types.Bool, text: strconv.FormatBool(lv == rv)}, true
		case "!=":
			return constant{kind: types.Bool, text: strconv.FormatBool(lv != rv)}, true
		}
	}
	if left.kind == types.Float || right.kind == types.Float {
		lv, le := strconv.ParseFloat(left.text, 64)
		rv, re := strconv.ParseFloat(right.text, 64)
		if le != nil || re != nil {
			return constant{}, false
		}
		kind := types.Float
		switch operator {
		case "+":
			return constant{kind: kind, text: formatFloat(lv + rv)}, true
		case "-":
			return constant{kind: kind, text: formatFloat(lv - rv)}, true
		case "*":
			return constant{kind: kind, text: formatFloat(lv * rv)}, true
		case "/":
			if rv == 0 {
				return constant{}, false
			}
			return constant{kind: kind, text: formatFloat(lv / rv)}, true
		case "**":
			return constant{kind: kind, text: formatFloat(math.Pow(lv, rv))}, true
		case "==", "!=", "<", ">", "<=", ">=":
			return constant{kind: types.Bool, text: strconv.FormatBool(compareFloat(operator, lv, rv))}, true
		}
	}
	lv, lok := parseInteger(left.text)
	rv, rok := parseInteger(right.text)
	if !lok || !rok {
		return constant{}, false
	}
	kind := left.kind
	if resultType != nil && resultType.Kind != types.Unknown {
		kind = resultType.Kind
	}
	if operator == "==" || operator == "!=" || operator == "<" || operator == ">" || operator == "<=" || operator == ">=" {
		return constant{kind: types.Bool, text: strconv.FormatBool(compareInt(operator, lv, rv))}, true
	}
	var value int64
	switch operator {
	case "+":
		value = lv + rv
	case "-":
		value = lv - rv
	case "*":
		value = lv * rv
	case "/":
		if rv == 0 {
			return constant{}, false
		}
		value = lv / rv
	case "%":
		if rv == 0 {
			return constant{}, false
		}
		value = lv % rv
	case "**":
		if rv < 0 {
			return constant{}, false
		}
		value = int64(math.Pow(float64(lv), float64(rv)))
	case "band":
		value = lv & rv
	case "bor":
		value = lv | rv
	case "bxor":
		value = lv ^ rv
	case "shl":
		if rv < 0 || rv > 63 {
			return constant{}, false
		}
		value = lv << uint(rv)
	case "shr":
		if rv < 0 || rv > 63 {
			return constant{}, false
		}
		value = lv >> uint(rv)
	default:
		return constant{}, false
	}
	return constant{kind: kind, text: formatInteger(value, kind)}, true
}

func parseInteger(text string) (int64, bool) {
	clean := text
	for _, suffix := range []string{"u16", "u32", "u64", "i16", "i32", "i64", "u8", "i8"} {
		if strings.HasSuffix(clean, suffix) {
			clean = strings.TrimSuffix(clean, suffix)
			break
		}
	}
	value, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		if unsigned, uerr := strconv.ParseUint(clean, 10, 64); uerr == nil {
			return int64(unsigned), true
		}
		return 0, false
	}
	return value, true
}

func formatInteger(value int64, kind types.Kind) string {
	if bits, signed, ok := fixedInfo(kind); ok {
		mask := uint64(math.MaxUint64)
		if bits < 64 {
			mask = (uint64(1) << bits) - 1
		}
		raw := uint64(value) & mask
		if signed && bits < 64 && raw&(uint64(1)<<(bits-1)) != 0 {
			value = int64(raw | ^mask)
		} else if signed {
			value = int64(raw)
		} else {
			return strconv.FormatUint(raw, 10) + string(kind)
		}
		return strconv.FormatInt(value, 10) + string(kind)
	}
	return strconv.FormatInt(value, 10)
}
func fixedInfo(kind types.Kind) (uint, bool, bool) {
	switch kind {
	case types.U8:
		return 8, false, true
	case types.U16:
		return 16, false, true
	case types.U32:
		return 32, false, true
	case types.U64:
		return 64, false, true
	case types.I8:
		return 8, true, true
	case types.I16:
		return 16, true, true
	case types.I32:
		return 32, true, true
	case types.I64:
		return 64, true, true
	}
	return 0, false, false
}
func formatFloat(value float64) string { return strconv.FormatFloat(value, 'g', -1, 64) }
func compareInt(op string, l, r int64) bool {
	switch op {
	case "==":
		return l == r
	case "!=":
		return l != r
	case "<":
		return l < r
	case ">":
		return l > r
	case "<=":
		return l <= r
	case ">=":
		return l >= r
	}
	return false
}
func compareFloat(op string, l, r float64) bool {
	switch op {
	case "==":
		return l == r
	case "!=":
		return l != r
	case "<":
		return l < r
	case ">":
		return l > r
	case "<=":
		return l <= r
	case ">=":
		return l >= r
	}
	return false
}

func eliminateDeadValues(region *Region) {
	if region == nil {
		return
	}
	for _, inst := range region.Instructions {
		for _, nested := range inst.Regions {
			eliminateDeadValues(nested)
		}
	}

	producers := map[ValueID]*Instruction{}
	for _, inst := range region.Instructions {
		if inst.Result != 0 {
			producers[inst.Result] = inst
		}
	}
	used := map[ValueID]bool{}
	if region.Result != 0 {
		used[region.Result] = true
	}
	keptReverse := make([]*Instruction, 0, len(region.Instructions))
	for i := len(region.Instructions) - 1; i >= 0; i-- {
		inst := region.Instructions[i]
		keep := hasSideEffects(inst.Op) || (inst.Result != 0 && used[inst.Result])
		if inst.Op == OpDrop {
			keep = false
		}
		if !keep {
			continue
		}
		for _, arg := range inst.Args {
			if arg != 0 {
				used[arg] = true
			}
		}
		for _, nested := range inst.Regions {
			for value := range externalRegionUses(nested) {
				used[value] = true
			}
		}
		keptReverse = append(keptReverse, inst)
	}
	kept := make([]*Instruction, len(keptReverse))
	for i := range keptReverse {
		kept[len(keptReverse)-1-i] = keptReverse[i]
	}
	region.Instructions = kept

	// A region result may have belonged only to a discarded expression
	// statement. Clear it rather than leaving an invalid reference.
	if region.Result != 0 {
		if _, exists := producers[region.Result]; exists {
			found := false
			for _, inst := range kept {
				if inst.Result == region.Result {
					found = true
					break
				}
			}
			if !found {
				region.Result = 0
			}
		}
	}
}

func hasSideEffects(op Op) bool {
	switch op {
	case OpDeclare, OpStore, OpSetIndex, OpSetField, OpCall,
		OpIf, OpMatch, OpCase, OpWhile, OpFor, OpForEach, OpForRange, OpForever,
		OpReturn, OpBreak, OpContinue, OpSpawn, OpAwait, OpTry, OpHandler,
		OpImport, OpExtern, OpUnsafe, OpTypeAlias, OpStruct, OpStructField, OpEnum, OpUnknown:
		return true
	default:
		return false
	}
}

func externalRegionUses(region *Region) map[ValueID]bool {
	result := map[ValueID]bool{}
	if region == nil {
		return result
	}
	local := map[ValueID]bool{}
	for _, inst := range region.Instructions {
		if inst.Result != 0 {
			local[inst.Result] = true
		}
	}
	if region.Result != 0 && !local[region.Result] {
		result[region.Result] = true
	}
	for _, inst := range region.Instructions {
		for _, arg := range inst.Args {
			if arg != 0 && !local[arg] {
				result[arg] = true
			}
		}
		for _, nested := range inst.Regions {
			for value := range externalRegionUses(nested) {
				if !local[value] {
					result[value] = true
				}
			}
		}
	}
	return result
}
