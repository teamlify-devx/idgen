package uuidv7

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// Sentinel errors for structured error handling.
var (
	ErrRandomSource = errors.New("uuidv7: failed to read random bytes")
	ErrInvalidUUID  = errors.New("uuidv7: invalid UUID string")
	ErrWrongVersion = errors.New("uuidv7: not a UUID v7")
)

// UUID is a 16-byte UUID v7.
type UUID [16]byte

// Generator produces time-ordered UUID v7 values with monotonic guarantees
// within the same millisecond.
type Generator struct {
	mu     sync.Mutex
	lastMs int64
	seq    uint16 // 12-bit counter, max 4095
}

var global = &Generator{}

// New generates a UUID v7 using the package-level generator.
func New() (UUID, error) {
	return global.New()
}

// Must panics if New returns an error.
func Must() UUID {
	return global.Must()
}

// New generates a UUID v7.
func (g *Generator) New() (UUID, error) {
	g.mu.Lock()
	nowMs := time.Now().UnixMilli()
	var seq uint16
	if nowMs <= g.lastMs {
		nowMs = g.lastMs
		g.seq++
		if g.seq > 0x0fff {
			// overflow: busy-wait for next ms
			for nowMs <= g.lastMs {
				g.mu.Unlock()
				time.Sleep(time.Millisecond)
				g.mu.Lock()
				nowMs = time.Now().UnixMilli()
			}
			g.seq = 0
		}
	} else {
		g.seq = 0
	}
	seq = g.seq
	g.lastMs = nowMs
	g.mu.Unlock()

	var randB [8]byte
	if _, err := io.ReadFull(rand.Reader, randB[:]); err != nil {
		return UUID{}, fmt.Errorf("%w: %w", ErrRandomSource, err)
	}

	var uuid UUID
	// unix_ts_ms: bits 0-47
	binary.BigEndian.PutUint64(uuid[0:8], uint64(nowMs)<<16)
	// version: bits 48-51 = 0111 (7)
	uuid[6] = (uuid[6] & 0x0f) | 0x70
	// rand_a: bits 52-63 (12 bits of monotonic sequence)
	uuid[6] = (uuid[6] & 0xf0) | byte(seq>>8)
	uuid[7] = byte(seq & 0xff)
	// rand_b: bits 64-127
	copy(uuid[8:], randB[:])
	// variant: bits 64-65 = 10
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return uuid, nil
}

// Must panics if New returns an error.
func (g *Generator) Must() UUID {
	uuid, err := g.New()
	if err != nil {
		panic(err)
	}
	return uuid
}

// String returns the canonical hyphenated form.
func (u UUID) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u[10:16])
	return string(buf[:])
}

// Time extracts the embedded timestamp from a UUID v7.
func (u UUID) Time() (time.Time, error) {
	if u[6]>>4 != 7 {
		return time.Time{}, fmt.Errorf("%w: version nibble is %d", ErrWrongVersion, u[6]>>4)
	}
	ms := int64(binary.BigEndian.Uint64(u[0:8]) >> 16)
	return time.UnixMilli(ms), nil
}

// Bytes returns the raw 16-byte representation.
func (u UUID) Bytes() []byte {
	b := make([]byte, 16)
	copy(b, u[:])
	return b
}

// Parse decodes a canonical UUID string (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
// Returns ErrInvalidUUID if the format is wrong or ErrWrongVersion if version != 7.
func Parse(s string) (UUID, error) {
	if len(s) != 36 {
		return UUID{}, fmt.Errorf("%w: length must be 36, got %d", ErrInvalidUUID, len(s))
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return UUID{}, fmt.Errorf("%w: missing hyphens at positions 8, 13, 18, 23", ErrInvalidUUID)
	}

	var uuid UUID
	seg := [5]struct{ dst, src int }{
		{0, 0}, {4, 9}, {6, 14}, {8, 19}, {10, 24},
	}
	lens := [5]int{4, 2, 2, 2, 6}
	for i, info := range seg {
		n, err := hex.Decode(uuid[info.dst:info.dst+lens[i]], []byte(s[info.src:info.src+lens[i]*2]))
		if err != nil || n != lens[i] {
			return UUID{}, fmt.Errorf("%w: invalid hex in segment %d", ErrInvalidUUID, i+1)
		}
	}

	if uuid[6]>>4 != 7 {
		return UUID{}, fmt.Errorf("%w: version nibble is %d", ErrWrongVersion, uuid[6]>>4)
	}
	return uuid, nil
}
