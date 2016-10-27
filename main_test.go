package main

import (
	"log"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
	m.Run()
}
