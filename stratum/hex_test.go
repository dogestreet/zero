package stratum

import (
	"math/rand"
	"testing"
)

func TestHex(t *testing.T) {
	inversion := func(a, b, c, d uint64, e, f, g, h uint32) {
		if r, _ := HexToInt32(ToHex(int32(a))); r != int32(a) {
			t.Fatal("failed")
		}

		if r, _ := HexToUint32(ToHex(uint32(a))); r != uint32(a) {
			t.Fatal("failed")
		}

		if r, _ := HexToUint64(ToHex(uint64(a))); r != uint64(a) {
			t.Fatal("failed")
		}

		x := ToHex(uint32(a)) + ToHex(uint32(b)) + ToHex(uint32(c)) + ToHex(uint32(d))
		if r, _ := HexToUint128(x); ToHex(r) != x {
			t.Fatal("failed", x, r)
		}

		y := ToHex(uint32(a)) + ToHex(uint32(b)) + ToHex(uint32(c)) + ToHex(uint32(d)) + ToHex(e) + ToHex(f) + ToHex(g) + ToHex(h)
		if r, _ := HexToUint256(y); ToHex(r) != y {
			t.Fatal("failed")
		}
	}

	// Test boundary values
	inversion(0, 0, 0, 0, 0, 0, 0, 0)
	inversion(
		0xffffffffffffffff,
		0xffffffffffffffff,
		0xffffffffffffffff,
		0xffffffffffffffff,
		0xffffffff,
		0xffffffff,
		0xffffffff,
		0xffffffff,
	)

	// Random sample values
	rng := rand.New(rand.NewSource(0))
	for i := 0; i < 1000; i++ {
		a := uint64(rng.Uint32())<<32 + uint64(rng.Uint32())
		b := uint64(rng.Uint32())<<32 + uint64(rng.Uint32())
		c := uint64(rng.Uint32())<<32 + uint64(rng.Uint32())
		d := uint64(rng.Uint32())<<32 + uint64(rng.Uint32())

		e := rng.Uint32()
		f := rng.Uint32()
		g := rng.Uint32()
		h := rng.Uint32()

		inversion(a, b, c, d, e, f, g, h)
	}

	// Test lower half.
	for i := 0; i < 1000; i++ {
		a := uint64(rng.Uint32())
		b := uint64(rng.Uint32())
		c := uint64(rng.Uint32())
		d := uint64(rng.Uint32())

		e := uint32(rng.Intn(0xffff))
		f := uint32(rng.Intn(0xffff))
		g := uint32(rng.Intn(0xffff))
		h := uint32(rng.Intn(0xffff))

		inversion(a, b, c, d, e, f, g, h)
	}

	// Test bad values
	_, err := HexToInt32("zzzz")
	if err == nil {
		t.Fatal("bad parse")
	}

	_, err = HexToInt32("120x000")
	if err == nil {
		t.Fatal("bad parse")
	}

	_, err = HexToInt32("1200000000000000")
	if err == nil {
		t.Fatal("bad parse")
	}
}
