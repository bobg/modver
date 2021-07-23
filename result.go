package modver

import "fmt"

// Result is the result of Compare.
type Result interface {
	Code() ResultCode
	String() string

	wrap(string) Result
	sub(code ResultCode) Result
}

// ResultCode is the required version-bump level as detected by Compare.
type ResultCode int

const (
	None ResultCode = iota
	Patchlevel
	Minor
	Major
)

func (r ResultCode) Code() ResultCode           { return r }
func (r ResultCode) sub(code ResultCode) Result { return code }
func (r ResultCode) wrap(why string) Result     { return wrapped{r: r, why: why} }

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

func (w wrapped) Code() ResultCode           { return w.r.Code() }
func (w wrapped) sub(code ResultCode) Result { return wrapped{r: w.r.sub(code), why: w.why} }
func (w wrapped) wrap(why string) Result {
	return wrapped{r: w.Code(), why: fmt.Sprintf("%s: %s", why, w.why)}
}

func (w wrapped) String() string {
	return fmt.Sprintf("%s: %s", w.r, w.why)
}
