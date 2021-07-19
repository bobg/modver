// Package subcmd provides types and functions for creating command-line interfaces with subcommands and flags.
package subcmd

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/pkg/errors"
)

var errType = reflect.TypeOf((*error)(nil)).Elem()

// Cmd is the way a command tells Run how to parse and run its subcommands.
type Cmd interface {
	// Subcmds returns this Cmd's subcommands as a map,
	// whose keys are subcommand names and values are Subcmd objects.
	// Implementations may use the Commands function to build this map.
	Subcmds() Map
}

// Map is the type of the data structure returned by Cmd.Subcmds and by Commands.
// It maps a subcommand name to its Subcmd structure.
type Map = map[string]Subcmd

// Subcmd is one subcommand of a Cmd.
type Subcmd struct {
	// F is the function implementing the subcommand.
	// Its signature must be func(context.Context, ..., []string) error,
	// where the number and types of parameters between the context and the string slice
	// is given by Params.
	F interface{}

	// Params describes the parameters to F
	// (excluding the initial context.Context that F takes, and the final []string).
	Params []Param
}

// Param is one parameter of a Subcmd.
type Param struct {
	// Name is the flag name for the parameter (e.g., "verbose" for a -verbose flag).
	Name string

	// Type is the type of the parameter.
	Type Type

	// Default is a default value for the parameter.
	// Its type must be suitable for Type.
	Default interface{}

	// Doc is a docstring for the parameter.
	Doc string
}

// Commands is a convenience function for producing the map needed by a Cmd.
// It takes 3n arguments,
// where n is the number of subcommands.
// Each group of three is:
// - the subcommand name, a string;
// - the function implementing the subcommand;
// - the list of parameters for the function, a slice of Param (which can be produced with Params).
//
// A call like this:
//
//   Commands(
//     "foo", foo, Params(
//       "verbose", Bool, false, "be verbose",
//     ),
//     "bar", bar, Params(
//       "level", Int, 0, "barness level",
//     ),
//   )
//
// is equivalent to:
//
//   Map{
//     "foo": Subcmd{
//       F: foo,
//       Params: []Param{
//         {
//           Name: "verbose",
//           Type: Bool,
//           Default: false,
//           Doc: "be verbose",
//         },
//       },
//     },
//     "bar": Subcmd{
//       F: bar,
//       Params: []Param{
//         {
//           Name: "level",
//           Type: Int,
//           Default: 0,
//           Doc: "barness level",
//         },
//       },
//     },
//  }
//
// This function panics if the number or types of the arguments are wrong.
func Commands(args ...interface{}) Map {
	if len(args)%3 != 0 {
		panic(fmt.Sprintf("S has %d arguments, which is not divisible by 3", len(args)))
	}

	result := make(Map)

	for len(args) > 0 {
		var (
			name = args[0].(string)
			f    = args[1]
			p    = args[2]
		)
		subcmd := Subcmd{F: f}
		if p != nil {
			subcmd.Params = p.([]Param)
		}
		result[name] = subcmd

		args = args[3:]
	}

	return result
}

// Params is a convenience function for producing the list of parameters needed by a Subcmd.
// It takes 4n arguments,
// where n is the number of parameters.
// Each group of four is:
// - the flag name for the parameter, a string (e.g. "verbose" for a -verbose flag);
// - the type of the parameter, a Type constant;
// - the default value of the parameter,
// - the doc string for the parameter.
//
// This function panics if the number or types of the arguments are wrong.
func Params(a ...interface{}) []Param {
	if len(a)%4 != 0 {
		panic(fmt.Sprintf("Params has %d arguments, which is not divisible by 4", len(a)))
	}
	var result []Param
	for len(a) > 0 {
		var (
			name = a[0].(string)
			typ  = a[1].(Type)
			dflt = a[2]
			doc  = a[3].(string)
		)
		result = append(result, Param{Name: name, Type: typ, Default: dflt, Doc: doc})
		a = a[4:]
	}
	return result
}

var (
	// ErrNoArgs is the error when Run is called with an empty list of args.
	ErrNoArgs = errors.New("no arguments")

	// ErrUnknown is the error when Run is called with an unknown subcommand as args[0].
	ErrUnknown = errors.New("unknown subcommand")
)

