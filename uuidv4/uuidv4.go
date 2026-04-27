package uuidv4

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// Sentinel errors for structured error handling.
var (
	ErrRandomSource = errors.New("uuidv4: failed to read random bytes")
	ErrInvalidUUID  = errors.New("uuidv4: invalid UUID string")
)

// UUID is a 16-byte UUID v4.
type UUID [16]byte

// New generates a random UUID v4.
func New() (UUID, error) {
	var uuid UUID
	if _, err := io.ReadFull(rand.Reader, uuid[:]); err != nil {
		return uuid, fmt.Errorf("%w: %w", ErrRandomSource, err)
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant bits
	return uuid, nil
}

// Must panics if New returns an error. Suitable for package-level vars.
func Must() UUID {
	uuid, err := New()
	if err != nil {
		panic(err)
	}
	return uuid
}

// String returns the canonical hyphenated form: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
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

// Bytes returns the raw 16-byte representation.
func (u UUID) Bytes() []byte {
	b := make([]byte, 16)
	copy(b, u[:])
	return b
}

// Parse decodes a canonical UUID string (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
// Returns ErrInvalidUUID if the format is wrong or the version nibble is not 4.
func Parse(s string) (UUID, error) {
	if len(s) != 36 {
		return UUID{}, fmt.Errorf("%w: length must be 36, got %d", ErrInvalidUUID, len(s))
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return UUID{}, fmt.Errorf("%w: missing hyphens at positions 8, 13, 18, 23", ErrInvalidUUID)
	}

	// Decode hex segments in order: 4-2-2-2-6 bytes.
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

	if uuid[6]>>4 != 4 {
		return UUID{}, fmt.Errorf("%w: version nibble is %d, want 4", ErrInvalidUUID, uuid[6]>>4)
	}
	return uuid, nil
}
