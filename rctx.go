package modver

type ReportContext interface{}

type ArrayElem struct {
	older, newer types.Type
}

type ChanElem struct {
	older, newer types.Type
}

type PointerElem struct {
	older, newer types.Type
}

type MapKey struct {
	older, newer types.Type
}

type MapElem struct {
	older, newer types.Type
}

type SliceElem struct {
	older, newer types.Type
}
