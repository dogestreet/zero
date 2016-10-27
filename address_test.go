package main

import (
	"log"
	"testing"
)

func TestAddresses(t *testing.T) {
	table := []struct {
		str     string
		ok      bool
		testnet bool
	}{
		{"t1h8SqgtM3QM5e2M8EzhhT1yL2PXXtA6oqe", true, false},
		{"t1Xxa5ZVPKvs9bGMn7aWTiHjyHvR31XkUst", true, false},
		{"t1ffus9J1vhxvFqLoExGBRPjE7BcJxiSCTC", true, false},
		{"t1VJL2dPUyXK7avDRGqhqQA5bw2eEMdhyg6", true, false},
		{"tmSqX1vTCEnRoMf7mccjXQh9frHyxMYirLs", true, true},
		{"tmSqX1vTCEnRoMf7mccjXQh9frHyxMYixLs", false, false},
	}

	for _, x := range table {
		if ok, testnet := IsValidAddress(x.str); ok != x.ok || testnet != x.testnet {
			log.Println(x)
			t.Fatal("Failed")
		}
	}
}
