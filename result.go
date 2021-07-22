package modver

import "fmt"

// Result is the result of Compare.
type Result interface {
	Code() ResultCode
	Sub(code ResultCode) Result
	String() string
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
func (r ResultCode) Sub(code ResultCode) Result { return code }

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
func (w wrapped) Sub(code ResultCode) Result { return wrapped{r: w.r.Sub(code), why: w.why} }

func (w wrapped) String() string {
	return fmt.Sprintf("%s: %s", w.why, w.r)
}
