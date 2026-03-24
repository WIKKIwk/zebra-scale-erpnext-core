package core

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/bits"
	"os"
	"sync"
	"time"
)

type EPCGenerator struct {
	mu     sync.Mutex
	lastNS int64
	seq    uint32
	salt   uint32
}

func NewEPCGenerator() *EPCGenerator {
	return &EPCGenerator{salt: newEPCSalt()}
}

func (g *EPCGenerator) Next(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}

	ns := t.UnixNano()
	g.mu.Lock()
	if ns != g.lastNS {
		g.lastNS = ns
		g.seq = 0
	} else {
		g.seq++
	}
	seq := g.seq
	salt := g.salt
	g.mu.Unlock()

	return formatEPC24(ns, seq, salt)
}

func formatEPC24(ns int64, seq, salt uint32) string {
	atom := uint32((uint64(ns) / 1_000) & 0xFFFFFFFF)
	tail := atom ^ bits.RotateLeft32(uint32(ns), 13) ^ bits.RotateLeft32(seq, 7) ^ salt
	tail |= 1
	return fmt.Sprintf("30%014X%08X", uint64(ns)&0x00FFFFFFFFFFFFFF, tail)
}

func newEPCSalt() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err == nil {
		return binary.BigEndian.Uint32(b[:]) | 1
	}
	fallback := uint32(time.Now().UnixNano()) ^ (uint32(os.Getpid()) << 16)
	return fallback | 1
}
