package base58

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestCheck(t *testing.T) {
	table := []struct {
		str   string
		value string
		ok    bool
	}{
		{str: "5KMRizy9Xp5wLMbLEaDvkV5rWL1znyfMSF6CuVhdKbka1VUamXY", value: "80CA677924AF50FA95D087E797C6D9DA1B58BA78651ECD557348441C0B14DBDD80", ok: true},
		{str: "1KAyFgLxpbPXEUHaPomC82iBvW5DqWVa5L", value: "00C754F728CCC7C48616F5854EAD1E548ED2227F58", ok: true},
		{str: "1GTSsjyZaGhQM9F48fgZ2qdrzDZJ82N7gj", value: "00A98A4D0E05DE87B6C0646FAC9C1D3B3658BAD5FB", ok: true},
		{str: "16wMu7bHyHhRHtpUbZMHuL4GChF2cVEK6Z", value: "004120893AD60FDDA60D2500C2496DD26220C263F5", ok: true},
		{str: "1BszU6UDgRyBz44464eyf9S9F5Jobf5Pi1", value: "00775606EEE058684A3EE0260A555096FA440609FA", ok: true},
		{str: "1KG7GD39TcAWTJriXXqT5iDcvJRK2QsajM", value: "00C84DBB1FE14D1977224F183607EB21645B637F36", ok: true},
		{str: "1NzAqNnF6V1h6DTXjefbDx6V8EdoTSCved", value: "00F12A8F728E92399568D50C0DD48920CAF9D82594", ok: true},

		{str: "2GTSsjyZaGhQM9F48fgZ2qdrzDZJ82N7gj", value: "00A98A4D0E05DE87B6C0646FAC9C1D3B3658BAD5FB", ok: false},
		{str: "16wMu7bHyHhRHtpUbZMHuL4GChF2cVEK6a", value: "004120893AD60FDDA60D2500C2496DD26220C263F5", ok: false},
		{str: "1BszU6UDgRyBz44464eyf9S9F5Jobf5Pi0", value: "00775606EEE058684A3EE0260A555096FA440609FA", ok: false},
		{str: "1KGzGD39TcAWTJriXXqT5iDcvJRK2QsajM", value: "00C84DBB1FE14D1977224F183607EB21645B637F36", ok: false},
		{str: "1NzAqNnq6V1h6DTXjefbDx6V8EdoTSCved", value: "00F12A8F728E92399568D50C0DD48920CAF9D82594", ok: false},
	}

	for _, x := range table {
		value, err := DecodeCheck(x.str)
		if err != nil {
			if !x.ok {
				continue
			}

			t.Fatal("Failed", err)
		}

		realVal, err := hex.DecodeString(x.value)
		if err != nil {
			t.Fatal("Bad data")
		}

		if !bytes.Equal(realVal, value) {
			t.Fatal("Failed")
		}
	}
}
