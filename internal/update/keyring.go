package update

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	// ProductionTrustRootHex is injected into release binaries with -ldflags.
	// A release build must provide the public key that corresponds to the signing
	// key kept in the protected release environment. Empty values intentionally
	// leave the default keyring without a production trust root.
	ProductionTrustRootHex string
	// ProductionTrustRoot is the decoded Ed25519 public key used for production signing.
	ProductionTrustRoot ed25519.PublicKey
)

func init() {
	ProductionTrustRoot, _ = decodePublicKey(ProductionTrustRootHex)
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("production trust root is not valid hex: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("production trust root must be %d bytes", ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(decoded), nil
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
