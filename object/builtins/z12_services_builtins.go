package builtins

import (
	"bufio"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"zumbra/object"
)

func objectDictMap(value object.Object, name string) (map[string]object.Object, *object.Error) {
	if value == nil || value.Type() == object.NULL_OBJ {
		return map[string]object.Object{}, nil
	}
	dict, ok := value.(*object.Dict)
	if !ok {
		return nil, NewError("%s expects dictionary, got %s", name, value.Type())
	}
	result := make(map[string]object.Object, len(dict.Pairs))
	for _, pair := range dict.Pairs {
		key, ok := pair.Key.(*object.String)
		if !ok {
			return nil, NewError("%s dictionary keys must be strings", name)
		}
		result[key.Value] = pair.Value
	}
	return result, nil
}
func objectStringMap(value object.Object, name string) (map[string]string, *object.Error) {
	values, e := objectDictMap(value, name)
	if e != nil {
		return nil, e
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		stringValue, ok := value.(*object.String)
		if !ok {
			return nil, NewError("%s value %s must be string", name, key)
		}
		result[key] = stringValue.Value
	}
	return result, nil
}
func objectMapDict(values map[string]object.Object) object.Object {
	pairs := make(map[object.DictKey]object.DictPair, len(values))
	for name, value := range values {
		key := &object.String{Value: name}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: value}
	}
	return &object.Dict{Pairs: pairs}
}

// ---------------- typed configuration ----------------
func newConfig(values map[string]object.Object) *object.Config {
	copyValues := make(map[string]object.Object, len(values))
	for key, value := range values {
		copyValues[key] = value
	}
	return &object.Config{Values: copyValues, Secrets: map[string]bool{}}
}
func parseDotEnv(text string) map[string]object.Object {
	values := map[string]object.Object{}
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		values[key] = &object.String{Value: value}
	}
	return values
}
func flattenConfig(prefix string, value any, target map[string]object.Object) {
	switch current := value.(type) {
	case map[string]any:
		for key, item := range current {
			name := key
			if prefix != "" {
				name = prefix + "." + key
			}
			flattenConfig(name, item, target)
		}
	default:
		target[prefix] = goValueToObject(current)
	}
}
func ConfigLoadBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("configLoad expects path")
		}
		path, ok := args[0].(*object.String)
		if !ok {
			return NewError("configLoad path must be string")
		}
		raw, err := os.ReadFile(path.Value)
		if err != nil {
			return NewError("load config: %s", err)
		}
		values := map[string]object.Object{}
		if strings.HasSuffix(strings.ToLower(path.Value), ".json") {
			var decoded any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				return NewError("parse config JSON: %s", err)
			}
			flattenConfig("", decoded, values)
			delete(values, "")
		} else {
			values = parseDotEnv(string(raw))
		}
		return newConfig(values)
	}}
}
func ConfigFromBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("configFrom expects dictionary")
		}
		values, e := objectDictMap(args[0], "configFrom")
		if e != nil {
			return e
		}
		return newConfig(values)
	}}
}
func ConfigEnvBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) > 1 {
			return NewError("configEnv expects optional prefix")
		}
		prefix := ""
		if len(args) == 1 {
			value, ok := args[0].(*object.String)
			if !ok {
				return NewError("configEnv prefix must be string")
			}
			prefix = value.Value
		}
		values := map[string]object.Object{}
		for _, entry := range os.Environ() {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) != 2 || !strings.HasPrefix(parts[0], prefix) {
				continue
			}
			key := strings.TrimPrefix(parts[0], prefix)
			values[key] = &object.String{Value: parts[1]}
		}
		return newConfig(values)
	}}
}
func configArg(value object.Object, name string) (*object.Config, *object.Error) {
	config, ok := value.(*object.Config)
	if !ok {
		return nil, NewError("%s expects Config, got %s", name, value.Type())
	}
	return config, nil
}
func configLookup(config *object.Config, key string) (object.Object, bool) {
	config.Mu.RLock()
	defer config.Mu.RUnlock()
	value, ok := config.Values[key]
	return value, ok
}
func configKeyDefault(args []object.Object, name string) (*object.Config, string, object.Object, *object.Error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, "", nil, NewError("%s expects config, key and optional default", name)
	}
	config, e := configArg(args[0], name)
	if e != nil {
		return nil, "", nil, e
	}
	key, ok := args[1].(*object.String)
	if !ok {
		return nil, "", nil, NewError("%s key must be string", name)
	}
	var defaultValue object.Object = &object.Null{}
	if len(args) == 3 {
		defaultValue = args[2]
	}
	return config, key.Value, defaultValue, nil
}
func ConfigMergeBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("configMerge expects two configs")
		}
		left, e := configArg(args[0], "configMerge")
		if e != nil {
			return e
		}
		right, e := configArg(args[1], "configMerge")
		if e != nil {
			return e
		}
		values := map[string]object.Object{}
		secrets := map[string]bool{}
		left.Mu.RLock()
		for key, value := range left.Values {
			values[key] = value
		}
		for key, value := range left.Secrets {
			secrets[key] = value
		}
		left.Mu.RUnlock()
		right.Mu.RLock()
		for key, value := range right.Values {
			values[key] = value
		}
		for key, value := range right.Secrets {
			secrets[key] = value
		}
		right.Mu.RUnlock()
		result := newConfig(values)
		result.Secrets = secrets
		return result
	}}
}
func ConfigRequiredBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		config, key, _, e := configKeyDefault(args, "configRequired")
		if e != nil {
			return e
		}
		value, ok := configLookup(config, key)
		if !ok || value.Type() == object.NULL_OBJ {
			return NewError("required configuration %s is missing", key)
		}
		return value
	}}
}
func ConfigStringBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		config, key, defaultValue, e := configKeyDefault(args, "configString")
		if e != nil {
			return e
		}
		value, ok := configLookup(config, key)
		if !ok {
			value = defaultValue
		}
		switch current := value.(type) {
		case *object.String:
			return current
		case *object.Null:
			return current
		default:
			return &object.String{Value: current.Inspect()}
		}
	}}
}
func ConfigIntBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		config, key, defaultValue, e := configKeyDefault(args, "configInt")
		if e != nil {
			return e
		}
		value, ok := configLookup(config, key)
		if !ok {
			value = defaultValue
		}
		switch current := value.(type) {
		case *object.Integer:
			return current
		case *object.FixedInteger:
			return &object.Integer{Value: current.SignedValue()}
		case *object.Float:
			return &object.Integer{Value: int64(current.Value)}
		case *object.String:
			parsed, err := strconv.ParseInt(strings.TrimSpace(current.Value), 10, 64)
			if err != nil {
				return NewError("configuration %s is not an integer", key)
			}
			return &object.Integer{Value: parsed}
		case *object.Null:
			return current
		default:
			return NewError("configuration %s is not an integer", key)
		}
	}}
}
func ConfigFloatBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		config, key, defaultValue, e := configKeyDefault(args, "configFloat")
		if e != nil {
			return e
		}
		value, ok := configLookup(config, key)
		if !ok {
			value = defaultValue
		}
		switch current := value.(type) {
		case *object.Float:
			return current
		case *object.Integer:
			return &object.Float{Value: float64(current.Value)}
		case *object.String:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(current.Value), 64)
			if err != nil {
				return NewError("configuration %s is not a float", key)
			}
			return &object.Float{Value: parsed}
		case *object.Null:
			return current
		default:
			return NewError("configuration %s is not a float", key)
		}
	}}
}
func ConfigBoolBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		config, key, defaultValue, e := configKeyDefault(args, "configBool")
		if e != nil {
			return e
		}
		value, ok := configLookup(config, key)
		if !ok {
			value = defaultValue
		}
		switch current := value.(type) {
		case *object.Boolean:
			return current
		case *object.Integer:
			return NewBoolean(current.Value != 0)
		case *object.String:
			parsed, err := strconv.ParseBool(strings.TrimSpace(current.Value))
			if err != nil {
				return NewError("configuration %s is not a boolean", key)
			}
			return NewBoolean(parsed)
		case *object.Null:
			return current
		default:
			return NewError("configuration %s is not a boolean", key)
		}
	}}
}
func ConfigSecretBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("configSecret expects config and key")
		}
		config, e := configArg(args[0], "configSecret")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("configSecret key must be string")
		}
		config.Mu.Lock()
		config.Secrets[key.Value] = true
		config.Mu.Unlock()
		return config
	}}
}
func ConfigRedactedBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("configRedacted expects config")
		}
		config, e := configArg(args[0], "configRedacted")
		if e != nil {
			return e
		}
		config.Mu.RLock()
		values := map[string]object.Object{}
		for key, value := range config.Values {
			if config.Secrets[key] {
				values[key] = &object.String{Value: "[REDACTED]"}
			} else {
				values[key] = value
			}
		}
		config.Mu.RUnlock()
		return objectMapDict(values)
	}}
}

