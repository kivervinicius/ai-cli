package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func generateTestKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}
	return pub, priv
}

func TestVerifyManifest_ValidSignature(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	keyID := "test-key-2026-v1"
	ring := NewKeyRing()
	ring.AddKey(keyID, pub)

	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         keyID,
		Artifacts: map[string]Artifact{
			"linux_amd64": {
				URL:    "https://github.com/kivervinicius/ai-cli/releases/download/v0.5.1/nexus_Linux_x86_64.tar.gz",
				Size:   1234567,
				SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		},
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	sig := ed25519.Sign(priv, manifestBytes)
	sigHex := hex.EncodeToString(sig)

	verified, err := ring.VerifyManifest(manifestBytes, sigHex)
	if err != nil {
		t.Fatalf("expected signature verification to succeed, got: %v", err)
	}
	if verified.Version != "0.5.1" {
		t.Errorf("expected version 0.5.1, got %s", verified.Version)
	}
	if verified.Channel != "beta" {
		t.Errorf("expected channel beta, got %s", verified.Channel)
	}
}

func TestVerifyManifest_TamperedManifest(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	keyID := "test-key-2026-v1"
	ring := NewKeyRing()
	ring.AddKey(keyID, pub)

	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         keyID,
	}

	manifestBytes, _ := json.Marshal(manifest)
	sig := ed25519.Sign(priv, manifestBytes)
	sigHex := hex.EncodeToString(sig)

	// Tamper bytes
	tamperedBytes := []byte(string(manifestBytes) + " ")

	_, err := ring.VerifyManifest(tamperedBytes, sigHex)
	if err == nil {
		t.Fatal("expected tampered manifest verification to fail, but it succeeded")
	}
}

func TestVerifyManifest_UntrustedKeyID(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	ring := NewKeyRing() // empty ring

	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "untrusted-key",
	}

	manifestBytes, _ := json.Marshal(manifest)
	sig := ed25519.Sign(priv, manifestBytes)
	sigHex := hex.EncodeToString(sig)

	_, err := ring.VerifyManifest(manifestBytes, sigHex)
	if err == nil {
		t.Fatal("expected verification to fail for untrusted key ID")
	}
}

func TestUpdatePlan_ChecksumVerificationAndRollback(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "nexus")

	// Create original binary
	origContent := []byte("nexus-v0.5.0-binary-content")
	if err := os.WriteFile(binPath, origContent, 0755); err != nil {
		t.Fatalf("failed to write original binary: %v", err)
	}

	// Target artifact
	newContent := []byte("nexus-v0.5.1-binary-content")
	sum := sha256.Sum256(newContent)
	sumHex := hex.EncodeToString(sum[:])

	updater := NewUpdater(binPath, tempDir)

	// 1. Verify Checksum matches
	if err := updater.VerifyArtifactChecksum(newContent, sumHex); err != nil {
		t.Fatalf("expected valid checksum to pass: %v", err)
	}

	// 2. Verify Checksum mismatch fails
	if err := updater.VerifyArtifactChecksum(newContent, "invalidhash"); err == nil {
		t.Fatal("expected invalid checksum to fail")
	}

	// 3. Apply update with atomic backup and receipt
	receipt, err := updater.ApplyUpdate("0.5.0", "0.5.1", newContent, sumHex)
	if err != nil {
		t.Fatalf("failed to apply update: %v", err)
	}
	if receipt.Status != StatusSuccess {
		t.Fatalf("expected status %s, got %s", StatusSuccess, receipt.Status)
	}

	// Verify new content is in place
	currBytes, _ := os.ReadFile(binPath)
	if string(currBytes) != string(newContent) {
		t.Fatalf("expected binary content %q, got %q", string(newContent), string(currBytes))
	}

	// 4. Test Rollback
	rollbackReceipt, err := updater.Rollback()
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}
	if rollbackReceipt.Status != StatusRolledBack {
		t.Fatalf("expected status %s, got %s", StatusRolledBack, rollbackReceipt.Status)
	}

	restoredBytes, _ := os.ReadFile(binPath)
	if string(restoredBytes) != string(origContent) {
		t.Fatalf("expected restored content %q, got %q", string(origContent), string(restoredBytes))
	}
}
