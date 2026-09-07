package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSignedManifestSignsPublishedBytes(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dist := t.TempDir()
	jsonBytes := []byte(`{"schema_version":1,"channel":"stable","version":"1.0.0"}`)

	if err := writeSignedManifest(dist, jsonBytes, privateKey); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(dist, "update-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	signatureText, err := os.ReadFile(filepath.Join(dist, "update-manifest.sig"))
	if err != nil {
		t.Fatal(err)
	}
	signature, err := hex.DecodeString(strings.TrimSpace(string(signatureText)))
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(publicKey, manifestBytes, signature) {
		t.Fatal("signature does not cover the published manifest bytes")
	}
}
