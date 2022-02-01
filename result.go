package modver

import "fmt"

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
	r   Result
	why string
}

// Code implements Result.Code.
func (w wrapped) Code() ResultCode           { return w.r.Code() }
func (w wrapped) sub(code ResultCode) Result { return wrapped{r: w.r.sub(code), why: w.why} }

// String implements Result.String.
func (w wrapped) String() string {
	return fmt.Sprintf("%s: %s", w.r, w.why)
}

func rwrap(r Result, s string) Result {
	return rwrapf(r, "%s", s)
}

func rwrapf(r Result, format string, args ...any) Result {
	if r.Code() == None {
		return r
	}
	return wrapped{r: r, why: fmt.Sprintf(format, args...)}
}
