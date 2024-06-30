package modver

import (
	"fmt"
	"io"
	"strings"
)

// Result is the result of Compare.
type Result interface {
	Code() ResultCode
	String() string

	sub(code ResultCode) Result
}

// ResultCode is the required version-bump level as detected by Compare.
type ResultCode int

// Values for ResultCode.
const (
	None ResultCode = iota
	Patchlevel
	Minor
	Major
)

// Code implements Result.Code.
func (r ResultCode) Code() ResultCode           { return r }
func (r ResultCode) sub(code ResultCode) Result { return code }

// String implements Result.String.
func (r ResultCode) String() string {
	switch r {
	case None:
		return "None"
	case Patchlevel:
		return "Patchlevel"
	case Minor:
		return "Minor"
	case Major:
		return "Major"
	default:
		return "unknown Result value"
	}
}

func (r ResultCode) MarshalText() ([]byte, error) {
	switch r {
	case None, Patchlevel, Minor, Major:
		return []byte(r.String()), nil
	}
	return nil, fmt.Errorf("unknown ResultCode value %d", r)
}

func (r *ResultCode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "None":
		*r = None
	case "Patchlevel":
		*r = Patchlevel
	case "Minor":
		*r = Minor
	case "Major":
		*r = Major
	default:
		return fmt.Errorf("unknown ResultCode value %q", text)
	}
	return nil
}

type wrapped struct {
	r       Result
	whyfmt  string
	whyargs []any
}

// Code implements Result.Code.
func (w wrapped) Code() ResultCode { return w.r.Code() }
func (w wrapped) sub(code ResultCode) Result {
	result := w
	result.r = w.r.sub(code)
	return result
}

func (w wrapped) why() string {
	return fmt.Sprintf(w.whyfmt, w.whyargs...)
}

// String implements Result.String.
func (w wrapped) String() string {
	return fmt.Sprintf("%s: %s", w.r, w.why())
}

func (w wrapped) pretty(out io.Writer, level int) {
	fmt.Fprintf(out, "%s%s\n", strings.Repeat("  ", level), w.why())
	prettyLevel(out, w.r, level+1)
}

func rwrap(r Result, s string) Result {
	return rwrapf(r, "%s", s)
}

func rwrapf(r Result, format string, args ...any) Result {
	if r.Code() == None {
		return r
	}
	return wrapped{r: r, whyfmt: format, whyargs: args}
}

type prettyer interface {
	pretty(io.Writer, int)
}

// Pretty writes a pretty representation of res to out.
func Pretty(out io.Writer, res Result) {
	prettyLevel(out, res, 0)
}

func prettyLevel(out io.Writer, res Result, level int) {
	if p, ok := res.(prettyer); ok {
		p.pretty(out, level)
	} else {
		fmt.Fprintf(out, "%s%s\n", strings.Repeat("  ", level), res)
	}
}