// Run runs the subcommand of c named in args[0].
// The remaining args are parsed with a new flag.FlagSet,
// populated according to the parameters of the named Subcmd.
// The Subcmd's function is invoked with a context object,
// the parameter values parsed by the FlagSet,
// and a slice of the args left over after FlagSet parsing.
// The FlagSet is placed in the context object that's passed to the Subcmd's function,
// and can be retrieved if needed with the FlagSet function.
func Run(ctx context.Context, c Cmd, args []string) error {
	cmds := c.Subcmds()

	var cmdnames sort.StringSlice
	for cmdname := range cmds {
		cmdnames = append(cmdnames, cmdname)
	}
	cmdnames.Sort()

	if len(args) == 0 {
		return errors.Wrapf(ErrNoArgs, "possible subcommands: %v", cmdnames)
	}

	name := args[0]
	args = args[1:]
	subcmd, ok := cmds[name]
	if !ok {
		return errors.Wrapf(ErrUnknown, "got %s, want one of %v", name, cmdnames)
	}

	var ptrs []reflect.Value

	if len(subcmd.Params) > 0 {
		fs := flag.NewFlagSet("", flag.ContinueOnError)
		ctx = context.WithValue(ctx, fskey, fs)

		for _, p := range subcmd.Params {
			var v interface{}

			switch p.Type {
			case Bool:
				dflt, _ := p.Default.(bool)
				v = fs.Bool(p.Name, dflt, p.Doc)

			case Int:
				dflt, _ := p.Default.(int)
				v = fs.Int(p.Name, dflt, p.Doc)

			case Int64:
				dflt, _ := p.Default.(int64)
				v = fs.Int64(p.Name, dflt, p.Doc)

			case Uint:
				dflt, _ := p.Default.(uint)
				v = fs.Uint(p.Name, dflt, p.Doc)

			case Uint64:
				dflt, _ := p.Default.(uint64)
				v = fs.Uint64(p.Name, dflt, p.Doc)

			case String:
				dflt, _ := p.Default.(string)
				v = fs.String(p.Name, dflt, p.Doc)

			case Float64:
				dflt, _ := p.Default.(float64)
				v = fs.Float64(p.Name, dflt, p.Doc)

			case Duration:
				dflt, _ := p.Default.(time.Duration)
				v = fs.Duration(p.Name, dflt, p.Doc)

			default:
				return fmt.Errorf("unknown arg type %v", p.Type)
			}

			ptrs = append(ptrs, reflect.ValueOf(v))
		}

		err := fs.Parse(args)
		if err != nil {
			return errors.Wrap(err, "parsing args")
		}

		args = fs.Args()
	}

	argvals := []reflect.Value{reflect.ValueOf(ctx)}
	for _, ptr := range ptrs {
		argvals = append(argvals, ptr.Elem())
	}
	argvals = append(argvals, reflect.ValueOf(args))

	fv := reflect.ValueOf(subcmd.F)
	ft := fv.Type()
	if ft.Kind() != reflect.Func {
		return fmt.Errorf("implementation for subcommand %s is a %s, want a function", name, ft.Kind())
	}
	if numIn := ft.NumIn(); numIn != len(argvals) {
		return fmt.Errorf("function for subcommand %s takes %d arg(s), want %d", name, numIn, len(argvals))
	}
	for i, argval := range argvals {
		if !argval.Type().AssignableTo(ft.In(i)) {
			return fmt.Errorf("type of arg %d is %s, want %s", i, ft.In(i), argval.Type())
		}
	}

	if numOut := ft.NumOut(); numOut != 1 {
		return fmt.Errorf("function for subcommand %s returns %d args, want 1", name, numOut)
	}
	if !ft.Out(0).Implements(errType) {
		return fmt.Errorf("return type is not error")
	}

	rv := fv.Call(argvals)
	err, _ := rv[0].Interface().(error)
	return errors.Wrapf(err, "running %s", name)
}

// Type is the type of a Param.
type Type int

// Possible Param types.
// These correspond with the types in the standard flag package.
const (
	Bool Type = iota + 1
	Int
	Int64
	Uint
	Uint64
	String
	Float64
	Duration
)

type fskeytype struct{}

var fskey fskeytype

// FlagSet produces the *flag.FlagSet used in a call to a Subcmd function.
func FlagSet(ctx context.Context) *flag.FlagSet {
	val := ctx.Value(fskey)
	return val.(*flag.FlagSet)
}
