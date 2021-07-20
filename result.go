package modver

import "fmt"

// Result is the result of Compare,
// consisting of a ResultCode and an optional human-readable explanation.
type Result struct {
	Code ResultCode
	Why  string
}

func (r Result) String() string {
	if r.Why != "" {
		return fmt.Sprintf("%s: %s", r.Code, r.Why)
	}
	return r.Code.String()
}

// ResultCode is the required version-bump level as detected by Compare.
type ResultCode int

const (
	None ResultCode = iota
	Patchlevel
	Minor
	Major
)

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
