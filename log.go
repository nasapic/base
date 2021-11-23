package base

import (
	"bytes"
	"encoding"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type logLevel = string
type logOutput = string

type (
	// Logger is a simple interface that loggers supplied as dependencies
	// to services and workers should implement.
	Logger interface {
		// Enabled informs whether the logger is enabled for this level.
		Enabled(level string) bool

		// SetLevel sets logger log level
		SetLevel(level string)

		// Logs a debug message with the given key/values.
		Debug(msg string, keysAndValues ...interface{})

		// Logs an info message with the given key/values.
		Info(msg string, keysAndValues ...interface{})

		// Error logs an error, with the given message and key/values
		Error(err error, msg string, keysAndValues ...interface{})
	}
)

type (
	LogLevels struct {
		None  logLevel
		Debug logLevel
		Info  logLevel
		Error logLevel
		All   logLevel
	}

	LogOutputs struct {
		KeyValue logOutput
		JSON     logOutput
	}
)

type (
	// StdLogger is a basic reference implementations of Logger interface based
	// on Go's standard logger provided for the sake of simplicity.
	// A custom implementation can be provided if needed.
	StdLogger struct {
		level     string
		prefix    string
		valuesStr string
		output    string
		logger    *log.Logger
	}
)

type (
	Marshalable interface {
		Log() interface{}
	}
)

const (
	timestampFmt = "2006-01-02 15:04:05.000000"
	callDepth    = 2
	noVal        = "[n/a]"
)

// LogLevel valid values
var LogLevel = &LogLevels{
	None:  "none",
	Debug: "debug",
	Info:  "info",
	Error: "error",
	All:   "all",
}

var LogOutput = &LogOutputs{
	KeyValue: "keyvalue",
	JSON:     "json",
}

// New returns base logger implemented using Go's standard log package,
// Example: base.New(log.LstdFlags)
func NewLogger(level, prefix, output string, flags ...int) *StdLogger {
	flag := 0
	if len(flags) > 0 {
		flag = flags[0]
	}

	return &StdLogger{
		level:     level,
		prefix:    prefix,
		valuesStr: "",
		output:    output,
		logger:    log.New(os.Stdout, "", flag),
	}
}

func (sl *StdLogger) Enabled(level string) bool {
	return sl.level == level
}

func (sl *StdLogger) SetLevel(level string) {
	sl.level = level
}

func (sl *StdLogger) Debug(msg string, kvList ...interface{}) {
	prefix, args := sl.FormatDebug(msg, kvList)

	if prefix != "" {
		args = prefix + ": " + args
	}

	_ = sl.logger.Output(callDepth, args)
}

func (sl *StdLogger) Info(msg string, kvList ...interface{}) {
	prefix, args := sl.FormatInfo(msg, kvList)

	if prefix != "" {
		args = prefix + ": " + args
	}

	_ = sl.logger.Output(callDepth, args)
}

func (sl *StdLogger) Error(err error, msg string, kvList ...interface{}) {
	prefix, args := sl.FormatError(err, msg, kvList)

	if prefix != "" {
		args = prefix + ": " + args
	}

	_ = sl.logger.Output(callDepth, args)
}

// Formatters

// FormatDebug build a debug log message as a string.
// The prefix will be empty if not set or when the output is rendered as JSON.
func (sl *StdLogger) FormatDebug(msg string, kvList []interface{}) (prefix, argsStr string) {
	args := make([]interface{}, 0, 64)
	prefix = sl.prefix

	if sl.jsonOutput() {
		args = append(args, "logger", prefix)
		prefix = ""
	}

	args = append(args, "ts", time.Now().Format(timestampFmt))
	args = append(args, "msg", msg)

	return prefix, sl.render(args, kvList)
}

// FormatInfo build an info log message as a string.
// The prefix will be empty if not set or when the output is rendered as JSON.
func (sl *StdLogger) FormatInfo(msg string, kvList []interface{}) (prefix, argsStr string) {
	args := make([]interface{}, 0, 64) // using a constant here impacts perf
	prefix = sl.prefix

	if sl.jsonOutput() {
		args = append(args, "logger", prefix)
		prefix = ""
	}

	args = append(args, "ts", time.Now().Format(timestampFmt))
	args = append(args, "msg", msg)

	return prefix, sl.render(args, kvList)
}

// FormatError build an error log message as a string.
// The prefix will be empty if not set or when the output is rendered as JSON.
func (sl *StdLogger) FormatError(err error, msg string, kvList []interface{}) (prefix, argsStr string) {
	args := make([]interface{}, 0, 64) // using a constant here impacts perf
	prefix = sl.prefix

	if sl.jsonOutput() {
		args = append(args, "logger", prefix)
		prefix = ""
	}

	args = append(args, "ts", time.Now().Format(timestampFmt))

	args = append(args, "msg", msg)

	var logErr interface{}

	if err != nil {
		logErr = err.Error()
	}

	args = append(args, "error", logErr)

	return prefix, sl.render(args, kvList)
}

func (sl *StdLogger) render(implicit, args []interface{}) string {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	if sl.jsonOutput() {
		buf.WriteByte('{')
	}

	vals := sl.rectify(implicit)

	sl.makeFlat(buf, vals, false, false)

	keepOn := len(implicit) > 0
	if len(sl.valuesStr) > 0 {
		if keepOn {
			if sl.jsonOutput() {
				buf.WriteByte(',')

			} else {
				buf.WriteByte(' ')
			}
		}

		keepOn = true

		buf.WriteString(sl.valuesStr)
	}

	vals = sl.rectify(args)

	sl.makeFlat(buf, vals, keepOn, true)

	if sl.jsonOutput() {
		buf.WriteByte('}')
	}

	return buf.String()
}

func (sl *StdLogger) rectify(kvList []interface{}) []interface{} {
	if len(kvList)%2 != 0 {
		kvList = append(kvList, noVal)
	}

	for i := 0; i < len(kvList); i += 2 {
		_, ok := kvList[i].(string)
		if !ok {
			kvList[i] = sl.nonStringKey(kvList[i])
		}
	}

	return kvList
}

func (sl *StdLogger) makeFlat(buf *bytes.Buffer, kvList []interface{}, keepOn bool, escapeKeys bool) []interface{} {
	if len(kvList)%2 != 0 {
		kvList = append(kvList, noVal)
	}

	for i := 0; i < len(kvList); i += 2 {
		k, ok := kvList[i].(string)
		if !ok {
			k = sl.nonStringKey(kvList[i])
			kvList[i] = k
		}
		v := kvList[i+1]

		if i > 0 || keepOn {
			if sl.jsonOutput() {
				buf.WriteByte(',')

			} else {
				buf.WriteByte(' ')
			}
		}

		if escapeKeys {
			buf.WriteString(sl.prettify(k))

		} else {
			buf.WriteByte('"')
			buf.WriteString(k)
			buf.WriteByte('"')
		}

		if sl.jsonOutput() {
			buf.WriteByte(':')

		} else {
			buf.WriteByte('=')
		}

		buf.WriteString(sl.prettifyValue(v))
	}
	return kvList
}

func (sl *StdLogger) nonStringKey(v interface{}) string {
	return fmt.Sprintf("%s", sl.stringify(v))
}

func (sl *StdLogger) stringify(val interface{}) string {
	const maxLen = 16

	stringed := sl.prettifyValue(val)
	if len(stringed) > maxLen {
		stringed = stringed[:maxLen]
	}

	return stringed
}

func (sl *StdLogger) prettify(str string) string {
	if sl.escape(str) {
		return strconv.Quote(str)
	}

	b := bytes.NewBuffer(make([]byte, 0, 1024))

	b.WriteByte('"')
	b.WriteString(str)
	b.WriteByte('"')

	return b.String()
}

func (sl *StdLogger) prettifyValue(value interface{}) string {
	printStructBraces := true // NOTE: Make this configurable

	if v, ok := value.(Marshalable); ok {
		value = v.Log()
	}

	switch v := value.(type) {
	case fmt.Stringer:
		value = v.String()

	case error:
		value = v.Error()
	}

	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case string:
		return sl.prettify(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case uintptr:
		return strconv.FormatUint(uint64(v), 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case complex64:
		return `"` + strconv.FormatComplex(complex128(v), 'f', -1, 64) + `"`
	case complex128:
		return `"` + strconv.FormatComplex(v, 'f', -1, 128) + `"`
	}

	buf := bytes.NewBuffer(make([]byte, 0, 256))
	t := reflect.TypeOf(value)
	if t == nil {
		return "null"
	}
	v := reflect.ValueOf(value)
	switch t.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.String:
		return sl.prettify(v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(int64(v.Int()), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(uint64(v.Uint()), 10)
	case reflect.Float32:
		return strconv.FormatFloat(float64(v.Float()), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Complex64:
		return `"` + strconv.FormatComplex(complex128(v.Complex()), 'f', -1, 64) + `"`
	case reflect.Complex128:
		return `"` + strconv.FormatComplex(v.Complex(), 'f', -1, 128) + `"`
	case reflect.Struct:
		if printStructBraces {
			buf.WriteByte('{')
		}
		for i := 0; i < t.NumField(); i++ {
			fld := t.Field(i)
			if fld.PkgPath != "" {
				continue
			}

			if !v.Field(i).CanInterface() {
				continue
			}

			name := ""
			omitempty := false
			if tag, found := fld.Tag.Lookup("json"); found {
				if tag == "-" {
					continue
				}
				if comma := strings.Index(tag, ","); comma != -1 {
					if n := tag[:comma]; n != "" {
						name = n
					}
					rest := tag[comma:]
					if strings.Contains(rest, ",omitempty,") || strings.HasSuffix(rest, ",omitempty") {
						omitempty = true
					}

				} else {
					name = tag
				}
			}

			if omitempty && sl.isEmpty(v.Field(i)) {
				continue
			}
			if i > 0 {
				buf.WriteByte(',')
			}
			if fld.Anonymous && fld.Type.Kind() == reflect.Struct && name == "" {
				buf.WriteString(sl.prettifyValue(v.Field(i).Interface()))
				continue
			}
			if name == "" {
				name = fld.Name
			}
			// field names can't contain characters which need escaping
			buf.WriteByte('"')
			buf.WriteString(name)
			buf.WriteByte('"')
			buf.WriteByte(':')
			buf.WriteString(sl.prettifyValue(v.Field(i).Interface()))
		}
		if printStructBraces {
			buf.WriteByte('}')
		}
		return buf.String()

	case reflect.Slice, reflect.Array:
		buf.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}

			e := v.Index(i)
			buf.WriteString(sl.prettifyValue(e.Interface()))
		}

		buf.WriteByte(']')
		return buf.String()

	case reflect.Map:
		buf.WriteByte('{')
		it := v.MapRange()
		i := 0
		for it.Next() {
			if i > 0 {
				buf.WriteByte(',')
			}

			keystr := ""
			if m, ok := it.Key().Interface().(encoding.TextMarshaler); ok {
				txt, err := m.MarshalText()
				if err != nil {
					keystr = fmt.Sprintf("<error-MarshalText: %s>", err.Error())
				} else {
					keystr = string(txt)
				}
				keystr = sl.prettify(keystr)
			} else {
				keystr = sl.prettifyValue(it.Key().Interface())
				if t.Key().Kind() != reflect.String {
					keystr = sl.prettify(keystr)
				}
			}
			buf.WriteString(keystr)
			buf.WriteByte(':')
			buf.WriteString(sl.prettifyValue(it.Value().Interface()))
			i++
		}
		buf.WriteByte('}')
		return buf.String()
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return "null"
		}
		return sl.prettifyValue(v.Elem().Interface())
	}

	return fmt.Sprintf(`"[%s]"`, t.Kind().String())
}

func (sl *StdLogger) isEmpty(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return val.Complex() == 0
	case reflect.Interface, reflect.Ptr:
		return val.IsNil()
	}
	return false
}

func (sl *StdLogger) escape(str string) bool {
	for _, s := range str {
		if !strconv.IsPrint(s) || s == '\\' || s == '"' {
			return true
		}
	}

	return false
}

func (sl *StdLogger) jsonOutput() bool {
	return sl.level == LogOutput.JSON
}
