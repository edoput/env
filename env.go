// TODO(edoput) HELP=t should print out the usage equivalent of the EnvSet
package env

import (
  "encoding"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// errParse is returned by Set if a variable's value fails to parse,
// such as with an invalid integer for Int.
// It then gets wrapped through failf to provide more information.
var errParse = errors.New("parse error")

// errRange is returned by Set if a variable's value is out of range.
// It then gets wrapped through failf to provide more information.
var errRange = errors.New("value out of range")

func numError(err error) error {
	ne, ok := err.(*strconv.NumError)
	if !ok {
		return err
	}
	if ne.Err == strconv.ErrSyntax {
		return errParse
	}
	if ne.Err == strconv.ErrRange {
		return errRange
	}
	return err
}

// -- boolValue
type boolValue bool

func newBoolValue(val bool, p *bool) *boolValue {
	*p = val
	return (*boolValue)(p)
}

func (b *boolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		err = errParse
	}
	*b = boolValue(v)
	return err
}

func (b *boolValue) Get() any { return bool(*b) }

func (b *boolValue) String() string { return strconv.FormatBool(bool(*b)) }

func (b *boolValue) IsBoolVar() bool { return true }

type boolVar interface {
	Value
	IsBoolVar() bool
}

// -- intValue
type intValue int

func newIntValue(val int, p *int) *intValue {
	*p = val
	return (*intValue)(p)
}

func (b *intValue) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, strconv.IntSize)
	if err != nil {
		err = numError(err)
	}
	*b = intValue(v)
	return err
}

func (b *intValue) Get() any { return int(*b) }

func (b *intValue) String() string { return strconv.Itoa(int(*b)) }

// -- int64Value
type int64Value int64

func newInt64Value(val int64, p *int64) *int64Value {
	*p = val
	return (*int64Value)(p)
}

func (i *int64Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		err = numError(err)
	}
	*i = int64Value(v)
	return err
}

func (i *int64Value) Get() any { return int64(*i) }

func (i *int64Value) String() string { return strconv.FormatInt(int64(*i), 10) }

// -- uintValue
type uintValue uint

func newUintValue(val uint, p *uint) *uintValue {
	*p = val
	return (*uintValue)(p)
}

func (u *uintValue) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, strconv.IntSize)
	if err != nil {
		err = numError(err)
	}
	*u = uintValue(v)
	return err
}

func (u *uintValue) Get() any { return uint(*u) }

func (u *uintValue) String() string { return strconv.FormatUint(uint64(*u), 10) }

// -- uint64Value
type uint64Value uint64

func newUint64Value(val uint64, p *uint64) *uint64Value {
	*p = val
	return (*uint64Value)(p)
}

func (u *uint64Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		err = numError(err)
	}
	*u = uint64Value(v)
	return err
}

func (u *uint64Value) Get() any { return uint64(*u) }

func (u *uint64Value) String() string { return strconv.FormatUint(uint64(*u), 10) }

// -- stringValue
type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Get() any { return string(*s) }

func (s *stringValue) String() string { return string(*s) }

// -- float64Value
type float64Value float64

func newFloat64Value(val float64, p *float64) *float64Value {
	*p = val
	return (*float64Value)(p)
}

func (f *float64Value) Set(s string) error {
  v, err := strconv.ParseFloat(s, 64)
	if err != nil {
			return numError(err)
	}
	*f = float64Value(v)
	return nil
}

func (s *float64Value) Get() any { return float64(*s) }

func (s *float64Value) String() string { return strconv.FormatFloat(float64(*s), 'g', -1, 64) }

// -- durationValue
type durationValue time.Duration

func newDurationValue(val time.Duration, p *time.Duration) *durationValue {
	*p = val
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) error {
  v, err := time.ParseDuration(s)
	if err != nil {
			return errParse
	}
	*d = durationValue(v)
	return nil
}

func (d *durationValue) Get() any { return time.Duration(*d) }

func (d *durationValue) String() string { return time.Duration(*d).String() }

// -- textValue
type textValue struct { p encoding.TextUnmarshaler }

