package modver

type Finding interface {
	Level() Level
	String() string
}

type NoNewPkg struct {
	PkgPath string
}

func (n NoNewPkg) Level() Level { return LevelMajor }
func (n NoNewPkg) String() string {
	return fmt.Sprintf("no new version of package %s", n.PkgPath)
}

type NoNewObj struct {
	PkgPath, ID string
}

func (n NoNewObj) Level() Level { return LevelMajor }
func (n NoNewObj) String() string {
	return fmt.Sprintf("no object %s in new version of package %s", n.ID, n.PkgPath)
}

type NoOldPkg struct {
	PkgPath string
}

func (n NoOldPkg) Level() Level { return LevelMinor }
func (n NoOldPkg) String() string {
	return fmt.Sprintf("no old version of package %s", n.PkgPath)
}

type NoOldObj struct {
	PkgPath, ID string
}

func (n NoOldObj) Level() Level { return LevelMinor }
func (n NoOldObj) String() string {
	return fmt.Sprintf("no object %s in old version of package %s", n.ID, n.PkgPath)
}

type Level int

const (
	LevelNone Level = iota
	LevelPatchlevel
	LevelMinor
	LevelMajor
)

