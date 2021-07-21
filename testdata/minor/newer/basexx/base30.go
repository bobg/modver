package basexx

const base30digits = "0123456789bcdfghjkmnpqrstvwxyz"

var base30digitVals [256]int64

type base30 struct{}

func (b base30) N() int64 { return 30 }

func (b base30) Encode(val int64) ([]byte, error) {
	if val < 0 || val > 49 {
		return nil, ErrInvalid
	}
	return []byte{byte(base30digits[val])}, nil
}

func (b base30) Decode(inp []byte) (int64, error) {
	if len(inp) != 1 {
		return 0, ErrInvalid
	}
	val := base30digitVals[inp[0]]
	if val < 0 {
		return 0, ErrInvalid
	}
	return val, nil
}

// Base30 uses digits 0-9, then lower-case bcdfghjkmnpqrstvwxyz.
// It excludes vowels (to avoid inadvertently spelling naughty words) and the letter "l".
var Base30 base30

func init() {
	for i := 0; i < 256; i++ {
		base30digitVals[i] = -1
	}
	for i := 0; i < len(base30digits); i++ {
		base30digitVals[base30digits[i]] = int64(i)
	}
}
