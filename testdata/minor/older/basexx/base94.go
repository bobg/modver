package basexx

type base94 struct{}

func (b base94) N() int64 { return 94 }

func (b base94) Encode(val int64) ([]byte, error) {
	if val < 0 || val > 93 {
		return nil, ErrInvalid
	}
	return []byte{byte(val + 33)}, nil
}

func (b base94) Decode(inp []byte) (int64, error) {
	if len(inp) != 1 {
		return 0, ErrInvalid
	}
	digit := inp[0]
	if digit < 33 || digit > 126 {
		return 0, ErrInvalid
	}
	return int64(digit - 33), nil
}

// Base94 uses all printable ASCII characters (33 through 126) as digits.
var Base94 base94
