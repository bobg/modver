package addtypeparam

type T[X, Y any] struct {
	F func(X) Y
}