// ---------------- structured logging ----------------
var logLevels = map[string]int{"trace": 0, "debug": 1, "info": 2, "warn": 3, "error": 4, "fatal": 5}

type structuredLogger struct {
	mu     sync.Mutex
	name   string
	level  string
	fields map[string]object.Object
	writer io.Writer
	closer io.Closer
}

func (l *structuredLogger) Level() string { l.mu.Lock(); defer l.mu.Unlock(); return l.level }
func (l *structuredLogger) SetLevel(level string) error {
	level = strings.ToLower(level)
	if _, ok := logLevels[level]; !ok {
		return fmt.Errorf("unknown log level %q", level)
	}
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
	return nil
}
func (l *structuredLogger) With(fields map[string]object.Object) object.LoggerRuntime {
	l.mu.Lock()
	defer l.mu.Unlock()
	combined := map[string]object.Object{}
	for key, value := range l.fields {
		combined[key] = value
	}
	for key, value := range fields {
		combined[key] = value
	}
	return &structuredLogger{name: l.name, level: l.level, fields: combined, writer: l.writer}
}
func redactLogField(key string, value object.Object) object.Object {
	lower := strings.ToLower(key)
	if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "authorization") || strings.Contains(lower, "cookie") {
		return &object.String{Value: "[REDACTED]"}
	}
	return value
}
func (l *structuredLogger) Log(level, message string, fields map[string]object.Object) error {
	level = strings.ToLower(level)
	minimum, ok := logLevels[l.level]
	if !ok {
		minimum = 2
	}
	current, ok := logLevels[level]
	if !ok {
		return fmt.Errorf("unknown log level %q", level)
	}
	if current < minimum {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := map[string]any{"timestamp": time.Now().UTC().Format(time.RFC3339Nano), "level": level, "logger": l.name, "message": message}
	for key, value := range l.fields {
		entry[key] = objectToGoValue(redactLogField(key, value))
	}
	for key, value := range fields {
		entry[key] = objectToGoValue(redactLogField(key, value))
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = l.writer.Write(append(raw, '\n'))
	return err
}
func (l *structuredLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closer != nil {
		return l.closer.Close()
	}
	return nil
}
func loggerArg(value object.Object, name string) (*object.Logger, *object.Error) {
	logger, ok := value.(*object.Logger)
	if !ok || logger.Runtime == nil {
		return nil, NewError("%s expects Logger", name)
	}
	return logger, nil
}
func LoggerBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 3 {
			return NewError("logger expects name, optional level and optional file path")
		}
		name, ok := args[0].(*object.String)
		if !ok {
			return NewError("logger name must be string")
		}
		level := "info"
		if len(args) > 1 {
			value, ok := args[1].(*object.String)
			if !ok {
				return NewError("logger level must be string")
			}
			level = value.Value
		}
		var writer io.Writer = os.Stderr
		var closer io.Closer
		if len(args) > 2 {
			path, ok := args[2].(*object.String)
			if !ok {
				return NewError("logger path must be string")
			}
			file, err := os.OpenFile(path.Value, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return NewError("open log file: %s", err)
			}
			writer = file
			closer = file
		}
		runtime := &structuredLogger{name: name.Value, level: "info", fields: map[string]object.Object{}, writer: writer, closer: closer}
		if err := runtime.SetLevel(level); err != nil {
			return NewError("%s", err)
		}
		return &object.Logger{Runtime: runtime, Name: name.Value}
	}}
}
func LoggerWithBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("loggerWith expects logger and fields")
		}
		logger, e := loggerArg(args[0], "loggerWith")
		if e != nil {
			return e
		}
		fields, e := objectDictMap(args[1], "loggerWith")
		if e != nil {
			return e
		}
		return &object.Logger{Runtime: logger.Runtime.With(fields), Name: logger.Name}
	}}
}
func LoggerSetLevelBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("loggerSetLevel expects logger and level")
		}
		logger, e := loggerArg(args[0], "loggerSetLevel")
		if e != nil {
			return e
		}
		level, ok := args[1].(*object.String)
		if !ok {
			return NewError("level must be string")
		}
		if err := logger.Runtime.SetLevel(level.Value); err != nil {
			return NewError("%s", err)
		}
		return logger
	}}
}
func LoggerLogBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 3 || len(args) > 4 {
			return NewError("loggerLog expects logger, level, message and optional fields")
		}
		logger, e := loggerArg(args[0], "loggerLog")
		if e != nil {
			return e
		}
		level, ok1 := args[1].(*object.String)
		message, ok2 := args[2].(*object.String)
		if !ok1 || !ok2 {
			return NewError("loggerLog level and message must be strings")
		}
		fields := map[string]object.Object{}
		if len(args) == 4 {
			fields, e = objectDictMap(args[3], "loggerLog")
			if e != nil {
				return e
			}
		}
		if err := logger.Runtime.Log(level.Value, message.Value, fields); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func LoggerCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("loggerClose expects logger")
		}
		logger, e := loggerArg(args[0], "loggerClose")
		if e != nil {
			return e
		}
		if err := logger.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}

