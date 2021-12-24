package basexx

// Alnum is a type for bases from 2 through 36,
// where the digits for the first 10 digit values are '0' through '9'
// and the remaining digits are 'a' through 'z'.
// For decoding, upper-case 'A' through 'Z' are the same as lower-case.
type Alnum int

func (a Alnum) N() int64 { return int64(a) }

func (a Alnum) Encode(val int64) ([]byte, error) {
	if val < 0 || val >= int64(a) {
		return nil, ErrInvalid
	}
	if val < 10 {
		return []byte{byte(val) + '0'}, nil
	}
	return []byte{byte(val) + 'a' - 10}, nil
}

func (a Alnum) Decode(inp []byte) (int64, error) {
	if len(inp) != 1 {
		return 0, ErrInvalid
	}
	digit := byte(inp[0])
	switch {
	case '0' <= digit && digit <= '9':
		return int64(digit - '0'), nil
	case 'a' <= digit && digit <= 'z':
		return int64(digit - 'a' + 10), nil
	case 'A' <= digit && digit <= 'Z':
		return int64(digit - 'A' + 10), nil
	default:
		return 0, ErrInvalid
	}
}

const (
	Base2  = Alnum(2)
	Base8  = Alnum(8)
	Base10 = Alnum(10)
	Base12 = Alnum(12)
	Base16 = Alnum(16)
	Base32 = Alnum(32)
	Base36 = Alnum(36)
)