func newTextValue (val encoding.TextUnmarshaler, p encoding.TextUnmarshaler) textValue {
		ptrVal := reflect.ValueOf(p)
		if ptrVal.Kind() != reflect.Ptr {
				panic("variable value type must be a pointer")
		}
		defVal := reflect.ValueOf(val)
		if defVal.Kind() != reflect.Ptr {
				defVal = defVal.Elem()
		}
		if defVal.Type() != ptrVal.Type().Elem() {
				panic(fmt.Sprintf("default type value does not match variable type: %v != %v", defVal.Type(), ptrVal.Type().Elem()))
		}
		ptrVal.Elem().Set(defVal)
		return textValue{p}
}

func (v textValue) Set(s string) error {
		return v.p.UnmarshalText([]byte(s))
}

func (v textValue) Get() any {
		return v.p
}

func (v textValue) String() string {
		if m, ok := v.p.(encoding.TextMarshaler); ok {
				if b, err := m.MarshalText(); err == nil {
						return string(b)
				}
		}
		return ""
}

// -- funcValue
type funcValue func (string) error

func (f funcValue) Set(s string) error { return f(s) }

func (f funcValue) String() string { return "" }

func (f funcValue) Get() any {
		return nil
}

// -- boolFuncValue
type boolFuncValue func(string) error

func (f boolFuncValue) Set(s string) error { return f(s) }

func (f boolFuncValue) String() string { return "" }

func (f boolFuncValue) Get() any { return nil }

// Value is the interface to the dynamic value stored in a Spec.
// (The default value is represented as a string.)
//
// Set is called once, in declaration order, for each variable present.
// The env package may call the [String] method with a zero-valued receiver,
// such as a nil pointer.
//
// Get allows the contents of Value to be retrieved.
type Value interface {
	String() string
	Set(string) error
	Get() any
}

// ErrorHandling defines how [EnvSet.Parse] behaves if the parse fails.
type ErrorHandling int

// These constants cause [EnvSet.Parse] to behave as described if the parse fails.
const (
	ContinueOnError ErrorHandling = iota // returns a descriptive error
	ExitOnError                          // Call os.Exit(2)
	PanicOnError                         // Call panic with a descriptive error
)

// A EnvSet represents a set of defined environment variables. The zero value of a EnvSet
// has no name.
//
// [Var] name must be unique within an EnvSet. An attempt to define a variable whose
// name is already in use will cause a panic.
type EnvSet struct {
	// Usage is the function called when an error occur while parsing environment variables.
	// The field is a function (not a method) that may be changed to point to a
	// custom error handler. What happens after Usage is called depends on the
	// ErrorHandling setting; this defaults to ExitOnError, which exits the program
	// after calling Usage.
	Usage func()

	name          string
	parsed        bool
	actual        map[string]*Spec
	formal        map[string]*Spec
	environment   []string
	errorHandling ErrorHandling
	output        io.Writer         // nil means stderr; use Output() accessor
	undef         map[string]string // variables which didn't exists at the time of set
}

type Spec struct {
	Name        string // name as it appears in environment
	Description string // short description
	Value       Value  // value as set
	DefValue    string // default value (as text); for description message
}

// Output returns the destination for description and errro messages. [os.Stderr] is returned if
// output was not set or was set to nil.
func (e *EnvSet) Output() io.Writer {
	if e.output == nil {
		return os.Stderr
	}
	return e.output
}

// Name returns the name of the environment set.
func (e *EnvSet) Name() string {
	return e.name
}

// PrintDefaults print, to standard error unless configured otherwise, the
// default values of all defined environment variables in the set. See the
// documentation for the global function PrintDefaults for more information.
func (e *EnvSet) PrintDefaults() {
}

// PrintDefaults print, to standard error unless configured othersie, the
// default values of all defined environment variables.
// For an integer valued flag x, the default output has the form
//
// x int
// description-message-for-x (default: 7)
func PrintDefaults() {
		Environment.PrintDefaults()
}

func (e *EnvSet) defaultUsage() {
		if e.name == "" {
				fmt.Fprintf(e.Output(), "Environment:\n")
		} else {
				fmt.Fprintf(e.Output(), "Environment of %s:\n", e.name)
		}
		e.PrintDefaults()
}

func (e *EnvSet) BoolVar(p *bool, name string, value bool, description string) {
	e.Var(newBoolValue(value, p), name, description)
}

func BoolVar(p *bool, name string, value bool, description string) {
	Environment.Var(newBoolValue(value, p), name, description)
}

func (e *EnvSet) Bool(name string, value bool, description string) *bool {
	p := new(bool)
	e.Var(newBoolValue(value, p), name, description)
	return p
}

