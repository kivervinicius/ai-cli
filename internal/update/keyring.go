package update

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrUntrustedKeyID    = errors.New("key ID is not trusted or recognized in keyring")
	ErrInvalidSignature  = errors.New("ed25519 signature verification failed for manifest")
	ErrInvalidSchema     = errors.New("unsupported manifest schema version")
	ErrManifestMalformed = errors.New("manifest JSON is malformed")
)

const (
	// ProductionTrustRootKeyID is the canonical key ID used for production Nexus releases.
	ProductionTrustRootKeyID = "nexus-signing-key-2026-v1"
)

var (
	// ProductionTrustRoot is the hex-encoded Ed25519 public key used for production signing.
	// This key must be embedded at build time and cannot be overridden without recompilation.
	// To rotate: generate a new key pair, update this constant, rebuild, and sign the new manifest.
	ProductionTrustRoot ed25519.PublicKey
)

func init() {
	// Embed the production trust root at compile time.
	// This is a placeholder - replace with actual generated key before production releases.
	const hexKey = "REPLACE_WITH_GENERATED_HEX_PUBLIC_KEY"
	pub, err := hex.DecodeString(hexKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		// If the trust root is not properly embedded, the keyring will be empty
		// and signature verification will fail with ErrUntrustedKeyID.
		return
	}
	ProductionTrustRoot = ed25519.PublicKey(pub)
}

type KeyRing struct {
	keys map[string]ed25519.PublicKey
}

func NewKeyRing() *KeyRing {
	kr := &KeyRing{
		keys: make(map[string]ed25519.PublicKey),
	}
	// Always include the production trust root when available
	if ProductionTrustRoot != nil {
		kr.keys[ProductionTrustRootKeyID] = ProductionTrustRoot
	}
	return kr
}

func (kr *KeyRing) AddKey(keyID string, pub ed25519.PublicKey) {
	kr.keys[keyID] = pub
}

func (kr *KeyRing) VerifyManifest(manifestBytes []byte, sigHex string) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrManifestMalformed, err)
	}

	if err := manifest.Validate(ManifestPolicy{}); err != nil {
		if manifest.SchemaVersion != 1 {
			return nil, fmt.Errorf("%w: version %d", ErrInvalidSchema, manifest.SchemaVersion)
		}
		return nil, err
	}

	pubKey, ok := kr.keys[manifest.KeyID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUntrustedKeyID, manifest.KeyID)
	}

	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, fmt.Errorf("%w: malformed hex signature: %v", ErrInvalidSignature, err)
	}

	if !ed25519.Verify(pubKey, manifestBytes, sigBytes) {
		return nil, ErrInvalidSignature
	}

	return &manifest, nil
}
