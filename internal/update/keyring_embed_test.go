package update

import (
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"
)

func TestProductionTrustRootIsEmbedded(t *testing.T) {
	if len(ProductionTrustRoot) != ed25519.PublicKeySize {
		t.Fatalf("ProductionTrustRoot len=%d want %d", len(ProductionTrustRoot), ed25519.PublicKeySize)
	}
	hexKey := hex.EncodeToString(ProductionTrustRoot)
	if strings.Contains(strings.ToUpper(hexKey), "REPLACE") {
		t.Fatal("production trust root must not be a placeholder")
	}
	kr := NewKeyRing()
	if _, err := kr.VerifyManifest([]byte(`{"schema_version":1}`), "00"); err == nil {
		t.Fatal("malformed manifest must still fail")
	}
	// Key must be registered under the canonical ID.
	if _, ok := kr.keys[ProductionTrustRootKeyID]; !ok {
		t.Fatalf("keyring missing %s", ProductionTrustRootKeyID)
	}
}
