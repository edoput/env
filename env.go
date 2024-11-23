// Copyright 2024, Edoardo Putti
// Portions Copyright (c) 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO(edoput): top-level Usage function missing
// TODO(edoput): boolValue: IsBoolVar can be removed?
package env

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ErrHelp is the error returned if the HELP or H environment variable is set
// but no such variable is defined.
var ErrHelp = errors.New("env: help requested")

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
type textValue struct{ p encoding.TextUnmarshaler }

func newTextValue(val encoding.TextUnmarshaler, p encoding.TextUnmarshaler) textValue {
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
type funcValue func(string) error

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

// A Spec represents the state of an environment variable.
type Spec struct {
	Name        string // name as it appears in environment
	Description string // short description
	Value       Value  // value as set
	DefValue    string // default value (as text); for description message
}

// sortVariables returns the variables as a slice in lexicographical sorted order.
func sortVariables(vars map[string]*Spec) []*Spec {
	result := make([]*Spec, len(vars))
	i := 0
	for _, s := range vars {
		result[i] = s
		i++
	}
	slices.SortFunc(result, func(a, b *Spec) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result
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

// ErrorHandling returns the error handling behavior of the variable set.
func (e *EnvSet) ErrorHandling() ErrorHandling {
	return e.errorHandling
}

// SetOutput sets the destination for usage and error messages.
// If output is nil, [os.Stderr] is used.
func (e *EnvSet) SetOutput(output io.Writer) {
	e.output = output
}

// VisitAll visits the variables in lexicographical order, calling fn for each.
// It visits all, even those not set.
func (e *EnvSet) VisitAll(fn func(*Spec)) {
	for _, spec := range sortVariables(e.formal) {
		fn(spec)
	}
}

// VisitAll visits the variables in lexicographical order, calling
// fn for each. It visits all, even those not set.
func VisitAll(fn func(*Spec)) {
	Environment.VisitAll(fn)
}

// Visit visits the variables in lexicographical order, calling fn for each.
// It visits only those that have been set.
func (e *EnvSet) Visit(fn func(*Spec)) {
	for _, spec := range sortVariables(e.actual) {
		fn(spec)
	}
}

// Visit visits the variables in lexicographical order, calling fn
// for each. It visits only those that have been set.
func Visit(fn func(*Spec)) {
	Environment.Visit(fn)
}

// isZeroValue determines whether the string represents the zero
// value for a variable.
func isZeroValue(spec *Spec, value string) (ok bool, err error) {
	// Build a zero value of the variable's Value type, and see if the
	// result of calling its String method equals the value passed in.
	// This works unless the Value type is itself an interface type.
	typ := reflect.TypeOf(spec.Value)
	var z reflect.Value
	if typ.Kind() == reflect.Pointer {
		z = reflect.New(typ.Elem())
	} else {
		z = reflect.Zero(typ)
	}
	// Catch panics calling the String method, which shouldn't prevent the
	// usage message from being printed, but that we should report to the
	// user so that they know to fix their code.
	defer func() {
		if e := recover(); e != nil {
			if typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}
			err = fmt.Errorf("panic calling String method on zero %v for variable %s: %v", typ, spec.Name, e)
		}
	}()
	return value == z.Interface().(Value).String(), nil
}

// UnquoteUsage extracts a back-quoted name from the usage
// string for an environment variable and returns it and the un-quoted usage.
// Given "a `name` to show" it returns ("name", "a name to show").
// If there are no back quotes, the name is an educated guess of the
// type of the variables's value.
func UnquoteUsage(spec *Spec) (name string, description string) {
	// Look for a back-quoted name, but avoid the strings package.
	description = spec.Description
	for i := 0; i < len(description); i++ {
		if description[i] == '`' {
			for j := i + 1; j < len(description); j++ {
				if description[j] == '`' {
					name = description[i+1 : j]
					description = description[:i] + name + description[j+1:]
					return name, description
				}
			}
			break // Only one back quote; use type name.
		}
	}
	// No explicit name, so use type if we can find one.
	name = "value"
	switch spec.Value.(type) {
	case *boolValue:
		name = "boolean"
	case *durationValue:
		name = "duration"
	case *float64Value:
		name = "float"
	case *intValue, *int64Value:
		name = "int"
	case *stringValue:
		name = "string"
	case *uintValue, *uint64Value:
		name = "uint"
	}
	return
}

// PrintDefaults print, to standard error unless configured otherwise, the
// default values of all defined environment variables in the set. See the
// documentation for the global function PrintDefaults for more information.
func (e *EnvSet) PrintDefaults() {
	var isZeroValueErrs []error
	e.VisitAll(func(spec *Spec) {
		var b strings.Builder
		fmt.Fprintf(&b, "  %s", spec.Name)
		name, usage := UnquoteUsage(spec)
		if len(name) > 0 {
			b.WriteString("  ")
			b.WriteString(name)
		}
		b.WriteString("\n    \t")
		b.WriteString(strings.ReplaceAll(usage, "\n", "\n    \t"))
		// Print the default value only if it differs to the zero value
		// for this variable type.
		if isZero, err := isZeroValue(spec, spec.DefValue); err != nil {
			isZeroValueErrs = append(isZeroValueErrs, err)
		} else if !isZero {
			if _, ok := spec.Value.(*stringValue); ok {
				// put quotes on the value
				fmt.Fprintf(&b, " (default %q)", spec.DefValue)
			} else {
				fmt.Fprintf(&b, " (default %v)", spec.DefValue)
			}
		}
		fmt.Fprint(e.Output(), b.String(), "\n")
	})
	// if calling string on any zero env.values triggered a panic, print
	// the messages after the full set of defaults so that the programmer
	// knows to fix the panic.
	if errs := isZeroValueErrs; len(errs) > 0 {
		fmt.Fprintln(e.Output())
		for _, err := range errs {
			fmt.Fprintln(e.Output(), err)
		}
	}
}

// PrintDefaults print, to standard error, unless configured otherwise, the
// default values of all defined environment variables.
// For an integer valued variable x, the default output has the form
//
// x  int
//
//	description-message-for-x (default: 7)
//
// The description message will appear on a separate line.
// The parenthetical default is omitted if the default is the zero value for
// the type. The listed type, here int, can be changed by placing a back-quoted
// name in the variable's description string; the first such item in the message is taken
// to be a parameter name to show in the message and the back quotes are
// stripped from the message when displayed. For instance, given
//
//	env.String("I", "", "search `directory` for include files")
//
// the output will be
//
// I  directory
//
//	search directory for include files.
//
// To change the destination for variable messages, call [Environment].SetOutput.
func PrintDefaults() {
	Environment.PrintDefaults()
}

// defaultEnvironment is the default function to print a usage message.
func (e *EnvSet) defaultEnvironment() {
	if e.name == "" {
		fmt.Fprintf(e.Output(), "Environment:\n")
	} else {
		fmt.Fprintf(e.Output(), "Environment of %s:\n", e.name)
	}
	e.PrintDefaults()
}

// BoolVar defines a bool environment variable with specified name, default value, and description string.
// The argument p points to a bool variable in which to store the value of the
// environment variable.
func (e *EnvSet) BoolVar(p *bool, name string, value bool, description string) {
	e.Var(newBoolValue(value, p), name, description)
}

// BoolVar defines a bool environment variable with specified name, default value, and description string.
// The argument p points to a bool variable in which to store the value of
// the environment variable.
func BoolVar(p *bool, name string, value bool, description string) {
	Environment.Var(newBoolValue(value, p), name, description)
}

// Bool defines a bool environment variable with specified name, default value, and description string.
// The return value is the address of a bool variable that stores the value of
// the environment variable.
func (e *EnvSet) Bool(name string, value bool, description string) *bool {
	p := new(bool)
	e.Var(newBoolValue(value, p), name, description)
	return p
}

// Bool defines a bool environment variable with specified name, default value, and description string.
// The return value is the address of a bool variable that stores the value of the environment variable.
func Bool(name string, value bool, description string) *bool {
	return Environment.Bool(name, value, description)
}

// IntVar defines an int environment variable with specified name, default value, and description string.
// The argument p points to an int variable in which to store the value of the environment variable.
func (e *EnvSet) IntVar(p *int, name string, value int, description string) {
	e.Var(newIntValue(value, p), name, description)
}

// IntVar defines an int environment variable with specified name, default value, and description string.
// The argument p points to an int variable in which to store the value of the variable.
func IntVar(p *int, name string, value int, description string) {
	Environment.Var(newIntValue(value, p), name, description)
}

// Int defines an int environment variable with specified name, default value, and description string.
// The return value is the address of an int variable that stores the value of the variable.
func (e *EnvSet) Int(name string, value int, description string) *int {
	p := new(int)
	e.Var(newIntValue(value, p), name, description)
	return p
}

// Int defines an int environment variable with specified name, default value, and description string.
// The return value is the address of an int variable that stores the value of the variable.
func Int(name string, value int, description string) *int {
	return Environment.Int(name, value, description)
}

// Int64Var defines an int64 environment variable with specified name, default value, and description string.
// The argument p points to an int64 variable in which to store the value of the variable.
func (e *EnvSet) Int64Var(p *int64, name string, value int64, description string) {
	e.Var(newInt64Value(value, p), name, description)
}

// Int64Var defines an int64 environment variable with specified name, default value, and description string.
// The argument p points to an int64 variable in which to store the value of the variable.
func Int64Var(p *int64, name string, value int64, description string) {
	Environment.Var(newInt64Value(value, p), name, description)
}

// Int64 defines an int64 environment variable with specified name, default value, and description string.
// The return value is the address of an int64 variable that stores the value of the variable.
func (e *EnvSet) Int64(name string, value int64, description string) *int64 {
	p := new(int64)
	e.Var(newInt64Value(value, p), name, description)
	return p
}

// Int64 defines an int64 environment variable with specified name, default value, and description string.
// The return value is the address of an int64 variable that stores the value of the variable.
func Int64(name string, value int64, description string) *int64 {
	return Environment.Int64(name, value, description)
}

// UintVar defines a uint environment variable with specified name, default value, and description string.
// The argument p points to a uint variable in which to store the value of the variable.
func (e *EnvSet) UintVar(p *uint, name string, value uint, description string) {
	e.Var(newUintValue(value, p), name, description)
}

// UintVar defines a uint environment variable with specified name, default value, and description string.
// The argument p points to a uint variable in which to store the value of the variable.
func UintVar(p *uint, name string, value uint, description string) {
	Environment.Var(newUintValue(value, p), name, description)
}

// Uint defines a uint environment variable with specified name, default value, and description string.
// The return value is the address of a uint variable that stores the value of the variable.
func (e *EnvSet) Uint(name string, value uint, description string) *uint {
	p := new(uint)
	e.Var(newUintValue(value, p), name, description)
	return p
}

// Uint defines a uint environment variable with specified name, default value, and description string.
// The return value is the address of a uint variable that stores the value of the variable.
func Uint(name string, value uint, description string) *uint {
	return Environment.Uint(name, value, description)
}

// Uint64Var defines a uint64 environment variable with specified name, default value, and description string.
// The argument p points to a uint64 variable in which to store the value of the variable.
func (e *EnvSet) Uint64Var(p *uint64, name string, value uint64, description string) {
	e.Var(newUint64Value(value, p), name, description)
}

// Uint64Var defines a uint64 environment variable with specified name, default value, and description string.
// The argument p points to a uint64 variable in which to store the value of the variable.
func Uint64Var(p *uint64, name string, value uint64, description string) {
	Environment.Var(newUint64Value(value, p), name, description)
}

// Uint64 defines a uint64 environment variable with specified name, default value, and description string.
// The return value is the address of a uint64 variable that stores the value of the variable.
func (e *EnvSet) Uint64(name string, value uint64, description string) *uint64 {
	p := new(uint64)
	e.Var(newUint64Value(value, p), name, description)
	return p
}

// Uint64 defines a uint64 environment variable with specified name, default value, and description string.
// The return value is the address of a uint64 variable that stores the value of the variable.
func Uint64(name string, value uint64, description string) *uint64 {
	return Environment.Uint64(name, value, description)
}

// StringVar defines a string environment variable with specified name, default value, and description string.
// The argument p points to a string variable in which to store the value of the variable.
func (e *EnvSet) StringVar(p *string, name string, value string, description string) {
	e.Var(newStringValue(value, p), name, description)
}

// StringVar defines a string environment variable with specified name, default value, and description string.
// The argument p points to a string variable in which to store the value of the variable.
func StringVar(p *string, name string, value string, description string) {
	Environment.Var(newStringValue(value, p), name, description)
}

// String defines a string environment variable with specified name, default value, and description string.
// The return value is the address of a string variable that stores the value of the variable.
func (e *EnvSet) String(name string, value string, description string) *string {
	p := new(string)
	e.Var(newStringValue(value, p), name, description)
	return p
}

// String defines a string environment variable with specified name, default value, and description string.
// The return value is the address of a string variable that stores the value of the variable.
func String(name string, value string, description string) *string {
	return Environment.String(name, value, description)
}

// Float64Var defines a float64 environment variable with specified name, default value, and description string.
// The argument p points to a float64 variable in which to store the value of the variable.
func (e *EnvSet) Float64Var(p *float64, name string, value float64, description string) {
	e.Var(newFloat64Value(value, p), name, description)
}

// Float64Var defines a float64 environment variable with specified name, default value, and description string.
// The argument p points to a float64 variable in which to store the value of the variable.
func Float64Var(p *float64, name string, value float64, description string) {
	Environment.Var(newFloat64Value(value, p), name, description)
}

// Float64 defines a float64 environment variable with specified name, default value, and description string.
// The return value is the address of a float64 variable that stores the value of the variable.
func (e *EnvSet) Float64(name string, value float64, description string) *float64 {
	p := new(float64)
	e.Var(newFloat64Value(value, p), name, description)
	return p
}

// Float64 defines a float64 environment variable with specified name, default value, and description string.
// The return value is the address of a float64 variable that stores the value of the variable.
func Float64(name string, value float64, description string) *float64 {
	return Environment.Float64(name, value, description)
}

// DurationVar defines a time.Duration environment variable with specified name, default value, and description string.
// The argument p points to a time.Duration variable in which to store the value of the variable.
// The environment variable accepts a value acceptable to time.ParseDuration.
func (e *EnvSet) DurationVar(p *time.Duration, name string, value time.Duration, description string) {
	e.Var(newDurationValue(value, p), name, description)
}

// DurationVar defines a time.Duration environment variable with specified name, default value, and description string.
// The argument p points to a time.Duration variable in which to store the value of the variable.
// The environment variable accepts a value acceptable to time.ParseDuration.
func DurationVar(p *time.Duration, name string, value time.Duration, description string) {
	Environment.Var(newDurationValue(value, p), name, description)
}

// Duration defines a time.Duration environment variable with specified name, default value, and description string.
// The return value is the address of a time.Duration variable that stores the value of the variable.
// The environment variable accepts a value acceptable to time.ParseDuration.
func (e *EnvSet) Duration(name string, value time.Duration, description string) *time.Duration {
	p := new(time.Duration)
	e.Var(newDurationValue(value, p), name, description)
	return p
}

// Duration defines a time.Duration environment variable with specified name, default value, and description string.
// The return value is the address of a time.Duration variable that stores the value of the variable.
// The environment variable accepts a value acceptable to time.ParseDuration.
func Duration(name string, value time.Duration, description string) *time.Duration {
	return Environment.Duration(name, value, description)
}

// TextVar defines a environment variable with a specified name, default value, and description string.
// The argument p must be a pointer to a variable that will hold the value
// of the variable, and p must implement encoding.TextUnmarshaler.
// If the environment variable is used, the environment variable's value will be passed to p's UnmarshalText method.
// The type of the default value must be the same as the type of p.
func (e *EnvSet) TextVar(p encoding.TextUnmarshaler, name string, value encoding.TextUnmarshaler, description string) {
	e.Var(newTextValue(value, p), name, description)
}

// TextVar defines an environment variable with a specified name, default value, and description string.
// The argument p must be a pointer to a variable that will hold the value
// of the variable, and p must implement encoding.TextUnmarshaler.
// If the environment variable is used, the environment variable's value will be passed to p's UnmarshalText method.
// The type of the default value must be the same as the type of p.
func TextVar(p encoding.TextUnmarshaler, name string, value encoding.TextUnmarshaler, description string) {
	Environment.Var(newTextValue(value, p), name, description)
}

// Func defines an environment variable with the specified name and description string.
// Each time the variable name is seen, fn is called with the associated value.
// If fn returns a non-nil error, it will be treated as a parsing error.
func (e *EnvSet) Func(name, description string, fn func(string) error) {
	e.Var(funcValue(fn), name, description)
}

// Func defines an environment variable with the specified name and description string.
// Each time the variable name is seen, fn is called with the associated value.
// If fn returns a non-nil error, it will be treated as a parsing error.
func Func(name, description string, fn func(string) error) {
	Environment.Func(name, description, fn)
}

// BoolFunc defines an environment variable with the specified name and description string without requiring values.
// Each time the variable name is seen, fn is called with the associated value.
// If fn returns a non-nil error, it will be treated as a parsing error.
func (e *EnvSet) BoolFunc(name, description string, fn func(string) error) {
	e.Var(boolFuncValue(fn), name, description)
}

// BoolFunc defines an environment variable with the specified name and description string without requiring values.
// Each time the variable name is seen, fn is called with the associated value.
// If fn returns a non-nil error, it will be treated as a parsing error.
func BoolFunc(name, description string, fn func(string) error) {
	Environment.BoolFunc(name, description, fn)
}

// Var defines an environment variable with the specified name and description string. They type and
// value of the variable are represented by the first argument, of type [Value], which typically holds
// a user-defined implementation of [Value]. For instance, the caller could create an environment
// variable that turns a comma-separated string into a slice of strings by giving the slice the
// methods of [Value]; in particular, [Set] would decompose the comma-separated string into the slice.
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

// Var defines an environment variable with the specified name and description string. The type and
// value of the variable are represented by the first argument, of type [Value], which
// typically holds a user-defined implementation of [Value]. For instance, the
// caller could create an environment variable that turns a comma-separated string into a slice
// of strings by giving the slice the methods of [Value]; in particular, [Set] would
// decompose the comma-separated string into the slice.
func Var(value Value, name string, description string) {
	Environment.Var(value, name, description)
}

// sprintf formats the message, prints it to output, and returns it.
func (e *EnvSet) sprintf(format string, a ...any) string {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(e.Output(), msg)
	return msg
}

// failf prints to standard error a formatted error and usage message and
// returns the error.
func (e *EnvSet) failf(format string, a ...any) error {
	msg := e.sprintf(format, a...)
	e.usage()
	return errors.New(msg)
}

// usage calls the Usage method for the env set if one is specified,
// or the appropriate default usage function otherwise.
func (e *EnvSet) usage() {
	if e.Usage == nil {
		e.defaultEnvironment()
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
	if name == "HELP" || name == "H" {
		e.usage()
		return ErrHelp, false
	}
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
// The return value will be [ErrHelp] if HELP or H were set but not defined.
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
			if err == ErrHelp {
				os.Exit(0)
			}
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
	Environment.Parse(os.Environ())
}

// Environment is the default set of environment values, parsed from [os.GetEnv].
// The top-leve functions such as [BoolVar], [Bool], and so on are wrappers for the
// methods of Environment.
var Environment = NewEnvSet(os.Args[0], ExitOnError)

// NewEnvSet returns a new, empty environment variables set with the specified name and
// error handling property. If the name is not empty, it will be printed
// in the default message and in error messages.
func NewEnvSet(name string, errorHandling ErrorHandling) *EnvSet {
	e := &EnvSet{
		name:          name,
		errorHandling: errorHandling,
	}
	return e
}

// Init sets the name and error handling property for an environment variable set.
// By default, the zero [EnvSet] uses an empty name and the
// [ContinueOnError] error handling policy.
func (e *EnvSet) Init(name string, errorHandling ErrorHandling) {
	e.name = name
	e.errorHandling = errorHandling
}

// Link associates EnvSet e to FlagSet f.
// Error messages when parsing command line flags will also print out
// the description of the environment variables expected.
func Link(f *flag.FlagSet, e *EnvSet) {
	flagSetUsage := f.Usage
	if flagSetUsage == nil {
		flagSetUsage = f.PrintDefaults
	}
	f.Usage = func() {
		flagSetUsage()
		e.usage()
	}
}

func init() {
	// Take over the default error reporting behavior of the flag package.
	// By default the flag package will call the flag.CommandLine.Usage
	// function when an error is encountered while parsing command line flags.

	Link(flag.CommandLine, Environment)
}
