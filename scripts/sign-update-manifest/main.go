package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type artifact struct {
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Target string `json:"target"`
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
	publicKey := flag.String("public-key", "", "hex Ed25519 public key corresponding to the private key")
	baseURL := flag.String("base-url", "", "absolute release artifact base URL")
	flag.Parse()
	if *version == "" || *keyID == "" || *privateKey == "" {
		fmt.Fprintln(os.Stderr, "version, key-id and private-key are required")
		os.Exit(2)
	}
	if *baseURL == "" {
		*baseURL = "https://github.com/kivervinicius/ai-cli/releases/download/v" + strings.TrimPrefix(*version, "v")
	}
	parsedBase, err := url.Parse(*baseURL)
	if err != nil || parsedBase.Scheme != "https" || parsedBase.Host == "" {
		panic("base-url must be an absolute https URL")
	}
	base := strings.TrimRight(*baseURL, "/")
	key, err := decodePrivateKey(*privateKey)
	if err != nil {
		panic(err)
	}
	if *publicKey != "" {
		pub, err := decodePublicKey(*publicKey)
		if err != nil {
			panic(err)
		}
		if !key.Public().(ed25519.PublicKey).Equal(pub) {
			panic("public key does not match private signing key")
		}
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
		keyName, target, ok := releaseArtifactMetadata(entry.Name())
		if !ok {
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
		m.Artifacts[keyName] = artifact{URL: base + "/" + url.PathEscape(entry.Name()), Size: info.Size(), SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Target: target}
	}
	bytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := writeSignedManifest(*dist, bytes, key); err != nil {
		panic(err)
	}
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be 64 hex characters")
	}
	return ed25519.PublicKey(decoded), nil
}

func releaseArtifactMetadata(name string) (string, string, bool) {
	lower := strings.ToLower(name)
	if !strings.HasPrefix(lower, "nexus_") {
		return "", "", false
	}
	osName := ""
	for _, candidate := range []string{"linux", "darwin", "windows"} {
		if strings.Contains(lower, "_"+candidate+"_") {
			osName = candidate
			break
		}
	}
	arch := ""
	switch {
	case strings.Contains(lower, "_x86_64") || strings.Contains(lower, "_amd64"):
		arch = "amd64"
	case strings.Contains(lower, "_arm64"):
		arch = "arm64"
	}
	if osName == "" || arch == "" {
		return "", "", false
	}
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return osName + "_" + arch, "tar.gz", true
	case strings.HasSuffix(lower, ".zip"):
		return osName + "_" + arch, "zip", true
	case strings.HasSuffix(lower, ".deb"):
		return osName + "_" + arch + "_deb", "deb", true
	case strings.HasSuffix(lower, ".rpm"):
		return osName + "_" + arch + "_rpm", "rpm", true
	default:
		return "", "", false
	}
}

func writeSignedManifest(dist string, jsonBytes []byte, key ed25519.PrivateKey) error {
	// Sign exactly the bytes that are published. The trailing newline is part of
	// the manifest body and must be covered by Ed25519 verification.
	manifestBytes := append(append([]byte(nil), jsonBytes...), '\n')
	sig := ed25519.Sign(key, manifestBytes)
	if err := os.WriteFile(filepath.Join(dist, "update-manifest.json"), manifestBytes, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dist, "update-manifest.sig"), []byte(hex.EncodeToString(sig)+"\n"), 0644)
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