// ---------------- metrics ----------------
type metricHistogram struct {
	Count int64
	Sum   float64
	Min   float64
	Max   float64
}
type metricsStore struct {
	mu         sync.RWMutex
	counters   map[string]float64
	gauges     map[string]float64
	histograms map[string]metricHistogram
}

func canonicalLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = key + "=" + labels[key]
	}
	return "{" + strings.Join(parts, ",") + "}"
}
func metricKey(name string, labels map[string]string) string { return name + canonicalLabels(labels) }
func (m *metricsStore) CounterAdd(name string, delta float64, labels map[string]string) {
	m.mu.Lock()
	m.counters[metricKey(name, labels)] += delta
	m.mu.Unlock()
}
func (m *metricsStore) GaugeSet(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	m.gauges[metricKey(name, labels)] = value
	m.mu.Unlock()
}
func (m *metricsStore) HistogramObserve(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	key := metricKey(name, labels)
	h := m.histograms[key]
	if h.Count == 0 {
		h.Min = value
		h.Max = value
	} else {
		if value < h.Min {
			h.Min = value
		}
		if value > h.Max {
			h.Max = value
		}
	}
	h.Count++
	h.Sum += value
	m.histograms[key] = h
	m.mu.Unlock()
}
func (m *metricsStore) Snapshot() map[string]object.Object {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counters := map[string]object.Object{}
	gauges := map[string]object.Object{}
	histograms := map[string]object.Object{}
	for key, value := range m.counters {
		counters[key] = &object.Float{Value: value}
	}
	for key, value := range m.gauges {
		gauges[key] = &object.Float{Value: value}
	}
	for key, h := range m.histograms {
		histograms[key] = objectMapDict(map[string]object.Object{"count": &object.Integer{Value: h.Count}, "sum": &object.Float{Value: h.Sum}, "min": &object.Float{Value: h.Min}, "max": &object.Float{Value: h.Max}})
	}
	return map[string]object.Object{"counters": objectMapDict(counters), "gauges": objectMapDict(gauges), "histograms": objectMapDict(histograms)}
}
func (m *metricsStore) Reset() {
	m.mu.Lock()
	m.counters = map[string]float64{}
	m.gauges = map[string]float64{}
	m.histograms = map[string]metricHistogram{}
	m.mu.Unlock()
}
func metricsArg(value object.Object, name string) (*object.MetricsRegistry, *object.Error) {
	metrics, ok := value.(*object.MetricsRegistry)
	if !ok || metrics.Runtime == nil {
		return nil, NewError("%s expects MetricsRegistry", name)
	}
	return metrics, nil
}
func metricNumber(value object.Object, name string) (float64, *object.Error) {
	switch current := value.(type) {
	case *object.Integer:
		return float64(current.Value), nil
	case *object.Float:
		return current.Value, nil
	case *object.FixedInteger:
		return float64(current.SignedValue()), nil
	default:
		return 0, NewError("%s value must be numeric", name)
	}
}
func metricLabels(value object.Object, name string) (map[string]string, *object.Error) {
	if value == nil || value.Type() == object.NULL_OBJ {
		return map[string]string{}, nil
	}
	return objectStringMap(value, name)
}
func MetricsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("metrics expects no arguments")
		}
		return &object.MetricsRegistry{Runtime: &metricsStore{counters: map[string]float64{}, gauges: map[string]float64{}, histograms: map[string]metricHistogram{}}}
	}}
}
func metricUpdateBuiltin(kind string) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 3 || len(args) > 4 {
			return NewError("metric update expects registry, name, value and optional labels")
		}
		metrics, e := metricsArg(args[0], "metric")
		if e != nil {
			return e
		}
		name, ok := args[1].(*object.String)
		if !ok {
			return NewError("metric name must be string")
		}
		value, e := metricNumber(args[2], "metric")
		if e != nil {
			return e
		}
		labels := map[string]string{}
		if len(args) == 4 {
			labels, e = metricLabels(args[3], "metric labels")
			if e != nil {
				return e
			}
		}
		switch kind {
		case "counter":
			metrics.Runtime.CounterAdd(name.Value, value, labels)
		case "gauge":
			metrics.Runtime.GaugeSet(name.Value, value, labels)
		case "histogram":
			metrics.Runtime.HistogramObserve(name.Value, value, labels)
		}
		return NewBoolean(true)
	}}
}
func MetricsCounterBuiltin() *object.Builtin   { return metricUpdateBuiltin("counter") }
func MetricsGaugeBuiltin() *object.Builtin     { return metricUpdateBuiltin("gauge") }
func MetricsHistogramBuiltin() *object.Builtin { return metricUpdateBuiltin("histogram") }
func MetricsSnapshotBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("metricsSnapshot expects registry")
		}
		metrics, e := metricsArg(args[0], "metricsSnapshot")
		if e != nil {
			return e
		}
		return objectMapDict(metrics.Runtime.Snapshot())
	}}
}
func MetricsResetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("metricsReset expects registry")
		}
		metrics, e := metricsArg(args[0], "metricsReset")
		if e != nil {
			return e
		}
		metrics.Runtime.Reset()
		return NewBoolean(true)
	}}
}

