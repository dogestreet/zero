package equihash

// #cgo LDFLAGS: -L${SRCDIR}/libs -lequi
// #include "include/equi.h"
import "C"
import (
	"bytes"
	"errors"
	"io"
	"unsafe"
)

type bitReader struct {
	reader io.ByteReader
	b      byte
	offset byte
}

func newBitReader(r io.ByteReader) *bitReader {
	return &bitReader{r, 0, 0}
}

func (r *bitReader) ReadBit() (bool, error) {
	if r.offset == 8 {
		r.offset = 0
	}
	if r.offset == 0 {
		var err error
		if r.b, err = r.reader.ReadByte(); err != nil {
			return false, err
		}
	}
	bit := (r.b & (0x80 >> r.offset)) != 0
	r.offset++
	return bit, nil
}

func (r *bitReader) Read21Bits() (uint32, error) {
	var result uint32

	for i := uint32(0); i < 21; i++ {
		on, err := r.ReadBit()
		if err != nil {
			return 0, err
		}

		if on {
			result |= uint32(1<<20) >> i
		}
	}

	return result, nil
}

// Verify POW provided by a miner.
func Verify(n, k int, headerNonce []byte, solution []byte) (bool, error) {
	if len(headerNonce) != 140 {
		return false, errors.New("bad header nonce (140 bytes required)")
	}

	proofLen := 1 << uint(k)

	// We expect solution to be made up a packed bit string of 21 bits
	if len(solution)*8 != 21*proofLen {
		return false, errors.New("bad solution")
	}

	// Extract each index in the proof
	proof := make([]uint32, proofLen)
	reader := newBitReader(bytes.NewBuffer(solution))
	for i := 0; i < proofLen; i++ {
		var err error
		proof[i], err = reader.Read21Bits()
		if err != nil {
			return false, err
		}
	}

	var res C.int
	if n == 200 && k == 9 {
		res = C.verify_200_9((*C.uint32_t)(unsafe.Pointer(&proof[0])), (*C.char)(unsafe.Pointer(&headerNonce[0])), (C.uint32_t)(len(headerNonce)))
	} else if n == 48 && k == 5 {
		res = C.verify_48_5((*C.uint32_t)(unsafe.Pointer(&proof[0])), (*C.char)(unsafe.Pointer(&headerNonce[0])), (C.uint32_t)(len(headerNonce)))
	}
	if res != 0 {
		return false, errors.New("pow failed")
	}

	return true, nil
}
