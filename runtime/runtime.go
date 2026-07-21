package runtime

func Runtime() string {
	return `

type zAnyInteger interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func zIntegerBig[T zAnyInteger](value T) *big.Int {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return big.NewInt(reflected.Int())
	default:
		return new(big.Int).SetUint64(reflected.Uint())
	}
}

func zIntegerBigDynamic(value interface{}) *big.Int {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return big.NewInt(reflected.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return new(big.Int).SetUint64(reflected.Uint())
	default:
		panic(fmt.Sprintf("expected integer, got %T", value))
	}
}

func zIntegerBounds(signed bool, bits uint) (*big.Int, *big.Int) {
	if signed {
		max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), bits-1), big.NewInt(1))
		min := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), bits-1))
		return min, max
	}
	return big.NewInt(0), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), bits), big.NewInt(1))
}

func zRequireIntegerRange[T zAnyInteger](value T, name string, signed bool, bits uint) *big.Int {
	number := zIntegerBig(value)
	min, max := zIntegerBounds(signed, bits)
	if number.Cmp(min) < 0 || number.Cmp(max) > 0 {
		panic(fmt.Sprintf("value %s is outside %s range", number.String(), name))
	}
	return number
}

func zU8[T zAnyInteger](value T) uint8 { return uint8(zRequireIntegerRange(value, "u8", false, 8).Uint64()) }
func zU16[T zAnyInteger](value T) uint16 { return uint16(zRequireIntegerRange(value, "u16", false, 16).Uint64()) }
func zU32[T zAnyInteger](value T) uint32 { return uint32(zRequireIntegerRange(value, "u32", false, 32).Uint64()) }
func zU64[T zAnyInteger](value T) uint64 { return zRequireIntegerRange(value, "u64", false, 64).Uint64() }
func zI8[T zAnyInteger](value T) int8 { return int8(zRequireIntegerRange(value, "i8", true, 8).Int64()) }
func zI16[T zAnyInteger](value T) int16 { return int16(zRequireIntegerRange(value, "i16", true, 16).Int64()) }
func zI32[T zAnyInteger](value T) int32 { return int32(zRequireIntegerRange(value, "i32", true, 32).Int64()) }
func zI64[T zAnyInteger](value T) int64 { return zRequireIntegerRange(value, "i64", true, 64).Int64() }

func wrapAdd[T zAnyInteger](left, right T) T { return left + right }
func wrapSub[T zAnyInteger](left, right T) T { return left - right }
func wrapMul[T zAnyInteger](left, right T) T { return left * right }

func zFixedTypeInfo[T zAnyInteger]() (bool, uint) {
	var zero T
	typeOf := reflect.TypeOf(zero)
	switch typeOf.Kind() {
	case reflect.Int8:
		return true, 8
	case reflect.Int16:
		return true, 16
	case reflect.Int32:
		return true, 32
	case reflect.Int64, reflect.Int:
		return true, uint(typeOf.Bits())
	case reflect.Uint8:
		return false, 8
	case reflect.Uint16:
		return false, 16
	case reflect.Uint32:
		return false, 32
	case reflect.Uint64, reflect.Uint:
		return false, uint(typeOf.Bits())
	default:
		panic("unsupported fixed integer type")
	}
}

func zFixedFromBig[T zAnyInteger](value *big.Int) T {
	var zero T
	if reflect.TypeOf(zero).Kind() >= reflect.Uint && reflect.TypeOf(zero).Kind() <= reflect.Uint64 {
		return T(value.Uint64())
	}
	return T(value.Int64())
}

func zFixedArithmetic[T zAnyInteger](left, right T, operator string, mode string) T {
	leftBig := zIntegerBig(left)
	rightBig := zIntegerBig(right)
	result := new(big.Int)
	switch operator {
	case "+":
		result.Add(leftBig, rightBig)
	case "-":
		result.Sub(leftBig, rightBig)
	case "*":
		result.Mul(leftBig, rightBig)
	}

	signed, bits := zFixedTypeInfo[T]()
	min, max := zIntegerBounds(signed, bits)
	switch mode {
	case "checked":
		if result.Cmp(min) < 0 || result.Cmp(max) > 0 {
			panic("fixed integer overflow")
		}
	case "saturating":
		if result.Cmp(min) < 0 { result.Set(min) }
		if result.Cmp(max) > 0 { result.Set(max) }
	case "wrapping":
		modulus := new(big.Int).Lsh(big.NewInt(1), bits)
		result.Mod(result, modulus)
		if result.Sign() < 0 { result.Add(result, modulus) }
	}
	return zFixedFromBig[T](result)
}

func checkedAdd[T zAnyInteger](left, right T) T { return zFixedArithmetic(left, right, "+", "checked") }
func checkedSub[T zAnyInteger](left, right T) T { return zFixedArithmetic(left, right, "-", "checked") }
func checkedMul[T zAnyInteger](left, right T) T { return zFixedArithmetic(left, right, "*", "checked") }
func satAdd[T zAnyInteger](left, right T) T { return zFixedArithmetic(left, right, "+", "saturating") }
func satSub[T zAnyInteger](left, right T) T { return zFixedArithmetic(left, right, "-", "saturating") }
func satMul[T zAnyInteger](left, right T) T { return zFixedArithmetic(left, right, "*", "saturating") }

func zIndex(value interface{}) int {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		index := reflected.Int()
		if index < 0 { panic(fmt.Sprintf("index must be non-negative, got %d", index)) }
		return int(index)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(reflected.Uint())
	default:
		panic(fmt.Sprintf("index must be integer, got %T", value))
	}
}

func zBytes(size interface{}) []uint8 {
	return make([]uint8, zIndex(size))
}

func zArrayOf(kind string, size interface{}) interface{} {
	length := zIndex(size)
	switch kind {
	case "u8": return make([]uint8, length)
	case "u16": return make([]uint16, length)
	case "u32": return make([]uint32, length)
	case "u64": return make([]uint64, length)
	case "i8": return make([]int8, length)
	case "i16": return make([]int16, length)
	case "i32": return make([]int32, length)
	case "i64": return make([]int64, length)
	default: panic(fmt.Sprintf("unsupported arrayOf type %q", kind))
	}
}

func zGet(container interface{}, index interface{}) interface{} {
	value := reflect.ValueOf(container)
	i := zIndex(index)
	if value.Kind() == reflect.Map {
		key := reflect.ValueOf(index)
		result := value.MapIndex(key)
		if !result.IsValid() { return nil }
		return result.Interface()
	}
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice && value.Kind() != reflect.String {
		panic(fmt.Sprintf("value of type %T cannot be indexed", container))
	}
	if i < 0 || i >= value.Len() { panic(fmt.Sprintf("index out of bounds: %d (length %d)", i, value.Len())) }
	return value.Index(i).Interface()
}

func zConvertElement(value interface{}, target reflect.Type) reflect.Value {
	input := reflect.ValueOf(value)
	if input.IsValid() && input.Type().AssignableTo(target) { return input }
	if input.IsValid() && input.Type().ConvertibleTo(target) {
		converted := input.Convert(target)
		if target.Kind() >= reflect.Int && target.Kind() <= reflect.Int64 {
			original := zIntegerBigDynamic(value)
			if original.Cmp(big.NewInt(converted.Int())) != 0 { panic("integer value is outside typed array range") }
		}
		if target.Kind() >= reflect.Uint && target.Kind() <= reflect.Uint64 {
			original := zIntegerBigDynamic(value)
			convertedBig := new(big.Int).SetUint64(converted.Uint())
			if original.Sign() < 0 || original.Cmp(convertedBig) != 0 { panic("integer value is outside typed array range") }
		}
		return converted
	}
	panic(fmt.Sprintf("cannot store %T in %s", value, target))
}

func zSet(container interface{}, index interface{}, newValue interface{}) interface{} {
	value := reflect.ValueOf(container)
	i := zIndex(index)
	if value.Kind() == reflect.Map {
		value.SetMapIndex(reflect.ValueOf(index), reflect.ValueOf(newValue))
		return newValue
	}
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		panic(fmt.Sprintf("value of type %T cannot be changed by index", container))
	}
	if i < 0 || i >= value.Len() { panic(fmt.Sprintf("index out of bounds: %d (length %d)", i, value.Len())) }
	value.Index(i).Set(zConvertElement(newValue, value.Type().Elem()))
	return newValue
}

func zSlice(container interface{}, start interface{}, end interface{}) interface{} {
	value := reflect.ValueOf(container)
	from := zIndex(start)
	to := zIndex(end)
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		panic(fmt.Sprintf("value of type %T cannot be sliced", container))
	}
	if from < 0 || to < from || to > value.Len() { panic(fmt.Sprintf("invalid slice range [%d:%d] for length %d", from, to, value.Len())) }
	return value.Slice(from, to).Interface()
}

func zFill(container interface{}, newValue interface{}) interface{} {
	value := reflect.ValueOf(container)
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		panic(fmt.Sprintf("value of type %T cannot be filled", container))
	}
	for i := 0; i < value.Len(); i++ { value.Index(i).Set(zConvertElement(newValue, value.Type().Elem())) }
	return container
}

func zByteReflect(buffer interface{}) reflect.Value {
	value := reflect.ValueOf(buffer)
	if value.Kind() != reflect.Array && value.Kind() != reflect.Slice {
		panic(fmt.Sprintf("expected byte-compatible buffer, got %T", buffer))
	}
	kind := value.Type().Elem().Kind()
	if kind != reflect.Uint8 && kind != reflect.Int8 {
		panic(fmt.Sprintf("expected byte-compatible buffer, got %T", buffer))
	}
	return value
}

func zByteData(buffer interface{}) []byte {
	value := zByteReflect(buffer)
	data := make([]byte, value.Len())
	for i := 0; i < value.Len(); i++ {
		if value.Type().Elem().Kind() == reflect.Uint8 {
			data[i] = byte(value.Index(i).Uint())
		} else {
			data[i] = byte(int8(value.Index(i).Int()))
		}
	}
	return data
}

func zSetByte(buffer interface{}, index int, value byte) {
	reflected := zByteReflect(buffer)
	if index < 0 || index >= reflected.Len() {
		panic(fmt.Sprintf("byte index out of bounds: %d (length %d)", index, reflected.Len()))
	}
	if reflected.Type().Elem().Kind() == reflect.Uint8 {
		reflected.Index(index).SetUint(uint64(value))
	} else {
		reflected.Index(index).SetInt(int64(int8(value)))
	}
}

func zReadBytes(path string) []uint8 {
	data, err := os.ReadFile(path)
	if err != nil { panic(fmt.Sprintf("readBytes %q: %v", path, err)) }
	return []uint8(data)
}

func zWriteBytes(path string, buffer interface{}) int {
	data := zByteData(buffer)
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(fmt.Sprintf("writeBytes %q: %v", path, err))
	}
	return len(data)
}

func zReadUnsigned(buffer interface{}, offset interface{}, width int, order binary.ByteOrder) uint64 {
	data := zByteData(buffer)
	start := zIndex(offset)
	if start > len(data) || width > len(data)-start {
		panic(fmt.Sprintf("byte range [%d:%d] is outside buffer length %d", start, start+width, len(data)))
	}
	window := data[start:start+width]
	switch width {
	case 2: return uint64(order.Uint16(window))
	case 4: return uint64(order.Uint32(window))
	case 8: return order.Uint64(window)
	default: panic(fmt.Sprintf("unsupported integer width %d", width))
	}
}

func zReadU16LE(buffer interface{}, offset interface{}) uint16 { return uint16(zReadUnsigned(buffer, offset, 2, binary.LittleEndian)) }
func zReadU16BE(buffer interface{}, offset interface{}) uint16 { return uint16(zReadUnsigned(buffer, offset, 2, binary.BigEndian)) }
func zReadU32LE(buffer interface{}, offset interface{}) uint32 { return uint32(zReadUnsigned(buffer, offset, 4, binary.LittleEndian)) }
func zReadU32BE(buffer interface{}, offset interface{}) uint32 { return uint32(zReadUnsigned(buffer, offset, 4, binary.BigEndian)) }
func zReadU64LE(buffer interface{}, offset interface{}) uint64 { return zReadUnsigned(buffer, offset, 8, binary.LittleEndian) }
func zReadU64BE(buffer interface{}, offset interface{}) uint64 { return zReadUnsigned(buffer, offset, 8, binary.BigEndian) }

func zWriteUnsigned(buffer interface{}, offset interface{}, input interface{}, width int, order binary.ByteOrder) interface{} {
	start := zIndex(offset)
	reflected := zByteReflect(buffer)
	if start > reflected.Len() || width > reflected.Len()-start {
		panic(fmt.Sprintf("byte range [%d:%d] is outside buffer length %d", start, start+width, reflected.Len()))
	}
	number := zIntegerBigDynamic(input)
	_, max := zIntegerBounds(false, uint(width*8))
	if number.Sign() < 0 || number.Cmp(max) > 0 {
		panic(fmt.Sprintf("value %s is outside u%d range", number.String(), width*8))
	}
	temp := make([]byte, width)
	raw := number.Uint64()
	switch width {
	case 2: order.PutUint16(temp, uint16(raw))
	case 4: order.PutUint32(temp, uint32(raw))
	case 8: order.PutUint64(temp, raw)
	default: panic(fmt.Sprintf("unsupported integer width %d", width))
	}
	for i, value := range temp { zSetByte(buffer, start+i, value) }
	return buffer
}

func zWriteU16LE(buffer interface{}, offset interface{}, value interface{}) interface{} { return zWriteUnsigned(buffer, offset, value, 2, binary.LittleEndian) }
func zWriteU16BE(buffer interface{}, offset interface{}, value interface{}) interface{} { return zWriteUnsigned(buffer, offset, value, 2, binary.BigEndian) }
func zWriteU32LE(buffer interface{}, offset interface{}, value interface{}) interface{} { return zWriteUnsigned(buffer, offset, value, 4, binary.LittleEndian) }
func zWriteU32BE(buffer interface{}, offset interface{}, value interface{}) interface{} { return zWriteUnsigned(buffer, offset, value, 4, binary.BigEndian) }
func zWriteU64LE(buffer interface{}, offset interface{}, value interface{}) interface{} { return zWriteUnsigned(buffer, offset, value, 8, binary.LittleEndian) }
func zWriteU64BE(buffer interface{}, offset interface{}, value interface{}) interface{} { return zWriteUnsigned(buffer, offset, value, 8, binary.BigEndian) }

func zCopyBytes(destination interface{}, destinationStart interface{}, source interface{}, sourceStart interface{}, length interface{}) interface{} {
	dst := zByteReflect(destination)
	src := zByteData(source)
	dstStart := zIndex(destinationStart)
	srcStart := zIndex(sourceStart)
	count := zIndex(length)
	if dstStart > dst.Len() || count > dst.Len()-dstStart { panic("copyBytes destination range is outside buffer") }
	if srcStart > len(src) || count > len(src)-srcStart { panic("copyBytes source range is outside buffer") }
	temp := append([]byte(nil), src[srcStart:srcStart+count]...)
	for i, value := range temp { zSetByte(destination, dstStart+i, value) }
	return destination
}

func zBytesEqual(first interface{}, second interface{}) bool {
	return bytes.Equal(zByteData(first), zByteData(second))
}

func zSHA256(buffer interface{}) string {
	sum := sha256.Sum256(zByteData(buffer))
	return fmt.Sprintf("%x", sum)
}

func sizeOf(value interface{}) int {
	if value == nil { return 0 }
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return reflected.Len()
	default:
		return 0
	}
}

func toUppercase(s string) string {
	return strings.ToUpper(s)
}

func toLowercase(s string) string {
	return strings.ToLower(s)
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func removeWhiteSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "")
}

func replace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func addToArrayStart(arr []interface{}, elem interface{}) []interface{} {
	return append([]interface{}{elem}, arr...)
}

func addToArrayEnd(arr []interface{}, elem interface{}) []interface{} {
	return append(arr, elem)
}

func removeFromArray(arr []interface{}, index int) []interface{} {
	if index < 0 || index >= len(arr) {
		return arr
	}
	return append(arr[:index], arr[index+1:]...)
}

func max(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	maxVal := arr[0].(int)
	for _, v := range arr[1:] {
		val := v.(int)
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal
}

func min(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	minVal := arr[0].(int)
	for _, v := range arr[1:] {
		val := v.(int)
		if val < minVal {
			minVal = val
		}
	}
	return minVal
}

func first(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	return arr[0]
}

func last(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	return arr[len(arr)-1]
}

func allButFirst(arr []interface{}) []interface{} {
	if len(arr) == 0 {
		return arr
	}
	return arr[1:]
}

func indexOf(arr []interface{}, elem interface{}) int {
	for i, v := range arr {
		if v == elem {
			return i
		}
	}
	return -1
}

func organize(arr []interface{}, order string) []interface{} {
	intArr := make([]int, len(arr))
	for i, v := range arr {
		intArr[i] = v.(int)
	}
	if order == "desc" {
		sort.Sort(sort.Reverse(sort.IntSlice(intArr)))
	} else {
		sort.Ints(intArr)
	}
	result := make([]interface{}, len(intArr))
	for i, v := range intArr {
		result[i] = v
	}
	return result
}

func sum(arr []interface{}) interface{} {
	total := 0.0
	for _, v := range arr {
		switch val := v.(type) {
		case int:
			total += float64(val)
		case float64:
			total += val
		}
	}
	if float64(int(total)) == total {
		return int(total)
	}
	return total
}

type ZumbraDate struct {
	fullDate time.Time
	hour     int
	minute   int
	second   int
	day      int
	month    int
	year     int
}

func date() ZumbraDate {
	now := time.Now()
	return ZumbraDate{
		fullDate: now,
		hour:     now.Hour(),
		minute:   now.Minute(),
		second:   now.Second(),
		day:      now.Day(),
		month:    int(now.Month()),
		year:     now.Year(),
	}
}

func addToDict(dict map[string]interface{}, key string, value interface{}) map[string]interface{} {
	dict[key] = value
	return dict
}

func deleteFromDict(dict map[string]interface{}, key string) map[string]interface{} {
	delete(dict, key)
	return dict
}

func getFromDict(dict map[string]interface{}, key string) interface{} {
	return dict[key]
}

func dictKeys(dict map[string]interface{}) []string {
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	return keys
}

func dictValues(dict map[string]interface{}) []interface{} {
	values := make([]interface{}, 0, len(dict))
	for _, v := range dict {
		values = append(values, v)
	}
	return values
}

var EnvVars = map[string]string{}

func dotenvLoad(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("failed to open file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			EnvVars[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("failed to read file:", err)
	}
}

func dotenvGet(key string) interface{} {
	if value, ok := EnvVars[key]; ok {
		return value
	}
	return nil
}

func hashCode(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	hashInBytes := hash.Sum(nil)
	return fmt.Sprintf("%x", hashInBytes)
}

func input(prompt ...string) string {
	if len(prompt) > 0 {
		fmt.Print(prompt[0])
	}
	var value string
	fmt.Scanln(&value)
	return value
}

func bhaskara(a, b, c float64) interface{} {
	delta := (b * b) - (4 * a * c)
	if delta < 0 {
		return nil
	}
	if delta == 0 {
		return -b / (2 * a)
	}
	sqrtDelta := math.Sqrt(delta)
	x1 := (-b + sqrtDelta) / (2 * a)
	x2 := (-b - sqrtDelta) / (2 * a)
	return []interface{}{x1, x2}
}

func randomInteger(args ...int) int {
	min := 0
	max := 10
	if len(args) == 1 {
		max = args[0]
	} else if len(args) == 2 {
		min = args[0]
		max = args[1]
	}
	if min > max {
		min, max = max, min
	}
	return min + rand.Intn(max-min+1)
}

func randomFloat(args ...float64) float64 {
	min := 0.0
	max := 10.0
	if len(args) == 1 {
		max = args[0]
	} else if len(args) == 2 {
		min = args[0]
		max = args[1]
	}
	if min > max {
		min, max = max, min
	}
	return min + rand.Float64()*(max-min)
}

func toString(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int(math.Floor(v))
	case bool:
		if v {
			return 1
		}
		return 0
	case int:
		return v
	default:
		return 0
	}
}

func toFloat(value interface{}) float64 {
	switch v := value.(type) {
	case string:
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return n
	case float64:
		return v
	case bool:
		if v {
			return 1.0
		}
		return 0.0
	case int:
		return float64(v)
	default:
		return 0.0
	}
}

func toBool(value interface{}) bool {
	return isTruthy(value)
}


func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case string:
		return v != ""
	case bool:
		return v
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case float32:
		return v != 0
	case float64:
		return v != 0
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

func zOr(left interface{}, right interface{}) interface{} {
	if isTruthy(left) {
		return left
	}
	return right
}

func zAnd(left interface{}, right interface{}) interface{} {
	if isTruthy(left) {
		return right
	}
	return left
}

func jsonParse(input string) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	if err != nil {
		return map[string]interface{}{}
	}
	return result
}

var secretKey string

func jwtCreateToken(username string, secret string, expirationHours int) (string, error) {
	secretKey = secret

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * time.Duration(expirationHours)).Unix(),
	})

	tokenStr, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to create token: %v", err)
	}

	return tokenStr, nil
}

func jwtVerifyToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", errors.New("username not found in token")
		}
		return username, nil
	}

	return "", errors.New("invalid token")
}

var db_connection *sql.DB

func mysqlConnection(host, port, user, password, database string) error {
	var err error
	db_connection, err = sql.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+database)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	err = db_connection.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping: %v", err)
	}

	fmt.Printf("Database '%s' connected successfully\n", database)
	return nil
}

func mysqlCreateTable(tableName, fields string) error {
	_, err := db_connection.Exec("CREATE TABLE " + tableName + " (" + fields + ");")
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}
	fmt.Printf("Table '%s' created successfully\n", tableName)
	return nil
}

func mysqlShowTables() ([]string, error) {
	rows, err := db_connection.Query("SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("failed to show tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func mysqlShowTableColumns(tableName string) ([]string, error) {
	rows, err := db_connection.Query("SHOW COLUMNS FROM " + tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to show columns: %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var field, colType, null, key, extra string
		var defaultValue sql.NullString
		if err := rows.Scan(&field, &colType, &null, &key, &defaultValue, &extra); err != nil {
			return nil, err
		}
		columns = append(columns, field)
	}
	return columns, nil
}

func mysqlDeleteTable(tableName string) error {
	query := fmt.Sprintf("DROP TABLE %s", tableName)
	_, err := db_connection.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop table: %v", err)
	}
	fmt.Printf("Table '%s' deleted successfully\n", tableName)
	return nil
}

func mysqlDropTable(tableName string) error {
	return mysqlDeleteTable(tableName)
}

func mysqlInsertIntoTable(tableName string, data map[string]interface{}) error {
	keys := []string{}
	placeholders := []string{}
	args := []interface{}{}

	for key, value := range data {
		keys = append(keys, key)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, strings.Join(keys, ","), strings.Join(placeholders, ","))
	_, err := db_connection.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert: %v", err)
	}

	fmt.Println("Record inserted successfully")
	return nil
}

func mysqlGetFromTable(tableName, fields, condition string) ([]map[string]interface{}, error) {
	if db_connection == nil {
		return nil, errors.New("not connected")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", fields, tableName)
	if condition != "" {
		query += " WHERE " + condition
	}

	rows, err := db_connection.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v", err)
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	result := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := map[string]interface{}{}
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		result = append(result, rowMap)
	}

	return result, nil
}

func mysqlUpdateIntoTable(tableName string, data map[string]interface{}, condition string) error {
	assignments := []string{}
	args := []interface{}{}

	for key, value := range data {
		assignments = append(assignments, fmt.Sprintf("%s = ?", key))
		args = append(args, value)
	}

	query := fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(assignments, ", "))
	if condition != "" {
		query += " WHERE " + condition
	}

	_, err := db_connection.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update: %v", err)
	}
	fmt.Println("Record updated successfully")
	return nil
}

func mysqlDeleteFromTable(tableName, condition string) error {
	query := fmt.Sprintf("DELETE FROM %s", tableName)
	if condition != "" {
		query += " WHERE " + condition
	}
	_, err := db_connection.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete: %v", err)
	}
	fmt.Println("Record deleted successfully")
	return nil
}

func objectToPlainString(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

func toStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	default:
		return []string{}
	}
}

func decodeJSONBody(body []byte) interface{} {
	if len(body) == 0 {
		return nil
	}

	var decoded interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return string(body)
	}
	return decoded
}

var postgresDB *sql.DB
var redisClient *redis.Client
var supabaseURL string
var supabaseKey string

func postgresConnection(connectionString string) error {
	var err error
	postgresDB, err = sql.Open("postgres", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open postgres connection: %v", err)
	}

	if err := postgresDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping postgres: %v", err)
	}

	return nil
}

func postgresExec(query string) error {
	if postgresDB == nil {
		return errors.New("postgres is not connected")
	}

	_, err := postgresDB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to exec postgres query: %v", err)
	}

	return nil
}

func postgresQuery(query string) ([]map[string]interface{}, error) {
	if postgresDB == nil {
		return nil, errors.New("postgres is not connected")
	}

	rows, err := postgresDB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query postgres: %v", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres columns: %v", err)
	}

	result := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("failed to scan postgres row: %v", err)
		}

		rowMap := map[string]interface{}{}
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}

		result = append(result, rowMap)
	}

	return result, nil
}

func redisConnection(addr, password string, db int) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %v", err)
	}

	return nil
}

func redisSet(key string, value interface{}) error {
	if redisClient == nil {
		return errors.New("redis is not connected")
	}

	ctx := context.Background()
	if err := redisClient.Set(ctx, key, objectToPlainString(value), 0).Err(); err != nil {
		return fmt.Errorf("failed to set redis key: %v", err)
	}

	return nil
}

func redisGet(key string) interface{} {
	if redisClient == nil {
		return nil
	}

	ctx := context.Background()
	value, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return nil
	}

	return value
}

func redisDel(key string) error {
	if redisClient == nil {
		return errors.New("redis is not connected")
	}

	ctx := context.Background()
	if err := redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete redis key: %v", err)
	}

	return nil
}

func supabaseConnection(url, key string) {
	supabaseURL = strings.TrimRight(url, "/")
	supabaseKey = key
}

func supabaseRequest(method, path string, payload interface{}, prefer string, extraHeaders map[string]string) interface{} {
	var bodyReader io.Reader

	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil
		}
		bodyReader = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequest(method, supabaseURL+path, bodyReader)
	if err != nil {
		return nil
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/json")

	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return map[string]interface{}{
			"error":  true,
			"status": resp.StatusCode,
			"body":   string(body),
		}
	}

	return decodeJSONBody(body)
}

func supabaseSelect(table, selectQuery string) interface{} {
	query := "select=" + neturl.QueryEscape(selectQuery)
	return supabaseRequest("GET", "/rest/v1/"+table+"?"+query, nil, "", nil)
}

func supabaseQuery(table, queryString string) interface{} {
	qs := strings.TrimPrefix(queryString, "?")
	return supabaseRequest("GET", "/rest/v1/"+table+"?"+qs, nil, "", nil)
}

func supabaseSingle(table, queryString string) interface{} {
	qs := strings.TrimPrefix(queryString, "?")
	return supabaseRequest("GET", "/rest/v1/"+table+"?"+qs, nil, "", map[string]string{
		"Accept": "application/vnd.pgrst.object+json",
	})
}

func supabaseCount(table, filterQuery string) int {
	qs := strings.TrimPrefix(filterQuery, "?")
	if qs != "" {
		qs += "&"
	}
	qs += "select=*"

	req, err := http.NewRequest("GET", supabaseURL+"/rest/v1/"+table+"?"+qs, nil)
	if err != nil {
		return 0
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Prefer", "count=exact")
	req.Header.Set("Range", "0-0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	rangeHeader := resp.Header.Get("Content-Range")
	if rangeHeader == "" {
		return 0
	}

	parts := strings.Split(rangeHeader, "/")
	if len(parts) != 2 {
		return 0
	}

	total, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}

	return total
}

func supabaseInsert(table string, payload interface{}) interface{} {
	return supabaseRequest("POST", "/rest/v1/"+table, payload, "return=representation", nil)
}

func supabaseUpdate(table, filterQuery string, payload interface{}) interface{} {
	qs := strings.TrimPrefix(filterQuery, "?")
	path := "/rest/v1/" + table
	if qs != "" {
		path += "?" + qs
	}
	return supabaseRequest("PATCH", path, payload, "return=representation", nil)
}

func supabaseDelete(table, filterQuery string) interface{} {
	qs := strings.TrimPrefix(filterQuery, "?")
	path := "/rest/v1/" + table
	if qs != "" {
		path += "?" + qs
	}
	return supabaseRequest("DELETE", path, nil, "return=representation", nil)
}

func supabaseUpsert(table string, payload interface{}) interface{} {
	return supabaseRequest("POST", "/rest/v1/"+table, payload, "resolution=merge-duplicates,return=representation", nil)
}

func supabaseRpc(functionName string, payload interface{}) interface{} {
	return supabaseRequest("POST", "/rest/v1/rpc/"+functionName, payload, "", nil)
}

func supabaseStorageUpload(bucket, remotePath, localFilePath string) interface{} {
	fileBytes, err := os.ReadFile(localFilePath)
	if err != nil {
		return map[string]interface{}{"error": true, "body": err.Error()}
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, strings.TrimPrefix(remotePath, "/"))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(fileBytes))
	if err != nil {
		return map[string]interface{}{"error": true, "body": err.Error()}
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("x-upsert", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return map[string]interface{}{"error": true, "body": err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return map[string]interface{}{
			"error":  true,
			"status": resp.StatusCode,
			"body":   string(body),
		}
	}

	return decodeJSONBody(body)
}

func supabaseStorageDelete(bucket string, paths interface{}) interface{} {
	payload := map[string]interface{}{
		"prefixes": toStringSlice(paths),
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s", supabaseURL, bucket)
	reqBody, _ := json.Marshal(payload)

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return decodeJSONBody(body)
}

func supabaseStoragePublicUrl(bucket, path string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", supabaseURL, bucket, strings.TrimPrefix(path, "/"))
}

func supabaseStorageSignedUrl(bucket, path string, expiresIn int) interface{} {
	payload := map[string]interface{}{
		"expiresIn": expiresIn,
	}

	url := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", supabaseURL, bucket, strings.TrimPrefix(path, "/"))
	reqBody, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return decodeJSONBody(body)
}

func supabaseStorageDownload(bucket, path string) string {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, strings.TrimPrefix(path, "/"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return ""
	}

	return base64.StdEncoding.EncodeToString(body)
}

func supabaseAuthSignUp(email, password string) interface{} {
	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}
	return supabaseRequest("POST", "/auth/v1/signup", payload, "", nil)
}

func supabaseAuthSignIn(email, password string) interface{} {
	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}
	return supabaseRequest("POST", "/auth/v1/token?grant_type=password", payload, "", nil)
}

func switchCase(key interface{}, cases map[string]interface{}, defaultValue interface{}) interface{} {
	strKey := fmt.Sprintf("%v", key)
	if value, ok := cases[strKey]; ok {
		return value
	}
	return defaultValue
}

func createFile(path string, content string) string {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return ""
	}
	return path
}

func createTxt(path string, content string) string {
	return createFile(path, content)
}

func createCsv(path string, content string) string {
	return createFile(path, content)
}

func createDoc(path string, title string, content string) string {
	doc := fmt.Sprintf("<html><head><meta charset=\"utf-8\"><title>%s</title></head><body><h1>%s</h1><p>%s</p></body></html>", title, title, strings.ReplaceAll(content, "\n", "<br>"))
	return createFile(path, doc)
}

func createPdf(path string, title string, content string) string {
	pdfText := "TITLE: " + title + "\n\n" + content
	return createFile(path, pdfText)
}

type ZRequest struct {
	Method  string
	Path    string
	Params  map[string]interface{}
	Query   map[string]interface{}
	Headers map[string]interface{}
	Body    interface{}
	RawBody string
}

type ZResponse struct {
	StatusCode  int
	Headers     map[string]string
	Body        interface{}
	ContentType string
}

type ZRoute struct {
	Method       string
	Path         string
	Handler      func(*ZRequest, *ZResponse)
	AsyncHandler ZAsyncHandler
	IsAsync      bool
}

type StaticRoute struct {
	Prefix string
	Dir    string
}

var routes []ZRoute
var staticRoutes []StaticRoute

func newResponse() *ZResponse {
	return &ZResponse{
		StatusCode: 200,
		Headers:    map[string]string{},
	}
}

func responseStatus(res *ZResponse, code int) *ZResponse {
	res.StatusCode = code
	return res
}

func responseHeader(res *ZResponse, key, value string) *ZResponse {
	res.Headers[key] = value
	return res
}

func responseJSON(res *ZResponse, data interface{}) *ZResponse {
	res.ContentType = "application/json"
	res.Body = data
	return res
}

func responseSend(res *ZResponse, data interface{}) *ZResponse {
	res.Body = data
	return res
}

func responseHTML(res *ZResponse, content string) *ZResponse {
	res.ContentType = "text/html; charset=utf-8"
	res.Body = content
	return res
}

func html(content string) string {
	return content
}

func restGet(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "GET",
		Path:    path,
		Handler: handler,
	})
}

func restPost(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "POST",
		Path:    path,
		Handler: handler,
	})
}

func restPut(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "PUT",
		Path:    path,
		Handler: handler,
	})
}

func restDelete(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "DELETE",
		Path:    path,
		Handler: handler,
	})
}

func restPatch(path string, handler func(*ZRequest, *ZResponse)) {
	routes = append(routes, ZRoute{
		Method:  "PATCH",
		Path:    path,
		Handler: handler,
	})
}

func restGetAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "GET",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func restPostAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "POST",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func restPutAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "PUT",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func restDeleteAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "DELETE",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func restPatchAsync(path string, handler ZAsyncHandler) {
	routes = append(routes, ZRoute{
		Method:       "PATCH",
		Path:         path,
		AsyncHandler: handler,
		IsAsync:      true,
	})
}

func registerRoute(path string, handler string) {
	restGet(path, func(req *ZRequest, res *ZResponse) {
		responseSend(res, handler)
	})
}

func serveStatic(prefix, dir string) {
	staticRoutes = append(staticRoutes, StaticRoute{
		Prefix: prefix,
		Dir:    dir,
	})
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func matchRoute(method, path string) (*ZRoute, map[string]interface{}) {
	reqParts := splitPath(path)

	for _, route := range routes {
		if route.Method != method {
			continue
		}

		routeParts := splitPath(route.Path)
		if len(routeParts) != len(reqParts) {
			continue
		}

		params := map[string]interface{}{}
		matched := true

		for i := 0; i < len(routeParts); i++ {
			if strings.HasPrefix(routeParts[i], ":") {
				params[strings.TrimPrefix(routeParts[i], ":")] = reqParts[i]
				continue
			}

			if routeParts[i] != reqParts[i] {
				matched = false
				break
			}
		}

		if matched {
			routeCopy := route
			return &routeCopy, params
		}
	}

	return nil, nil
}

func normalizeQuery(values map[string][]string) map[string]interface{} {
	result := map[string]interface{}{}
	for key, arr := range values {
		if len(arr) == 1 {
			result[key] = arr[0]
		} else {
			tmp := make([]interface{}, 0, len(arr))
			for _, item := range arr {
				tmp = append(tmp, item)
			}
			result[key] = tmp
		}
	}
	return result
}

func normalizeHeaders(values map[string][]string) map[string]interface{} {
	result := map[string]interface{}{}
	for key, arr := range values {
		if len(arr) == 1 {
			result[strings.ToLower(key)] = arr[0]
		} else {
			tmp := make([]interface{}, 0, len(arr))
			for _, item := range arr {
				tmp = append(tmp, item)
			}
			result[strings.ToLower(key)] = tmp
		}
	}
	return result
}

func parseRequestBody(r *http.Request) (interface{}, string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	raw := string(bodyBytes)

	if len(bodyBytes) == 0 {
		return nil, raw
	}

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		var decoded interface{}
		if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
			return decoded, raw
		}
	}

	return raw, raw
}

func writeResponse(w http.ResponseWriter, res *ZResponse) {
	if res == nil {
		w.WriteHeader(200)
		return
	}

	for k, v := range res.Headers {
		w.Header().Set(k, v)
	}

	if res.StatusCode <= 0 {
		res.StatusCode = 200
	}

	if res.Body == nil {
		w.WriteHeader(res.StatusCode)
		return
	}

	if res.ContentType != "" {
		w.Header().Set("Content-Type", res.ContentType)
	}

	switch body := res.Body.(type) {
	case string:
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		w.WriteHeader(res.StatusCode)
		_, _ = w.Write([]byte(body))
	case []byte:
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.WriteHeader(res.StatusCode)
		_, _ = w.Write(body)
	default:
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(res.StatusCode)
		encoded, err := json.Marshal(body)
		if err != nil {
	_, _ = w.Write([]byte("{\"error\":true,\"body\":\"failed to encode response\"}"))
	return
}
		_, _ = w.Write(encoded)
	}
}

func server(port int) {
	mux := http.NewServeMux()

	for _, sr := range staticRoutes {
		mux.Handle(sr.Prefix+"/", http.StripPrefix(sr.Prefix, http.FileServer(http.Dir(sr.Dir))))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		route, params := matchRoute(r.Method, r.URL.Path)
		if route == nil {
			http.NotFound(w, r)
			return
		}

		body, rawBody := parseRequestBody(r)

		reqObj := &ZRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Params:  params,
			Query:   normalizeQuery(r.URL.Query()),
			Headers: normalizeHeaders(r.Header),
			Body:    body,
			RawBody: rawBody,
		}

		resObj := newResponse()

if route.IsAsync && route.AsyncHandler != nil {
	result := route.AsyncHandler(reqObj, resObj)

	if isErrorValue(result) {
		if resObj.StatusCode == 200 {
			resObj.StatusCode = 500
		}

		if resObj.Body == nil {
			resObj.ContentType = "application/json"
			resObj.Body = map[string]interface{}{
				"error": true,
				"body":  fmt.Sprintf("%v", result),
			}
		}
	}
} else if route.Handler != nil {
	route.Handler(reqObj, resObj)
}

writeResponse(w, resObj)
	})

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to bind to port %s. got %s\n", addr, err)
		return
	}

	fmt.Printf("Zumbra server started on port %d\n", port)
	_ = http.Serve(ln, mux)
}

type ZAsyncHandler func(*ZRequest, *ZResponse) interface{}

type ZTask struct {
	Value interface{}
	Err   interface{}
	Done  bool
}

func isErrorValue(v interface{}) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case error:
		return true
	case map[string]interface{}:
		if e, ok := val["error"].(bool); ok && e {
			return true
		}
		return false
	case *ZTask:
		return val.Err != nil
	default:
		return false
	}
}

func errorValue(message string) interface{} {
	return map[string]interface{}{
		"error":   true,
		"message": message,
	}
}

func panicValue(message string) interface{} {
	panic(message)
}

func awaitValue(v interface{}) interface{} {
	switch t := v.(type) {
	case *ZTask:
		if t.Err != nil {
			return t.Err
		}
		return t.Value
	default:
		return v
	}
}

func tryValue(v interface{}) interface{} {
	return awaitValue(v)
}


type zMethod func(self *zStructInstance, args ...interface{}) interface{}

type zStructDefinition struct {
	Name string
	Fields []string
	Methods map[string]zMethod
}

type zStructInstance struct {
	Definition *zStructDefinition
	Fields map[string]interface{}
}

type zEnumValue struct { EnumName string; Name string; Ordinal int }
type zEnumDefinition struct { Name string; Members map[string]zEnumValue }
type zMatchCase struct { Pattern interface{}; Body func() interface{} }

func zStruct(name string, fields []string, methods map[string]zMethod) *zStructDefinition {
	return &zStructDefinition{Name: name, Fields: fields, Methods: methods}
}

func zConstruct(definition *zStructDefinition, args ...interface{}) *zStructInstance {
	instance := &zStructInstance{Definition: definition, Fields: map[string]interface{}{}}
	if len(args) == 1 {
		if named, ok := args[0].(map[string]interface{}); ok {
			for _, field := range definition.Fields {
				value, exists := named[field]
				if !exists { panic(fmt.Sprintf("missing field %s for %s", field, definition.Name)) }
				instance.Fields[field] = value
			}
			for field := range named {
				known := false
				for _, declared := range definition.Fields { if field == declared { known = true; break } }
				if !known { panic(fmt.Sprintf("unknown field %s for %s", field, definition.Name)) }
			}
			return instance
		}
	}
	if len(args) != len(definition.Fields) { panic(fmt.Sprintf("wrong number of fields for %s: want=%d, got=%d", definition.Name, len(definition.Fields), len(args))) }
	for index, field := range definition.Fields { instance.Fields[field] = args[index] }
	return instance
}

func zEnum(name string, members []string) *zEnumDefinition {
	definition := &zEnumDefinition{Name: name, Members: map[string]zEnumValue{}}
	for index, member := range members { definition.Members[member] = zEnumValue{EnumName: name, Name: member, Ordinal: index} }
	return definition
}

func zGetAttr(value interface{}, property string) interface{} {
	switch current := value.(type) {
	case *zStructInstance:
		if field, ok := current.Fields[property]; ok { return field }
		if current.Definition != nil { if _, ok := current.Definition.Methods[property]; ok { return property } }
		panic(fmt.Sprintf("unknown field or method %s", property))
	case *zEnumDefinition:
		member, ok := current.Members[property]
		if !ok { panic(fmt.Sprintf("unknown enum member %s.%s", current.Name, property)) }
		return member
	case map[string]interface{}:
		return current[property]
	default:
		reflected := reflect.ValueOf(value)
		if reflected.Kind() == reflect.Pointer { reflected = reflected.Elem() }
		if reflected.IsValid() && reflected.Kind() == reflect.Struct {
			field := reflected.FieldByName(property)
			if field.IsValid() { return field.Interface() }
		}
		panic(fmt.Sprintf("object %T has no attribute %s", value, property))
	}
}

func zSetAttr(value interface{}, property string, fieldValue interface{}) interface{} {
	instance, ok := value.(*zStructInstance)
	if !ok { panic(fmt.Sprintf("attribute assignment requires struct, got %T", value)) }
	if _, exists := instance.Fields[property]; !exists { panic(fmt.Sprintf("unknown field %s", property)) }
	instance.Fields[property] = fieldValue
	return fieldValue
}

func zCallMethod(value interface{}, method string, args ...interface{}) interface{} {
	instance, ok := value.(*zStructInstance)
	if !ok { panic(fmt.Sprintf("method call requires struct, got %T", value)) }
	fn, ok := instance.Definition.Methods[method]
	if !ok { panic(fmt.Sprintf("unknown method %s", method)) }
	return fn(instance, args...)
}

func zMatch(value interface{}, cases []zMatchCase, fallback func() interface{}) interface{} {
	for _, candidate := range cases { if reflect.DeepEqual(value, candidate.Pattern) { return candidate.Body() } }
	if fallback != nil { return fallback() }
	return nil
}

func zIntegerTarget(left reflect.Value, right reflect.Value) reflect.Type {
	if left.Type() == right.Type() { return left.Type() }
	isInteger := func(kind reflect.Kind) bool {
		return (kind >= reflect.Int && kind <= reflect.Int64) || (kind >= reflect.Uint && kind <= reflect.Uint64)
	}
	if !isInteger(left.Kind()) || !isInteger(right.Kind()) { return nil }
	if left.Kind() == reflect.Int && right.Kind() != reflect.Int { return right.Type() }
	if right.Kind() == reflect.Int && left.Kind() != reflect.Int { return left.Type() }
	return nil
}

func zCoerceInteger(value reflect.Value, target reflect.Type) reflect.Value {
	number := zIntegerBigDynamic(value.Interface())
	signed := target.Kind() >= reflect.Int && target.Kind() <= reflect.Int64
	minimum, maximum := zIntegerBounds(signed, uint(target.Bits()))
	if number.Cmp(minimum) < 0 || number.Cmp(maximum) > 0 {
		panic(fmt.Sprintf("value %s is outside %s range", number.String(), target.String()))
	}
	result := reflect.New(target).Elem()
	if signed { result.SetInt(number.Int64()) } else { result.SetUint(number.Uint64()) }
	return result
}

func zSignedResult(target reflect.Type, value int64) interface{} {
	result := reflect.New(target).Elem()
	result.SetInt(value)
	return result.Interface()
}

func zUnsignedResult(target reflect.Type, value uint64) interface{} {
	result := reflect.New(target).Elem()
	result.SetUint(value)
	return result.Interface()
}

func zBinary(operator string, left interface{}, right interface{}) interface{} {
	if operator == "==" { return reflect.DeepEqual(left, right) }
	if operator == "!=" { return !reflect.DeepEqual(left, right) }
	if l, ok := left.(string); ok {
		r, rok := right.(string)
		if operator == "+" && rok { return l + r }
	}
	lv, rv := reflect.ValueOf(left), reflect.ValueOf(right)
	if !lv.IsValid() || !rv.IsValid() { panic("invalid binary operand") }
	isFloat := func(k reflect.Kind) bool { return k == reflect.Float32 || k == reflect.Float64 }
	isSigned := func(k reflect.Kind) bool { return k >= reflect.Int && k <= reflect.Int64 }
	isUnsigned := func(k reflect.Kind) bool { return k >= reflect.Uint && k <= reflect.Uint64 }
	if isFloat(lv.Kind()) || isFloat(rv.Kind()) {
		toFloat := func(v reflect.Value) float64 { if isFloat(v.Kind()) { return v.Convert(reflect.TypeOf(float64(0))).Float() }; if isSigned(v.Kind()) { return float64(v.Int()) }; return float64(v.Uint()) }
		l, r := toFloat(lv), toFloat(rv)
		switch operator { case "+": return l+r; case "-": return l-r; case "*": return l*r; case "/": return l/r; case "<": return l<r; case ">": return l>r; case "<=": return l<=r; case ">=": return l>=r }
	}
	if target := zIntegerTarget(lv, rv); target != nil {
		lv, rv = zCoerceInteger(lv, target), zCoerceInteger(rv, target)
	}
	if operator == "shl" || operator == "shr" {
		var count uint64
		if isSigned(rv.Kind()) {
			if rv.Int() < 0 { panic("shift count must be non-negative") }
			count = uint64(rv.Int())
		} else if isUnsigned(rv.Kind()) {
			count = rv.Uint()
		} else { panic(fmt.Sprintf("shift count must be integer, got %T", right)) }
		if count >= uint64(lv.Type().Bits()) { panic(fmt.Sprintf("shift count out of range: %d", count)) }
		if isSigned(lv.Kind()) {
			if operator == "shl" { return zSignedResult(lv.Type(), lv.Int()<<count) }
			return zSignedResult(lv.Type(), lv.Int()>>count)
		}
		if isUnsigned(lv.Kind()) {
			if operator == "shl" { return zUnsignedResult(lv.Type(), lv.Uint()<<count) }
			return zUnsignedResult(lv.Type(), lv.Uint()>>count)
		}
	}
	if isSigned(lv.Kind()) && isSigned(rv.Kind()) && lv.Type() == rv.Type() {
		l, r := lv.Int(), rv.Int()
		switch operator {
		case "+": return zSignedResult(lv.Type(), l+r)
		case "-": return zSignedResult(lv.Type(), l-r)
		case "*": return zSignedResult(lv.Type(), l*r)
		case "/": return zSignedResult(lv.Type(), l/r)
		case "%": return zSignedResult(lv.Type(), l%r)
		case "band": return zSignedResult(lv.Type(), l&r)
		case "bor": return zSignedResult(lv.Type(), l|r)
		case "bxor": return zSignedResult(lv.Type(), l^r)
		case "<": return l<r; case ">": return l>r; case "<=": return l<=r; case ">=": return l>=r
		}
	}
	if isUnsigned(lv.Kind()) && isUnsigned(rv.Kind()) && lv.Type() == rv.Type() {
		l, r := lv.Uint(), rv.Uint()
		switch operator {
		case "+": return zUnsignedResult(lv.Type(), l+r)
		case "-": return zUnsignedResult(lv.Type(), l-r)
		case "*": return zUnsignedResult(lv.Type(), l*r)
		case "/": return zUnsignedResult(lv.Type(), l/r)
		case "%": return zUnsignedResult(lv.Type(), l%r)
		case "band": return zUnsignedResult(lv.Type(), l&r)
		case "bor": return zUnsignedResult(lv.Type(), l|r)
		case "bxor": return zUnsignedResult(lv.Type(), l^r)
		case "<": return l<r; case ">": return l>r; case "<=": return l<=r; case ">=": return l>=r
		}
	}
	panic(fmt.Sprintf("unsupported binary operation %T %s %T", left, operator, right))
}

`
}