// ---------------- tracing ----------------
type traceEvent struct {
	Name       string
	Time       time.Time
	Attributes map[string]object.Object
}
type traceSpanRuntime struct {
	mu         sync.Mutex
	traceID    string
	spanID     string
	parentID   string
	name       string
	start      time.Time
	end        time.Time
	status     string
	attributes map[string]object.Object
	events     []traceEvent
	active     bool
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := cryptorand.Read(buffer); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf("%0*x", bytes*2, now)
	}
	return hex.EncodeToString(buffer)
}
func newTraceSpan(name string, attributes map[string]object.Object, traceID, parentID string) *traceSpanRuntime {
	if traceID == "" {
		traceID = randomHex(16)
	}
	copyAttrs := map[string]object.Object{}
	for key, value := range attributes {
		copyAttrs[key] = value
	}
	return &traceSpanRuntime{traceID: traceID, spanID: randomHex(8), parentID: parentID, name: name, start: time.Now().UTC(), attributes: copyAttrs, active: true}
}
func (s *traceSpanRuntime) Child(name string, attributes map[string]object.Object) object.TraceSpanRuntime {
	s.mu.Lock()
	defer s.mu.Unlock()
	return newTraceSpan(name, attributes, s.traceID, s.spanID)
}
func (s *traceSpanRuntime) SetAttribute(key string, value object.Object) {
	s.mu.Lock()
	if s.active {
		s.attributes[key] = value
	}
	s.mu.Unlock()
}
func (s *traceSpanRuntime) AddEvent(name string, attributes map[string]object.Object) {
	s.mu.Lock()
	if s.active {
		s.events = append(s.events, traceEvent{Name: name, Time: time.Now().UTC(), Attributes: attributes})
	}
	s.mu.Unlock()
}
func (s *traceSpanRuntime) Active() bool    { s.mu.Lock(); defer s.mu.Unlock(); return s.active }
func (s *traceSpanRuntime) TraceID() string { return s.traceID }
func (s *traceSpanRuntime) SpanID() string  { return s.spanID }
func (s *traceSpanRuntime) Finish(status string) map[string]object.Object {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		s.active = false
		s.end = time.Now().UTC()
		s.status = status
	}
	events := make([]object.Object, 0, len(s.events))
	for _, event := range s.events {
		events = append(events, objectMapDict(map[string]object.Object{"name": &object.String{Value: event.Name}, "timestamp": &object.String{Value: event.Time.Format(time.RFC3339Nano)}, "attributes": objectMapDict(event.Attributes)}))
	}
	return map[string]object.Object{"traceId": &object.String{Value: s.traceID}, "spanId": &object.String{Value: s.spanID}, "parentId": &object.String{Value: s.parentID}, "name": &object.String{Value: s.name}, "start": &object.String{Value: s.start.Format(time.RFC3339Nano)}, "end": &object.String{Value: s.end.Format(time.RFC3339Nano)}, "durationMs": &object.Float{Value: float64(s.end.Sub(s.start).Microseconds()) / 1000}, "status": &object.String{Value: s.status}, "attributes": objectMapDict(s.attributes), "events": &object.Array{Elements: events}}
}
func traceArg(value object.Object, name string) (*object.TraceSpan, *object.Error) {
	span, ok := value.(*object.TraceSpan)
	if !ok || span.Runtime == nil {
		return nil, NewError("%s expects TraceSpan", name)
	}
	return span, nil
}
func TraceStartBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("traceStart expects name and optional attributes")
		}
		name, ok := args[0].(*object.String)
		if !ok {
			return NewError("trace name must be string")
		}
		attributes := map[string]object.Object{}
		var e *object.Error
		if len(args) == 2 {
			attributes, e = objectDictMap(args[1], "traceStart")
			if e != nil {
				return e
			}
		}
		return &object.TraceSpan{Runtime: newTraceSpan(name.Value, attributes, "", "")}
	}}
}
func TraceChildBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 || len(args) > 3 {
			return NewError("traceChild expects parent, name and optional attributes")
		}
		parent, e := traceArg(args[0], "traceChild")
		if e != nil {
			return e
		}
		name, ok := args[1].(*object.String)
		if !ok {
			return NewError("trace child name must be string")
		}
		attributes := map[string]object.Object{}
		if len(args) == 3 {
			attributes, e = objectDictMap(args[2], "traceChild")
			if e != nil {
				return e
			}
		}
		return &object.TraceSpan{Runtime: parent.Runtime.Child(name.Value, attributes)}
	}}
}
func TraceSetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("traceSet expects span, key and value")
		}
		span, e := traceArg(args[0], "traceSet")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("trace key must be string")
		}
		span.Runtime.SetAttribute(key.Value, args[2])
		return span
	}}
}
func TraceEventBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 || len(args) > 3 {
			return NewError("traceEvent expects span, name and optional attributes")
		}
		span, e := traceArg(args[0], "traceEvent")
		if e != nil {
			return e
		}
		name, ok := args[1].(*object.String)
		if !ok {
			return NewError("trace event name must be string")
		}
		attributes := map[string]object.Object{}
		if len(args) == 3 {
			attributes, e = objectDictMap(args[2], "traceEvent")
			if e != nil {
				return e
			}
		}
		span.Runtime.AddEvent(name.Value, attributes)
		return span
	}}
}
func TraceFinishBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("traceFinish expects span and optional status")
		}
		span, e := traceArg(args[0], "traceFinish")
		if e != nil {
			return e
		}
		status := "ok"
		if len(args) == 2 {
			value, ok := args[1].(*object.String)
			if !ok {
				return NewError("trace status must be string")
			}
			status = value.Value
		}
		return objectMapDict(span.Runtime.Finish(status))
	}}
}
func TraceActiveBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("traceActive expects span")
		}
		span, e := traceArg(args[0], "traceActive")
		if e != nil {
			return e
		}
		return NewBoolean(span.Runtime.Active())
	}}
}

