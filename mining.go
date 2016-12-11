package main

import (
	"errors"
	"math/big"
	"time"

	"github.com/dogestreet/zero/stratum"
)

// Difficulty ...
type Difficulty float64

// POWLimit as defined by chainparams.cpp
var POWLimit *big.Int

func init() {
	POWLimit, _ = new(big.Int).SetString("0x03ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0)
}

// ToTarget converts a vardiff difficulty to a Uint256.
func (d Difficulty) ToTarget() stratum.Uint256 {
	var result stratum.Uint256
	if d == 0 {
		copy(result[:], POWLimit.Bytes())
		return result
	}
	diff := big.NewInt(int64(d))

	bytes := new(big.Int).Div(POWLimit, diff).Bytes()

	if len(bytes) > len(result) {
		copy(result[:], POWLimit.Bytes())
		return result
	}

	copy(result[len(result)-len(bytes):], bytes)

	return result
}

// FromTarget converts a target difficulty to a difficulty value.
func FromTarget(target stratum.Uint256) Difficulty {
	targ := target.ToInteger()
	if targ.Cmp(new(big.Int)) == 0 {
		// Don't return infinity, it'll just mess everything up
		return 10000000000000000000
	}

	res := new(big.Int).Div(POWLimit, targ)
	return Difficulty(res.Uint64())
}

// TargetCompare targets.
func TargetCompare(a, b stratum.Uint256) int {
	return a.ToInteger().Cmp(b.ToInteger())
}

// CompactToTarget converts a NDiff value to a Target.
func CompactToTarget(x uint32) (stratum.Uint256, error) {
	x = reverseUint32(x)

	// The top byte is the number of bytes in the final string
	var result stratum.Uint256
	i := (x & 0xff000000) >> 24

	if int(i) > len(result) {
		return result, errors.New("invalid compact target")
	}

	parts := []byte{byte((x & 0x7f0000) >> 16), byte((x & 0xff00) >> 8), byte(x & 0xff)}
	copy(result[len(result)-int(i):], parts)

	return result, nil
}

const c = 1 / 32

// EstimateHashPerSecond uses the shares returned by the miner to estimate mining speed.
// We do this by the following deduction:
// A miner generates `x` trials per second, and each trial has a probability p = (`diff`.ToTarget()) / 2^256 of finding a share
// We observed `n` shares in the last `elapsed` seconds
// The expected number of trials required to find a share is defined by a geometric distribution
// Thus E[number of trials before finding a share] = 1/p
// Therefore we expect that the miner would have completed 1/p * `n` trials in `elapsed` seconds
//
// We note that:
//     p  = diff.ToTarget() / 2^256 = (POWLimit / diff) / 2^256 = (POWLimit / 2^256) / diff
// Thus:
//     x = ((diff / 0.0156249) * n) / elapsed
func EstimateHashPerSecond(n int, diff Difficulty, elapsed time.Duration) float64 {
	return (float64(diff) / c) * float64(n) / elapsed.Seconds()
}

// FindDifficulty returns the optimal difficulty by fixing x in the equation above.
func FindDifficulty(x float64, n int, elapsed int) Difficulty {
	return Difficulty(((x * float64(elapsed)) / float64(n)) * c)
}
