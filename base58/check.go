package base58

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/big"
)

var decoder = map[byte]int64{
	'1': 0,
	'2': 1,
	'3': 2,
	'4': 3,
	'5': 4,
	'6': 5,
	'7': 6,
	'8': 7,
	'9': 8,
	'A': 9,
	'B': 10,
	'C': 11,
	'D': 12,
	'E': 13,
	'F': 14,
	'G': 15,
	'H': 16,
	'J': 17,
	'K': 18,
	'L': 19,
	'M': 20,
	'N': 21,
	'P': 22,
	'Q': 23,
	'R': 24,
	'S': 25,
	'T': 26,
	'U': 27,
	'V': 28,
	'W': 29,
	'X': 30,
	'Y': 31,
	'Z': 32,
	'a': 33,
	'b': 34,
	'c': 35,
	'd': 36,
	'e': 37,
	'f': 38,
	'g': 39,
	'h': 40,
	'i': 41,
	'j': 42,
	'k': 43,
	'm': 44,
	'n': 45,
	'o': 46,
	'p': 47,
	'q': 48,
	'r': 49,
	's': 50,
	't': 51,
	'u': 52,
	'v': 53,
	'w': 54,
	'x': 55,
	'y': 56,
	'z': 57,
}

// DecodeCheck decodes a base58check encoded address.
func DecodeCheck(str string) ([]byte, error) {
	leading := true
	var leadingZeros int

	// Run the base58 decode
	x := big.NewInt(0)
	base := big.NewInt(58)
	for _, v := range []byte(str) {
		y, ok := decoder[v]
		if !ok {
			return nil, errors.New("base58: invalid character detected")
		}

		if y == 0 {
			if leading {
				leadingZeros++
			}
		} else {
			leading = false
		}

		x.Mul(x, base)
		x.Add(x, big.NewInt(y))
	}

	var data []byte
	for i := 0; i < leadingZeros; i++ {
		data = append(data, 0)
	}

	data = append(data, x.Bytes()...)
	if len(data) < 4 {
		return nil, errors.New("base58: too short")
	}

	raw := data[:len(data)-4]
	checksum := data[len(data)-4:]

	result1 := sha256.Sum256(raw)
	result2 := sha256.Sum256(result1[:])

	if !bytes.Equal(checksum, result2[:4]) {
		return nil, errors.New("base58: base checksum")
	}

	return raw, nil
}
