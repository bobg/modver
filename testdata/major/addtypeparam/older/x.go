package addtypeparam

type T[X any] struct {
	F func(X) X
}