// ---------------- persistent sessions ----------------
type sqliteSessionStore struct {
	database *sqliteHandle
	mu       sync.Mutex
	open     bool
}

func newSQLiteSessionStore(path string) (*sqliteSessionStore, error) {
	database, err := openSQLite(path)
	if err != nil {
		return nil, err
	}
	store := &sqliteSessionStore{database: database, open: true}
	_, err = database.Exec(`CREATE TABLE IF NOT EXISTS _zumbra_sessions (id TEXT PRIMARY KEY, data TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)`, emptySQLiteParams())
	if err != nil {
		database.Close()
		return nil, err
	}
	return store, nil
}
func sessionID() string { return randomHex(32) }
func sessionDataJSON(data map[string]object.Object) (string, error) {
	plain := map[string]any{}
	for key, value := range data {
		plain[key] = objectToGoValue(value)
	}
	raw, err := json.Marshal(plain)
	return string(raw), err
}
func parseSessionData(raw string) (map[string]object.Object, error) {
	var plain map[string]any
	if err := json.Unmarshal([]byte(raw), &plain); err != nil {
		return nil, err
	}
	result := map[string]object.Object{}
	for key, value := range plain {
		result[key] = goValueToObject(value)
	}
	return result, nil
}
func (s *sqliteSessionStore) IsOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.open && s.database != nil && s.database.IsOpen()
}
func (s *sqliteSessionStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.open || s.database == nil {
		return nil
	}
	err := s.database.Close()
	if err == nil {
		s.open = false
	}
	return err
}
func (s *sqliteSessionStore) Create(data map[string]object.Object, ttl time.Duration) (string, error) {
	id := sessionID()
	if err := s.Set(id, data, ttl); err != nil {
		return "", err
	}
	return id, nil
}
func (s *sqliteSessionStore) Set(id string, data map[string]object.Object, ttl time.Duration) error {
	encoded, err := sessionDataJSON(data)
	if err != nil {
		return err
	}
	now := time.Now().UTC().UnixMilli()
	expires := now + ttl.Milliseconds()
	if ttl <= 0 {
		expires = now + int64((24 * time.Hour).Milliseconds())
	}
	params := &object.Array{Elements: []object.Object{&object.String{Value: id}, &object.String{Value: encoded}, &object.Integer{Value: expires}, &object.Integer{Value: now}, &object.Integer{Value: now}}}
	_, err = s.database.Exec(`INSERT INTO _zumbra_sessions(id,data,expires_at,created_at,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data, expires_at=excluded.expires_at, updated_at=excluded.updated_at`, params)
	return err
}
func (s *sqliteSessionStore) Get(id string) (map[string]object.Object, bool, error) {
	rows, err := s.database.Query(`SELECT data, expires_at FROM _zumbra_sessions WHERE id = ?`, &object.Array{Elements: []object.Object{&object.String{Value: id}}})
	if err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	expires := rows[0]["expires_at"].(*object.Integer).Value
	if expires <= time.Now().UTC().UnixMilli() {
		_ = s.Delete(id)
		return nil, false, nil
	}
	data, err := parseSessionData(rows[0]["data"].(*object.String).Value)
	return data, true, err
}
func (s *sqliteSessionStore) Delete(id string) error {
	_, err := s.database.Exec(`DELETE FROM _zumbra_sessions WHERE id = ?`, &object.Array{Elements: []object.Object{&object.String{Value: id}}})
	return err
}
func (s *sqliteSessionStore) Rotate(id string, ttl time.Duration) (string, error) {
	data, found, err := s.Get(id)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("session not found")
	}
	newID := sessionID()
	tx, err := s.database.Begin()
	if err != nil {
		return "", err
	}
	encoded, err := sessionDataJSON(data)
	if err != nil {
		_ = tx.Rollback()
		return "", err
	}
	now := time.Now().UTC().UnixMilli()
	expires := now + ttl.Milliseconds()
	if ttl <= 0 {
		expires = now + int64((24 * time.Hour).Milliseconds())
	}
	_, err = tx.Exec(`INSERT INTO _zumbra_sessions(id,data,expires_at,created_at,updated_at) VALUES(?,?,?,?,?)`, &object.Array{Elements: []object.Object{&object.String{Value: newID}, &object.String{Value: encoded}, &object.Integer{Value: expires}, &object.Integer{Value: now}, &object.Integer{Value: now}}})
	if err == nil {
		_, err = tx.Exec(`DELETE FROM _zumbra_sessions WHERE id=?`, &object.Array{Elements: []object.Object{&object.String{Value: id}}})
	}
	if err != nil {
		_ = tx.Rollback()
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return newID, nil
}
func (s *sqliteSessionStore) Touch(id string, ttl time.Duration) (bool, error) {
	now := time.Now().UTC().UnixMilli()
	expires := now + ttl.Milliseconds()
	result, err := s.database.Exec(`UPDATE _zumbra_sessions SET expires_at=?, updated_at=? WHERE id=? AND expires_at>?`, &object.Array{Elements: []object.Object{&object.Integer{Value: expires}, &object.Integer{Value: now}, &object.String{Value: id}, &object.Integer{Value: now}}})
	return result.RowsAffected > 0, err
}
func (s *sqliteSessionStore) Cleanup() (int64, error) {
	result, err := s.database.Exec(`DELETE FROM _zumbra_sessions WHERE expires_at <= ?`, &object.Array{Elements: []object.Object{&object.Integer{Value: time.Now().UTC().UnixMilli()}}})
	return result.RowsAffected, err
}

