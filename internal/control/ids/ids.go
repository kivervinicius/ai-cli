// Package ids provides collision-resistant, sortable identifier generation
// used across the control plane (runtime IDs, session IDs).
package ids

import (
	"crypto/rand"
	"time"
)

// ulidAlphabet is the Crockford base32 alphabet used by ULID (excludes I,L,O,U).
const ulidAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewRuntimeID returns a ULID-based runtime identifier.
// Requirements: collision-resistant (80 random bits), sortable by creation time,
// URL-safe and filename-safe (26 chars, no separators).
func NewRuntimeID() string {
	return NewULID()
}

// NewULID returns a new ULID string (128-bit: 48-bit millisecond timestamp +
// 80 random bits) encoded in 26 Crockford base32 characters.
func NewULID() string {
	var b [16]byte
	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	if _, err := rand.Read(b[6:]); err != nil {
		// Entropy source failure: degrade to time-derived bytes rather than panic.
		now := uint64(time.Now().UnixNano())
		for i := 6; i < len(b); i++ {
			now ^= now >> 7
			now *= 0x9E3779B97F4A7C15
			b[i] = byte(now >> (uint(i-6) % 8 * 8))
		}
	}
	return encodeULID(b)
}

// encodeULID encodes 128 bits into 26 Crockford base32 chars (130 bits, 2 pad bits).
func encodeULID(b [16]byte) string {
	var out [26]byte
	var acc uint32
	var bits uint
	idx := 0
	for i := 0; i < len(b); i++ {
		acc = (acc << 8) | uint32(b[i])
		bits += 8
		for bits >= 5 {
			bits -= 5
			out[idx] = ulidAlphabet[(acc>>bits)&0x1f]
			idx++
			acc &= (1 << bits) - 1
		}
	}
	// Remaining 3 bits padded with 2 zero bits produce the 26th character.
	if bits > 0 {
		acc <<= 5 - bits
		out[idx] = ulidAlphabet[acc&0x1f]
		idx++
	}
	for idx < 26 {
		out[idx] = ulidAlphabet[0]
		idx++
	}
	return string(out[:])
}

// DecodeULID reverses encodeULID (128 bits from 26 chars).
func DecodeULID(s string) ([16]byte, bool) {
	var out [16]byte
	if len(s) != 26 {
		return out, false
	}
	dec := [256]int8{}
	for i := range dec {
		dec[i] = -1
	}
	for i := 0; i < len(ulidAlphabet); i++ {
		dec[ulidAlphabet[i]] = int8(i)
	}
	var acc uint32
	var bits uint
	outIdx := 0
	for _, c := range []byte(s) {
		v := dec[c]
		if v < 0 {
			return out, false
		}
		acc = (acc << 5) | uint32(v)
		bits += 5
		if bits >= 8 {
			bits -= 8
			out[outIdx] = byte((acc >> bits) & 0xff)
			outIdx++
			acc &= (1 << bits) - 1
		}
	}
	return out, outIdx == len(out)
}
