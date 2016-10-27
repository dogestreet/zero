package main

import "github.com/dogestreet/zero/base58"

// IsValidAddress checks if the address is a valid z.cash address.
// Also returns if it a testnet address or not.
func IsValidAddress(addr string) (valid bool, testnet bool) {
	address, err := base58.DecodeCheck(addr)
	if err != nil {
		return false, false
	}

	switch len(address) {
	case 1 + 1 + 20: // T-Address
		if address[0] == 0x1c && (address[1] == 0xbd || address[1] == 0xb8) {
			return true, false
		}
		if address[0] == 0x1d && (address[1] == 0xba || address[1] == 0x25) {
			return true, true
		}
	case 1 + 1 + 32 + 32: // P-Address
		if address[0] == 0x16 {
			if address[1] == 0x9a {
				return true, false
			} else if address[1] == 0xb6 {
				return true, true
			}
		}
	default:
		return false, false
	}

	return false, false
}