type redisSessionStore struct {
	client *object.RedisClient
	prefix string
	open   bool
	mu     sync.Mutex
}

func newRedisSessionStore(client *object.RedisClient, prefix string) (*redisSessionStore, error) {
	if client == nil || client.Runtime == nil || !client.Runtime.IsOpen() {
		return nil, fmt.Errorf("Redis client is closed")
	}
	if prefix == "" {
		prefix = "zumbra:session:"
	}
	return &redisSessionStore{client: client, prefix: prefix, open: true}, nil
}
func (s *redisSessionStore) key(id string) string { return s.prefix + id }
func (s *redisSessionStore) IsOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.open && s.client != nil && s.client.Runtime != nil && s.client.Runtime.IsOpen()
}
func (s *redisSessionStore) Close() error {
	s.mu.Lock()
	s.open = false
	s.mu.Unlock()
	return nil
}
func (s *redisSessionStore) Create(data map[string]object.Object, ttl time.Duration) (string, error) {
	id := sessionID()
	if err := s.Set(id, data, ttl); err != nil {
		return "", err
	}
	return id, nil
}
func (s *redisSessionStore) Set(id string, data map[string]object.Object, ttl time.Duration) error {
	if !s.IsOpen() {
		return fmt.Errorf("Redis session store is closed")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return s.client.Runtime.Set(s.key(id), objectMapDict(data), ttl)
}
func (s *redisSessionStore) Get(id string) (map[string]object.Object, bool, error) {
	if !s.IsOpen() {
		return nil, false, fmt.Errorf("Redis session store is closed")
	}
	value, found, err := s.client.Runtime.Get(s.key(id))
	if err != nil || !found {
		return nil, found, err
	}
	data, errObj := objectDictMap(value, "Redis session data")
	if errObj != nil {
		return nil, false, fmt.Errorf("%s", errObj.Message)
	}
	return data, true, nil
}
func (s *redisSessionStore) Delete(id string) error {
	if !s.IsOpen() {
		return fmt.Errorf("Redis session store is closed")
	}
	_, err := s.client.Runtime.Delete([]string{s.key(id)})
	return err
}
func (s *redisSessionStore) Rotate(id string, ttl time.Duration) (string, error) {
	if !s.IsOpen() {
		return "", fmt.Errorf("Redis session store is closed")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	data, found, err := s.Get(id)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("session not found")
	}
	newID := sessionID()
	if atomic, ok := s.client.Runtime.(interface {
		RotateSession(oldKey, newKey string, value object.Object, ttl time.Duration) error
	}); ok {
		if err := atomic.RotateSession(s.key(id), s.key(newID), objectMapDict(data), ttl); err != nil {
			return "", err
		}
		return newID, nil
	}
	if err := s.client.Runtime.Set(s.key(newID), objectMapDict(data), ttl); err != nil {
		return "", err
	}
	if _, err := s.client.Runtime.Delete([]string{s.key(id)}); err != nil {
		_, _ = s.client.Runtime.Delete([]string{s.key(newID)})
		return "", err
	}
	return newID, nil
}
func (s *redisSessionStore) Touch(id string, ttl time.Duration) (bool, error) {
	if !s.IsOpen() {
		return false, fmt.Errorf("Redis session store is closed")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return s.client.Runtime.Expire(s.key(id), ttl)
}
func (s *redisSessionStore) Cleanup() (int64, error) {
	// Redis expires session keys automatically.
	if !s.IsOpen() {
		return 0, fmt.Errorf("Redis session store is closed")
	}
	return 0, nil
}

func sessionStoreArg(value object.Object, name string) (*object.SessionStore, *object.Error) {
	store, ok := value.(*object.SessionStore)
	if !ok || store.Runtime == nil {
		return nil, NewError("%s expects SessionStore", name)
	}
	return store, nil
}
func sessionData(value object.Object, name string) (map[string]object.Object, *object.Error) {
	return objectDictMap(value, name)
}
func SessionSQLiteBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sessionSQLite expects path")
		}
		path, ok := args[0].(*object.String)
		if !ok {
			return NewError("sessionSQLite path must be string")
		}
		runtime, err := newSQLiteSessionStore(path.Value)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SessionStore{Runtime: runtime, Driver: "sqlite"}
	}}
}
func SessionRedisBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 || len(args) > 2 {
			return NewError("sessionRedis expects RedisClient and optional key prefix")
		}
		client, ok := args[0].(*object.RedisClient)
		if !ok || client.Runtime == nil {
			return NewError("sessionRedis expects RedisClient")
		}
		prefix := "zumbra:session:"
		if len(args) == 2 {
			value, ok := args[1].(*object.String)
			if !ok {
				return NewError("sessionRedis prefix must be string")
			}
			prefix = value.Value
		}
		runtime, err := newRedisSessionStore(client, prefix)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.SessionStore{Runtime: runtime, Driver: "redis"}
	}}
}

func SessionCreateBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("sessionCreate expects store, data, ttlMs")
		}
		store, e := sessionStoreArg(args[0], "sessionCreate")
		if e != nil {
			return e
		}
		data, e := sessionData(args[1], "sessionCreate")
		if e != nil {
			return e
		}
		ttl, e := concurrencyInt(args[2], "sessionCreate")
		if e != nil {
			return e
		}
		id, err := store.Runtime.Create(data, time.Duration(ttl)*time.Millisecond)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.String{Value: id}
	}}
}
func SessionGetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("sessionGet expects store and id")
		}
		store, e := sessionStoreArg(args[0], "sessionGet")
		if e != nil {
			return e
		}
		id, ok := args[1].(*object.String)
		if !ok {
			return NewError("session id must be string")
		}
		data, found, err := store.Runtime.Get(id.Value)
		if err != nil {
			return NewError("%s", err)
		}
		if !found {
			return &object.Null{}
		}
		return objectMapDict(data)
	}}
}
func SessionSetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("sessionSet expects store, id, data, ttlMs")
		}
		store, e := sessionStoreArg(args[0], "sessionSet")
		if e != nil {
			return e
		}
		id, ok := args[1].(*object.String)
		if !ok {
			return NewError("session id must be string")
		}
		data, e := sessionData(args[2], "sessionSet")
		if e != nil {
			return e
		}
		ttl, e := concurrencyInt(args[3], "sessionSet")
		if e != nil {
			return e
		}
		if err := store.Runtime.Set(id.Value, data, time.Duration(ttl)*time.Millisecond); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func SessionDeleteBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("sessionDelete expects store and id")
		}
		store, e := sessionStoreArg(args[0], "sessionDelete")
		if e != nil {
			return e
		}
		id, ok := args[1].(*object.String)
		if !ok {
			return NewError("session id must be string")
		}
		if err := store.Runtime.Delete(id.Value); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}
func SessionRotateBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("sessionRotate expects store, id, ttlMs")
		}
		store, e := sessionStoreArg(args[0], "sessionRotate")
		if e != nil {
			return e
		}
		id, ok := args[1].(*object.String)
		if !ok {
			return NewError("session id must be string")
		}
		ttl, e := concurrencyInt(args[2], "sessionRotate")
		if e != nil {
			return e
		}
		newID, err := store.Runtime.Rotate(id.Value, time.Duration(ttl)*time.Millisecond)
		if err != nil {
			return NewError("%s", err)
		}
		return &object.String{Value: newID}
	}}
}
func SessionTouchBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("sessionTouch expects store, id, ttlMs")
		}
		store, e := sessionStoreArg(args[0], "sessionTouch")
		if e != nil {
			return e
		}
		id, ok := args[1].(*object.String)
		if !ok {
			return NewError("session id must be string")
		}
		ttl, e := concurrencyInt(args[2], "sessionTouch")
		if e != nil {
			return e
		}
		changed, err := store.Runtime.Touch(id.Value, time.Duration(ttl)*time.Millisecond)
		if err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(changed)
	}}
}
func SessionCleanupBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sessionCleanup expects store")
		}
		store, e := sessionStoreArg(args[0], "sessionCleanup")
		if e != nil {
			return e
		}
		count, err := store.Runtime.Cleanup()
		if err != nil {
			return NewError("%s", err)
		}
		return &object.Integer{Value: count}
	}}
}
func SessionCloseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sessionClose expects store")
		}
		store, e := sessionStoreArg(args[0], "sessionClose")
		if e != nil {
			return e
		}
		if err := store.Runtime.Close(); err != nil {
			return NewError("%s", err)
		}
		return NewBoolean(true)
	}}
}

// ---------------- fixed-window rate limiter (Z11 deferred item) ----------------
type rateBucket struct {
	start time.Time
	count int64
}
type fixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int64
	window  time.Duration
	buckets map[string]rateBucket
}

