package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/dogestreet/zero/equihash"
	"github.com/dogestreet/zero/stratum"
)

// BuildBlockHeader constructs a block header.
func BuildBlockHeader(version uint32, hashPrevBlock, hashMerkleRoot, hashReserved []byte, nTime, nBits uint32, noncePart1, noncePart2 []byte) *bytes.Buffer {
	buffer := bytes.NewBuffer(nil)
	_ = binary.Write(buffer, binary.BigEndian, version)
	_, _ = buffer.Write(hashPrevBlock)
	_, _ = buffer.Write(hashMerkleRoot)
	_, _ = buffer.Write(hashReserved)
	_ = binary.Write(buffer, binary.BigEndian, nTime)
	_ = binary.Write(buffer, binary.BigEndian, nBits)
	_, _ = buffer.Write(noncePart1)
	_, _ = buffer.Write(noncePart2)

	return buffer
}

// Validate checks POW validity of a header.
func Validate(n, k int, headerNonce []byte, solution []byte, shareTarget, globalTarget stratum.Uint256) ShareStatus {
	ok, err := equihash.Verify(n, k, headerNonce, solution)
	if err != nil {
		return ShareInvalid
	}

	if !ok {
		return ShareInvalid
	}

	// Double sha to check the target
	hash := sha256.New()
	_, _ = hash.Write(headerNonce)
	_, _ = hash.Write([]byte{0xfd, 0x40, 0x05})
	_, _ = hash.Write(solution)

	round1 := hash.Sum(nil)
	round2 := sha256.Sum256(round1[:])

	// Reverse the hash
	for i, j := 0, len(round2)-1; i < j; i, j = i+1, j-1 {
		round2[i], round2[j] = round2[j], round2[i]
	}

	// Check against the global target
	if TargetCompare(round2, globalTarget) <= 0 {
		return ShareBlock
	}

	if TargetCompare(round2, shareTarget) > 1 {
		return ShareInvalid
	}

	return ShareOK
}
