package ulid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const encoding = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// ULID is a 16-byte Universally Unique Lexicographically Sortable Identifier.
type ULID [16]byte

// Generator produces time-ordered ULIDs with monotonic guarantees within the
// same millisecond (the random part is incremented rather than re-randomized).
type Generator struct {
	mu     sync.Mutex
	lastMs int64
	rand   [10]byte
}

var global = &Generator{}

// New generates a ULID using the package-level generator.
func New() (ULID, error) {
	return global.New()
}

// Must panics if New returns an error.
func Must() ULID {
	return global.Must()
}

// New generates a ULID.
func (g *Generator) New() (ULID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	nowMs := time.Now().UnixMilli()

	if nowMs == g.lastMs {
		// Same ms: increment the random part (big-endian carry from right)
		for i := 9; i >= 0; i-- {
			g.rand[i]++
			if g.rand[i] != 0 {
				break
			}
			if i == 0 {
				return ULID{}, errors.New("ulid: random part overflow in same millisecond")
			}
		}
	} else {
		if _, err := io.ReadFull(rand.Reader, g.rand[:]); err != nil {
			return ULID{}, fmt.Errorf("ulid: failed to read random bytes: %w", err)
		}
		g.lastMs = nowMs
	}

	var u ULID
	ms := uint64(nowMs)
	u[0] = byte(ms >> 40)
	u[1] = byte(ms >> 32)
	u[2] = byte(ms >> 24)
	u[3] = byte(ms >> 16)
	u[4] = byte(ms >> 8)
	u[5] = byte(ms)
	copy(u[6:], g.rand[:])

	return u, nil
}

// Must panics if New returns an error.
func (g *Generator) Must() ULID {
	u, err := g.New()
	if err != nil {
		panic(err)
	}
	return u
}

// String returns the 26-character Crockford base32 encoded ULID.
func (u ULID) String() string {
	var buf [26]byte
	// timestamp (48 bits → 10 base32 chars)
	ms := uint64(u[0])<<40 | uint64(u[1])<<32 | uint64(u[2])<<24 |
		uint64(u[3])<<16 | uint64(u[4])<<8 | uint64(u[5])
	buf[0] = encoding[(ms>>45)&0x1f]
	buf[1] = encoding[(ms>>40)&0x1f]
	buf[2] = encoding[(ms>>35)&0x1f]
	buf[3] = encoding[(ms>>30)&0x1f]
	buf[4] = encoding[(ms>>25)&0x1f]
	buf[5] = encoding[(ms>>20)&0x1f]
	buf[6] = encoding[(ms>>15)&0x1f]
	buf[7] = encoding[(ms>>10)&0x1f]
	buf[8] = encoding[(ms>>5)&0x1f]
	buf[9] = encoding[ms&0x1f]
	// random (80 bits → 16 base32 chars)
	r := u[6:]
	buf[10] = encoding[(r[0]>>3)&0x1f]
	buf[11] = encoding[((r[0]<<2)|(r[1]>>6))&0x1f]
	buf[12] = encoding[(r[1]>>1)&0x1f]
	buf[13] = encoding[((r[1]<<4)|(r[2]>>4))&0x1f]
	buf[14] = encoding[((r[2]<<1)|(r[3]>>7))&0x1f]
	buf[15] = encoding[(r[3]>>2)&0x1f]
	buf[16] = encoding[((r[3]<<3)|(r[4]>>5))&0x1f]
	buf[17] = encoding[r[4]&0x1f]
	buf[18] = encoding[(r[5]>>3)&0x1f]
	buf[19] = encoding[((r[5]<<2)|(r[6]>>6))&0x1f]
	buf[20] = encoding[(r[6]>>1)&0x1f]
	buf[21] = encoding[((r[6]<<4)|(r[7]>>4))&0x1f]
	buf[22] = encoding[((r[7]<<1)|(r[8]>>7))&0x1f]
	buf[23] = encoding[(r[8]>>2)&0x1f]
	buf[24] = encoding[((r[8]<<3)|(r[9]>>5))&0x1f]
	buf[25] = encoding[r[9]&0x1f]
	return string(buf[:])
}

