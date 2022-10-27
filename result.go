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

// String implements Result.String.
func (w wrapped) String() string {
	return fmt.Sprintf("%s: %s", w.r, fmt.Sprintf(w.whyfmt, w.whyargs...))
}

func (w wrapped) pretty(out io.Writer, level int) {
	fmt.Fprintf(out, "%s%s\n\n", strings.Repeat("  ", level), fmt.Sprintf(w.whyfmt, w.whyargs))
	Pretty(out, w.r, level+1)
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

func Pretty(out io.Writer, res Result, level int) {
	if p, ok := res.(prettyer); ok {
		p.pretty(out, level)
	} else {
		fmt.Fprintf(out, "%s%s\n", strings.Repeat("  ", level), res)
	}
}
