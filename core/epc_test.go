package core

import (
	"testing"
	"time"
)

func TestEPCGenerator_Next_Unique(t *testing.T) {
	g := NewEPCGenerator()
	t0 := time.Unix(1_700_000_000, 123_456_789)
	a := g.Next(t0)
	b := g.Next(t0)

	if len(a) != 24 || len(b) != 24 {
		t.Fatalf("epc len mismatch: a=%d b=%d", len(a), len(b))
	}
	if a == b {
		t.Fatalf("epc should be unique: %s", a)
	}
}