// Parse decodes a 26-character ULID string.
func Parse(s string) (ULID, error) {
	s = strings.ToUpper(s)
	if len(s) != 26 {
		return ULID{}, fmt.Errorf("ulid: invalid length %d, want 26", len(s))
	}

	var dec [256]byte
	for i := range dec {
		dec[i] = 0xff
	}
	for i, c := range encoding {
		dec[c] = byte(i)
	}

	var vals [26]byte
	for i, c := range s {
		v := dec[c]
		if v == 0xff {
			return ULID{}, fmt.Errorf("ulid: invalid character %q at position %d", c, i)
		}
		vals[i] = v
	}

	var u ULID

	// Timestamp: 10 base32 chars → 48 bits → u[0..5]
	ms := uint64(vals[0])<<45 | uint64(vals[1])<<40 | uint64(vals[2])<<35 |
		uint64(vals[3])<<30 | uint64(vals[4])<<25 | uint64(vals[5])<<20 |
		uint64(vals[6])<<15 | uint64(vals[7])<<10 | uint64(vals[8])<<5 | uint64(vals[9])
	u[0] = byte(ms >> 40)
	u[1] = byte(ms >> 32)
	u[2] = byte(ms >> 24)
	u[3] = byte(ms >> 16)
	u[4] = byte(ms >> 8)
	u[5] = byte(ms)

	// Random: 16 base32 chars → 80 bits → u[6..15]
	// 80 bits doesn't fit in a single uint64; split into two chunks.
	// Upper 16 bits (vals[10..13] contribute the first 2 bytes):
	//   vals[10] = bits 79..75, vals[11] = bits 74..70, vals[12] = bits 69..65, vals[13] = bits 64..60
	// Lower 64 bits (vals[13] partial .. vals[25]):
	// We reconstruct byte by byte directly from the 5-bit groups.
	rndVals := vals[10:]
	// Reconstruct 10 bytes from 16 five-bit values (80 bits total).
	// Concatenate all 80 bits and emit bytes.
	//   Byte 0 (r[0]): bits 79-72 → vals[10] (5 bits) + vals[11] (top 3 bits)
	//   Byte 1 (r[1]): bits 71-64 → vals[11] (low 2 bits) + vals[12] (5 bits) + vals[13] (top 1 bit)
	//   Byte 2 (r[2]): bits 63-56 → vals[13] (low 4 bits) + vals[14] (5 bits) + vals[15] (top ... )
	// This matches the bit layout used in String().
	u[6] = rndVals[0]<<3 | rndVals[1]>>2
	u[7] = rndVals[1]<<6 | rndVals[2]<<1 | rndVals[3]>>4
	u[8] = rndVals[3]<<4 | rndVals[4]>>1
	u[9] = rndVals[4]<<7 | rndVals[5]<<2 | rndVals[6]>>3
	u[10] = rndVals[6]<<5 | rndVals[7]
	u[11] = rndVals[8]<<3 | rndVals[9]>>2
	u[12] = rndVals[9]<<6 | rndVals[10]<<1 | rndVals[11]>>4
	u[13] = rndVals[11]<<4 | rndVals[12]>>1
	u[14] = rndVals[12]<<7 | rndVals[13]<<2 | rndVals[14]>>3
	u[15] = rndVals[14]<<5 | rndVals[15]

	return u, nil
}

// Time extracts the embedded timestamp.
func (u ULID) Time() time.Time {
	ms := uint64(u[0])<<40 | uint64(u[1])<<32 | uint64(u[2])<<24 |
		uint64(u[3])<<16 | uint64(u[4])<<8 | uint64(u[5])
	return time.UnixMilli(int64(ms))
}

// Bytes returns the raw 16-byte representation.
func (u ULID) Bytes() []byte {
	b := make([]byte, 16)
	copy(b, u[:])
	return b
}
