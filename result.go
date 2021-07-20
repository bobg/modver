package modver

type Result int

const (
	None Result = iota
	Patchlevel
	Minor
	Major
)

func (r Result) String() string {
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
