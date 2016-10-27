package main

import (
	"log"
	"math"
	"testing"

	"github.com/dogestreet/zero/stratum"
)

func TestCompact(t *testing.T) {
	table := []struct {
		compact uint32
		diff    string
	}{
		{0x1e0b973a, "00000b973a000000000000000000000000000000000000000000000000000000"},
		{0x1e067349, "0000067349000000000000000000000000000000000000000000000000000000"},
		{0x2003ffff, "03ffff0000000000000000000000000000000000000000000000000000000000"},
		{0x007bcdef, "0000000000000000000000000000000000000000000000000000000000000000"},
		{0x017bcdef, "000000000000000000000000000000000000000000000000000000000000007b"},
		{0x027bcdef, "0000000000000000000000000000000000000000000000000000000000007bcd"},
		{0x037bcdef, "00000000000000000000000000000000000000000000000000000000007bcdef"},
		{0x047bcdef, "000000000000000000000000000000000000000000000000000000007bcdef00"},
	}

	for _, x := range table {
		res, _ := CompactToTarget(x.compact)
		if stratum.ToHex(res) != x.diff {
			log.Println(stratum.ToHex(res), x.diff)
			t.Fatal(x)
		}
	}
}

func TestDifficulty(t *testing.T) {
	table := []struct {
		compact uint32
		diff    Difficulty
	}{
		{0x1e067349, 40640.22966959},
		{0x1e07b77e, 33970.5762567},
		{0x2003ffff, 1},
		{0x2000c49a, 5.20848401},
	}

	for _, x := range table {
		res, _ := CompactToTarget(x.compact)
		cpDiff := FromTarget(res)
		if math.Abs(float64(x.diff-cpDiff)) > 1 {
			log.Println("Failed")
		}
	}
}