func (l *fixedWindowLimiter) Allow(key string) (bool, int64, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	bucket := l.buckets[key]
	if bucket.start.IsZero() || now.Sub(bucket.start) >= l.window {
		bucket = rateBucket{start: now}
	}
	if bucket.count >= l.limit {
		retry := l.window - now.Sub(bucket.start)
		if retry < 0 {
			retry = 0
		}
		return false, 0, retry
	}
	bucket.count++
	l.buckets[key] = bucket
	return true, l.limit - bucket.count, 0
}
func (l *fixedWindowLimiter) Reset(key string) { l.mu.Lock(); delete(l.buckets, key); l.mu.Unlock() }
func rateLimiterArg(value object.Object, name string) (*object.RateLimiter, *object.Error) {
	limiter, ok := value.(*object.RateLimiter)
	if !ok || limiter.Runtime == nil {
		return nil, NewError("%s expects RateLimiter", name)
	}
	return limiter, nil
}
func RateLimiterBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("rateLimiter expects limit and windowMs")
		}
		limit, e := concurrencyInt(args[0], "rateLimiter")
		if e != nil {
			return e
		}
		window, e := concurrencyInt(args[1], "rateLimiter")
		if e != nil {
			return e
		}
		if limit <= 0 || window <= 0 {
			return NewError("rateLimiter values must be positive")
		}
		return &object.RateLimiter{Runtime: &fixedWindowLimiter{limit: limit, window: time.Duration(window) * time.Millisecond, buckets: map[string]rateBucket{}}}
	}}
}
func RateAllowBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("rateAllow expects limiter and key")
		}
		limiter, e := rateLimiterArg(args[0], "rateAllow")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("rate key must be string")
		}
		allowed, remaining, retry := limiter.Runtime.Allow(key.Value)
		return &object.Dict{Pairs: func() map[object.DictKey]object.DictPair {
			values := map[string]object.Object{"allowed": NewBoolean(allowed), "remaining": &object.Integer{Value: remaining}, "retryAfterMs": &object.Integer{Value: retry.Milliseconds()}}
			return objectMapDict(values).(*object.Dict).Pairs
		}()}
	}}
}
func RateResetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("rateReset expects limiter and key")
		}
		limiter, e := rateLimiterArg(args[0], "rateReset")
		if e != nil {
			return e
		}
		key, ok := args[1].(*object.String)
		if !ok {
			return NewError("rate key must be string")
		}
		limiter.Runtime.Reset(key.Value)
		return NewBoolean(true)
	}}
}

func prependServiceReceiver(receiver object.Object, builtin *object.Builtin) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return builtin.Fn(append([]object.Object{receiver}, args...)...)
	}}
}
func ConfigMethod(config *object.Config, name string) object.Object {
	switch name {
	case "merge":
		return prependServiceReceiver(config, ConfigMergeBuiltin())
	case "required":
		return prependServiceReceiver(config, ConfigRequiredBuiltin())
	case "string":
		return prependServiceReceiver(config, ConfigStringBuiltin())
	case "int":
		return prependServiceReceiver(config, ConfigIntBuiltin())
	case "float":
		return prependServiceReceiver(config, ConfigFloatBuiltin())
	case "bool":
		return prependServiceReceiver(config, ConfigBoolBuiltin())
	case "secret":
		return prependServiceReceiver(config, ConfigSecretBuiltin())
	case "redacted":
		return prependServiceReceiver(config, ConfigRedactedBuiltin())
	default:
		return nil
	}
}
func LoggerMethod(logger *object.Logger, name string) object.Object {
	switch name {
	case "with":
		return prependServiceReceiver(logger, LoggerWithBuiltin())
	case "setLevel":
		return prependServiceReceiver(logger, LoggerSetLevelBuiltin())
	case "log":
		return prependServiceReceiver(logger, LoggerLogBuiltin())
	case "trace", "debug", "info", "warn", "error", "fatal":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return NewError("logger method expects message and optional fields")
			}
			forward := []object.Object{logger, &object.String{Value: name}, args[0]}
			if len(args) == 2 {
				forward = append(forward, args[1])
			}
			return LoggerLogBuiltin().Fn(forward...)
		}}
	case "close":
		return prependServiceReceiver(logger, LoggerCloseBuiltin())
	default:
		return nil
	}
}
func MetricsMethod(metrics *object.MetricsRegistry, name string) object.Object {
	switch name {
	case "counter":
		return prependServiceReceiver(metrics, MetricsCounterBuiltin())
	case "gauge":
		return prependServiceReceiver(metrics, MetricsGaugeBuiltin())
	case "observe":
		return prependServiceReceiver(metrics, MetricsHistogramBuiltin())
	case "snapshot":
		return prependServiceReceiver(metrics, MetricsSnapshotBuiltin())
	case "reset":
		return prependServiceReceiver(metrics, MetricsResetBuiltin())
	default:
		return nil
	}
}
func TraceSpanMethod(span *object.TraceSpan, name string) object.Object {
	switch name {
	case "child":
		return prependServiceReceiver(span, TraceChildBuiltin())
	case "set":
		return prependServiceReceiver(span, TraceSetBuiltin())
	case "event":
		return prependServiceReceiver(span, TraceEventBuiltin())
	case "finish":
		return prependServiceReceiver(span, TraceFinishBuiltin())
	case "active":
		return prependServiceReceiver(span, TraceActiveBuiltin())
	default:
		return nil
	}
}
func SessionStoreMethod(store *object.SessionStore, name string) object.Object {
	switch name {
	case "create":
		return prependServiceReceiver(store, SessionCreateBuiltin())
	case "get":
		return prependServiceReceiver(store, SessionGetBuiltin())
	case "set":
		return prependServiceReceiver(store, SessionSetBuiltin())
	case "delete":
		return prependServiceReceiver(store, SessionDeleteBuiltin())
	case "rotate":
		return prependServiceReceiver(store, SessionRotateBuiltin())
	case "touch":
		return prependServiceReceiver(store, SessionTouchBuiltin())
	case "cleanup":
		return prependServiceReceiver(store, SessionCleanupBuiltin())
	case "close":
		return prependServiceReceiver(store, SessionCloseBuiltin())
	default:
		return nil
	}
}
func RateLimiterMethod(limiter *object.RateLimiter, name string) object.Object {
	switch name {
	case "allow":
		return prependServiceReceiver(limiter, RateAllowBuiltin())
	case "reset":
		return prependServiceReceiver(limiter, RateResetBuiltin())
	default:
		return nil
	}
}
