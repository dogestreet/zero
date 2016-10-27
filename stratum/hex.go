package stratum

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
)

func readHex(s string, n int) ([]byte, error) {
	if len(s) > 2*n {
		return nil, errors.New("value oversized")
	}

	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}

	if len(bytes) != n {
		// Pad with zeros
		buf := make([]byte, n)
		copy(buf[n-len(bytes):], bytes)
		buf = bytes
	}

	return bytes, nil
}

// HexToInt32 ...
func HexToInt32(s string) (int32, error) {
	data, err := readHex(s, 4)
	if err != nil {
		return 0, err
	}

	return int32(binary.BigEndian.Uint32(data)), nil
}

// HexToUint32 ...
func HexToUint32(s string) (uint32, error) {
	data, err := readHex(s, 4)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(data), nil
}

// HexToUint64 ...
func HexToUint64(s string) (uint64, error) {
	data, err := readHex(s, 8)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(data), nil
}

// HexToUint128 ...
func HexToUint128(s string) (Uint128, error) {
	data, err := readHex(s, 16)
	if err != nil {
		return Uint128{}, err
	}

	var res Uint128
	copy(res[:], data)

	return res, nil
}

// HexToUint256 ...
func HexToUint256(s string) (Uint256, error) {
	data, err := readHex(s, 32)
	if err != nil {
		return Uint256{}, err
	}

	var res Uint256
	copy(res[:], data)

	return res, nil
}

// ToHex converts an integer to a hex.
func ToHex(x interface{}) string {
	switch x.(type) {
	case int32:
		return fmt.Sprintf("%08x", uint32(x.(int32)))
	case uint32:
		return fmt.Sprintf("%08x", x)
	case uint64:
		return fmt.Sprintf("%016x", x)
	case Uint128:
		x := x.(Uint128)
		return fmt.Sprintf("%032x", x)
	case Uint256:
		x := x.(Uint256)
		return fmt.Sprintf("%064x", x)
	default:
		log.Println("invalid type passed to ToHex")
		return ""
	}
}