func Bool(name string, value bool, description string) *bool {
	return Environment.Bool(name, value, description)
}

func (e *EnvSet) IntVar(p *int, name string, value int, description string) {
	e.Var(newIntValue(value, p), name, description)
}

func IntVar(p *int, name string, value int, description string) {
	Environment.Var(newIntValue(value, p), name, description)
}

func (e *EnvSet) Int(name string, value int, description string) *int {
	p := new(int)
	e.Var(newIntValue(value, p), name, description)
	return p
}

func Int(name string, value int, description string) *int {
	return Environment.Int(name, value, description)
}

func (e *EnvSet) Int64Var(p *int64, name string, value int64, description string) {
	e.Var(newInt64Value(value, p), name, description)
}

func Int64Var(p *int64, name string, value int64, description string) {
	Environment.Var(newInt64Value(value, p), name, description)
}

func (e *EnvSet) Int64(name string, value int64, description string) *int64 {
	p := new(int64)
	e.Var(newInt64Value(value, p), name, description)
	return p
}

func Int64(name string, value int64, description string) *int64 {
	return Environment.Int64(name, value, description)
}

func (e *EnvSet) UintVar(p *uint, name string, value uint, description string) {
	e.Var(newUintValue(value, p), name, description)
}

func UintVar(p *uint, name string, value uint, description string) {
	Environment.Var(newUintValue(value, p), name, description)
}

func (e *EnvSet) Uint(name string, value uint, description string) *uint {
	p := new(uint)
	e.Var(newUintValue(value, p), name, description)
	return p
}

func Uint(name string, value uint, description string) *uint {
	return Environment.Uint(name, value, description)
}

func (e *EnvSet) Uint64Var(p *uint64, name string, value uint64, description string) {
	e.Var(newUint64Value(value, p), name, description)
}

func Uint64Var(p *uint64, name string, value uint64, description string) {
	Environment.Var(newUint64Value(value, p), name, description)
}

func (e *EnvSet) Uint64(name string, value uint64, description string) *uint64 {
	p := new(uint64)
	e.Var(newUint64Value(value, p), name, description)
	return p
}

func Uint64(name string, value uint64, description string) *uint64 {
	return Environment.Uint64(name, value, description)
}

func (e *EnvSet) StringVar(p *string, name string, value string, description string) {
	e.Var(newStringValue(value, p), name, description)
}

func StringVar(p *string, name string, value string, description string) {
	Environment.Var(newStringValue(value, p), name, description)
}

func (e *EnvSet) String(name string, value string, description string) *string {
	p := new(string)
	e.Var(newStringValue(value, p), name, description)
	return p
}

func String(name string, value string, description string) *string {
	return Environment.String(name, value, description)
}

func (e *EnvSet) Float64Var(p *float64, name string, value float64, description string) {
	e.Var(newFloat64Value(value, p), name, description)
}

func Float64Var(p *float64, name string, value float64, description string) {
	Environment.Var(newFloat64Value(value, p), name, description)
}

func (e *EnvSet) Float64(name string, value float64, description string) *float64 {
	p := new(float64)
	e.Var(newFloat64Value(value, p), name, description)
	return p
}

func Float64(name string, value float64, description string) *float64 {
	return Environment.Float64(name, value, description)
}

func (e *EnvSet) DurationVar(p *time.Duration, name string, value time.Duration, description string) {
	e.Var(newDurationValue(value, p), name, description)
}

func DurationVar(p *time.Duration, name string, value time.Duration, description string) {
	Environment.Var(newDurationValue(value, p), name, description)
}

func (e *EnvSet) Duration(name string, value time.Duration, description string) *time.Duration {
	p := new(time.Duration)
	e.Var(newDurationValue(value, p), name, description)
	return p
}

func Duration(name string, value time.Duration, description string) *time.Duration {
	return Environment.Duration(name, value, description)
}

func (e *EnvSet) TextVar(p encoding.TextUnmarshaler, name string, value encoding.TextUnmarshaler, description string) {
	e.Var(newTextValue(value, p), name, description)
}

func TextVar(p encoding.TextUnmarshaler, name string, value encoding.TextUnmarshaler, description string) {
	Environment.Var(newTextValue(value, p), name, description)
}

