// Package basexx permits converting between digit strings of arbitrary bases.
package basexx

import (
	"errors"
	"io"
	"math"
	"math/big"
)

// Source is a source of digit values in a given base.
type Source interface {
	// Read produces the value of the next-least-significant digit in the source.
	// The value must be between 0 and Base()-1, inclusive.
	// End of input is signaled with the error io.EOF.
	Read() (int64, error)

	// Base gives the base of the Source.
	// Digit values in the Source must all be between 0 and Base()-1, inclusive.
	// Behavior is undefined if the value of Base() varies during the lifetime of a Source
	// or if Base() < 2.
	Base() int64
}

// Dest is a destination for writing digits in a given base.
// Digits are written right-to-left, from least significant to most.
type Dest interface {
	// Prepend encodes the next-most-significant digit value and prepends it to the destination.
	Prepend(int64) error

	// Base gives the base of the Dest.
	// Digit values in the Dest must all be between 0 and Base()-1, inclusive.
	// Behavior is undefined if the value of Base() varies during the lifetime of a Dest
	// or if Base() < 2.
	Base() int64
}

// Base is the type of a base.
type Base interface {
	// N is the number of the base,
	// i.e. the number of unique digits.
	// Behavior is undefined if the value of N() varies during the lifetime of a Base
	// or if N() < 2.
	N() int64

	// Encode converts a digit value to the string of bytes representing its digit.
	// The input must be a valid digit value between 0 and N()-1, inclusive.
	Encode(int64) ([]byte, error)

	// Decode converts a string of bytes representing a digit into its numeric value.
	Decode([]byte) (int64, error)
}

// ErrInvalid is used for invalid input to Base.Encode and Base.Decode.
var ErrInvalid = errors.New("invalid")

var zero = new(big.Int)

// Convert converts the digits of src, writing them to dest.
// Both src and dest specify their bases.
// Return value is the number of digits written to dest (even in case of error).
// This function consumes all of src before producing any of dest,
// so it may not be suitable for input streams of arbitrary length.
func Convert(dest Dest, src Source) (int, error) {
	var (
		accum    = new(big.Int)
		srcBase  = big.NewInt(src.Base())
		destBase = big.NewInt(dest.Base())
	)
	for {
		digit, err := src.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		accum.Mul(accum, srcBase)
		if digit != 0 {
			accum.Add(accum, big.NewInt(digit))
		}
	}
	var written int
	for accum.Cmp(zero) > 0 {
		r := new(big.Int)
		accum.QuoRem(accum, destBase, r)
		err := dest.Prepend(r.Int64())
		if err != nil {
			return written, err
		}
		written++
	}
	if written == 0 {
		err := dest.Prepend(0)
		if err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// Length computes the maximum number of digits needed
// to convert `n` digits in base `from` to base `to`.
func Length(from, to int64, n int) int {
	ratio := math.Log(float64(from)) / math.Log(float64(to))
	result := float64(n) * ratio
	return int(math.Ceil(result))
}
