package basexx

const base50digits = "0123456789bcdfghjkmnpqrstvwxyzBCDFGHJKMNPQRSTVWXYZ"

var base50digitVals [256]int64

type base50 struct{}

func (b base50) N() int64 { return 50 }

func (b base50) Encode(val int64) ([]byte, error) {
	if val < 0 || val > 49 {
		return nil, ErrInvalid
	}
	return []byte{byte(base50digits[val])}, nil
}

func (b base50) Decode(inp []byte) (int64, error) {
	if len(inp) != 1 {
		return 0, ErrInvalid
	}
	val := base50digitVals[inp[0]]
	if val < 0 {
		return 0, ErrInvalid
	}
	return val, nil
}

// Base50 uses digits 0-9, then lower-case bcdfghjkmnpqrstvwxyz, then upper-case BCDFGHJKMNPQRSTVWXYZ.
// It excludes vowels (to avoid inadvertently spelling naughty words) plus lower- and upper-case L.
var Base50 base50

func init() {
	for i := 0; i < 256; i++ {
		base50digitVals[i] = -1
	}
	for i := 0; i < len(base50digits); i++ {
		base50digitVals[base50digits[i]] = int64(i)
	}
}