func (e *EnvSet) Func(name, usage string, fn func(string) error) {
	e.Var(funcValue(fn), name, usage)
}

func Func(name, usage string, fn func(string) error) {
	Environment.Func(name, usage, fn)
}

func (e *EnvSet) BoolFunc(name, usage string, fn func(string) error) {
	e.Var(boolFuncValue(fn), name, usage)
}

func BoolFunc(name, usage string, fn func(string) error) {
	Environment.BoolFunc(name, usage, fn)
}

// Var defines an environment variable with the specified name and description string. They type and
// value of the variable are represented by the first argument, of type [Value], which typically holds
// a user-defined implementation of [Value]. For instance, the caller could create a flag that turns
// a comma-separated string into a slice of strings by giving the slice the methods of [Value]; in
// particular, [Set] would decompose the comma-separated string into the slice.
func (e *EnvSet) Var(value Value, name string, description string) {
	if strings.Contains(name, "=") {
		panic(e.sprintf("variable %q contains =", name))
	}

	// Remember the default value as a string; it won't change.
	v := &Spec{Name: name, Description: description, Value: value, DefValue: value.String()}
	_, alreadyThere := e.formal[name]
	if alreadyThere {
		var msg string
		if e.name == "" {
			msg = e.sprintf("variable redefined: %s", name)
		} else {
			msg = e.sprintf("%s variable redefined: %s", e.name, name)
		}
		panic(msg) // happens only if variables are declared with identical names
	}
	if pos := e.undef[name]; pos != "" {
		panic(fmt.Sprintf("variable %s set at %s before being defined", name, pos))
	}
	if e.formal == nil {
		e.formal = make(map[string]*Spec)
	}
	e.formal[name] = v
}

func Var(value Value, name string, description string) {
	Environment.Var(value, name, description)
}

// sprintf formats the message, prints it to output, and returns it.
func (e *EnvSet) sprintf(format string, a ...any) string {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(e.Output(), msg)
	return msg
}

func (e *EnvSet) failf(format string, a ...any) error {
	msg := e.sprintf(format, a...)
	return errors.New(msg)
}

// usage calls the Usage method for the env set if one is specified,
// or the appropriate default usage function otherwise.
func (e *EnvSet) usage() {
	if e.Usage == nil {
		e.defaultUsage()
	} else {
		e.Usage()
	}
}

// parseOne parses one variable. It reports wether a variable was seen.
func (e *EnvSet) parseOne() (error, bool) {
	if len(e.environment) == 0 {
		return nil, true
	}
	s := e.environment[0]
	e.environment = e.environment[1:]
	// assume there are two strings now, name and value
	name, value, _ := strings.Cut(s, "=")
	spec, ok := e.formal[name]
	if !ok {
		// saw an environment variable that is not in the list we want
		return nil, false
	}
	if err := spec.Value.Set(value); err != nil {
		return e.failf("invalid value %q for variable %s: %v", value, name, err), false
	}
	if e.actual == nil {
		e.actual = make(map[string]*Spec)
	}
	e.actual[name] = spec
	return nil, false
}

// Parse parses variables definitions from the environment list.
// Must be called after all variables in the [EnvSet] are defined
// and before the variables are accessed by the program.
func (e *EnvSet) Parse(environment []string) error {
	e.parsed = true
	e.environment = environment
	for {
		err, done := e.parseOne()
		if done {
			break
		}
		if err == nil {
			continue
		}
		switch e.errorHandling {
		case ContinueOnError:
			return err
		case ExitOnError:
			os.Exit(2)
		case PanicOnError:
			panic(err)
		}
	}
	return nil
}

// Parse parses the environment values from [os.Environ]. Must be called
// after all variables are defined and before variables are accessed by the program.
func Parse() {
	// TODO(edoput) ignore errors?
	Environment.Parse(os.Environ())
}

// Environment is the default set of environment values, parsed from [os.GetEnv].
// The top-leve functions such as [BoolVar], [Bool], and so on are wrappers for the
// methods of Environment.
var Environment = NewEnvSet(os.Args[0], ExitOnError)

func NewEnvSet(name string, errorHandling ErrorHandling) *EnvSet {
	e := &EnvSet{
		name:          name,
		errorHandling: errorHandling,
	}
	return e
}

func (e *EnvSet) Init(name string, errorHandling ErrorHandling) {
	e.name = name
	e.errorHandling = errorHandling
}
