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

type KeyRing struct {
	keys map[string]ed25519.PublicKey
}

func NewKeyRing() *KeyRing {
	return &KeyRing{
		keys: make(map[string]ed25519.PublicKey),
	}
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
