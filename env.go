// TODO(edoput) HELP=t should print out the usage equivalent of the EnvSet
package env

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// errParse is returned by Set if a flag's value fails to parse, such as with an invalid integer for Int.
// It then gets wrapped through failf to provide more information.
var errParse = errors.New("parse error")

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

// Value is the interface to the dynamic value stored in a Spec.
// (The default value is represented as a string.)
//
// TODO(edoput) is this ok?
// If a Value has an IsBoolVal() bool method returning true,
// the environment parser makes `NAME=` equivalent to `NAME=true`.
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
