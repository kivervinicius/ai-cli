package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type artifact struct {
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type manifest struct {
	SchemaVersion int                 `json:"schema_version"`
	Channel       string              `json:"channel"`
	Version       string              `json:"version"`
	ReleaseDate   string              `json:"release_date"`
	ExpiresAt     string              `json:"expires_at"`
	KeyID         string              `json:"key_id"`
	Artifacts     map[string]artifact `json:"artifacts"`
}

func main() {
	dist := flag.String("dist", "dist", "GoReleaser dist directory")
	version := flag.String("version", "", "release version")
	keyID := flag.String("key-id", "", "trusted public-key identifier")
	privateKey := flag.String("private-key", "", "base64 or hex Ed25519 private key")
	flag.Parse()
	if *version == "" || *keyID == "" || *privateKey == "" {
		fmt.Fprintln(os.Stderr, "version, key-id and private-key are required")
		os.Exit(2)
	}
	key, err := decodePrivateKey(*privateKey)
	if err != nil {
		panic(err)
	}
	m := manifest{SchemaVersion: 1, Channel: channel(*version), Version: strings.TrimPrefix(*version, "v"), ReleaseDate: time.Now().UTC().Format(time.RFC3339), ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339), KeyID: *keyID, Artifacts: map[string]artifact{}}
	entries, err := os.ReadDir(*dist)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".txt") || strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(*dist, entry.Name())
		info, err := entry.Info()
		if err != nil {
			panic(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		keyName := strings.ToLower(strings.ReplaceAll(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), "-", "_"))
		m.Artifacts[keyName] = artifact{URL: entry.Name(), Size: info.Size(), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	}
	bytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	sig := ed25519.Sign(key, bytes)
	if err := os.WriteFile(filepath.Join(*dist, "update-manifest.json"), append(bytes, '\n'), 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(*dist, "update-manifest.sig"), []byte(hex.EncodeToString(sig)+"\n"), 0644); err != nil {
		panic(err)
	}
}

func decodePrivateKey(value string) (ed25519.PrivateKey, error) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) == ed25519.PrivateKeySize {
		return ed25519.PrivateKey(decoded), nil
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key must be base64 or hex Ed25519 key")
	}
	return ed25519.PrivateKey(decoded), nil
}

func channel(version string) string {
	if strings.Contains(strings.ToLower(version), "beta") {
		return "beta"
	}
	return "stable"
}
