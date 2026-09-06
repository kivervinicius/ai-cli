package ids

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"
)

func TestEncodeULIDKnownVectors(t *testing.T) {
	// Golden vectors computed from the ULID spec algorithm.
	var allOnes [16]byte
	for i := range allOnes {
		allOnes[i] = 0x01
	}
	if got := encodeULID(allOnes); got != "040G2081040G2081040G208104" {
		t.Errorf("all-0x01 vector mismatch: got %q", got)
	}

	// Timestamp 1469918176385 with zero randomness.
	var tsBytes [16]byte
	binary.BigEndian.PutUint32(tsBytes[0:4], uint32(0x01563DF3)) // partial, covered below
	ts := uint64(1469918176385)
	tsBytes[0] = byte(ts >> 40)
	tsBytes[1] = byte(ts >> 32)
	tsBytes[2] = byte(ts >> 24)
	tsBytes[3] = byte(ts >> 16)
	tsBytes[4] = byte(ts >> 8)
	tsBytes[5] = byte(ts)
	if got := encodeULID(tsBytes); got != "05B3VWV4G40000000000000000" {
		t.Errorf("timestamp vector mismatch: got %q", got)
	}
}

func TestNewULIDRoundTrip(t *testing.T) {
	id := NewULID()
	if len(id) != 26 {
		t.Fatalf("expected 26-char ULID, got %d: %q", len(id), id)
	}
	for _, r := range id {
		if !strings.ContainsRune(ulidAlphabet, r) {
			t.Fatalf("ULID contains invalid character %q", r)
		}
	}
	dec, ok := DecodeULID(id)
	if !ok {
		t.Fatalf("failed to decode own ULID %q", id)
	}
	// Re-encoding the decoded bytes must reproduce the same string.
	if got := encodeULID(dec); got != id {
		t.Errorf("round-trip mismatch: %q != %q", got, id)
	}
	// First 6 bytes must be the millisecond timestamp.
	ms := binary.BigEndian.Uint64(append([]byte{0, 0}, dec[0:6]...))
	now := uint64(time.Now().UnixMilli())
	if now-ms > 5000 || ms > now {
		t.Errorf("timestamp out of range: decoded=%d now=%d", ms, now)
	}
}

func TestNewULIDUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		id := NewULID()
		if seen[id] {
			t.Fatalf("duplicate ULID generated: %q", id)
		}
		seen[id] = true
	}
}

func TestNewULIDSortable(t *testing.T) {
	// Lower timestamp must sort before higher timestamp (lexicographically).
	var low [16]byte
	var high [16]byte
	lowTs := uint64(1000)
	highTs := uint64(2000)
	for i := 0; i < 6; i++ {
		low[5-i] = byte(lowTs >> (8 * i))
		high[5-i] = byte(highTs >> (8 * i))
	}
	if encodeULID(low) >= encodeULID(high) {
		t.Errorf("expected low timestamp ULID to sort before high: %q >= %q",
			encodeULID(low), encodeULID(high))
	}
}

func TestDecodeULIDRejectsInvalid(t *testing.T) {
	if _, ok := DecodeULID(""); ok {
		t.Error("empty string must not decode")
	}
	if _, ok := DecodeULID("0000000000000000000000000"); ok {
		t.Error("invalid length must not decode")
	}
	if _, ok := DecodeULID("IIIIIIIIIIIIIIIIIIIIIIIIII"); ok {
		t.Error("invalid alphabet characters must not decode")
	}
}

func TestNewRuntimeID(t *testing.T) {
	a := NewRuntimeID()
	b := NewRuntimeID()
	if a == b {
		t.Fatal("runtime IDs must be unique")
	}
	if len(a) != 26 {
		t.Errorf("expected 26-char runtime ID, got %q", a)
	}
}
