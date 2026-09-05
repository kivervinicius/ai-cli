package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Status string

const (
	StatusSuccess    Status = "SUCCESS"
	StatusRolledBack Status = "ROLLED_BACK"
	StatusFailed     Status = "FAILED"
)

type Receipt struct {
	Timestamp       time.Time `json:"timestamp"`
	PreviousVersion string    `json:"previous_version"`
	AppliedVersion  string    `json:"applied_version"`
	SHA256          string    `json:"sha256"`
	Status          Status    `json:"status"`
	Message         string    `json:"message,omitempty"`
}

type Updater struct {
	BinaryPath string
	DataDir    string
}

func NewUpdater(binaryPath, dataDir string) *Updater {
	return &Updater{
		BinaryPath: binaryPath,
		DataDir:    dataDir,
	}
}

func (u *Updater) VerifyArtifactChecksum(data []byte, expectedHex string) error {
	sum := sha256.Sum256(data)
	actualHex := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actualHex, expectedHex) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHex, actualHex)
	}
	return nil
}

func (u *Updater) ApplyUpdate(prevVersion, newVersion string, newBinaryData []byte, sha256Hex string) (*Receipt, error) {
	if err := u.VerifyArtifactChecksum(newBinaryData, sha256Hex); err != nil {
		receipt := &Receipt{
			Timestamp:       time.Now().UTC(),
			PreviousVersion: prevVersion,
			AppliedVersion:  newVersion,
			SHA256:          sha256Hex,
			Status:          StatusFailed,
			Message:         err.Error(),
		}
		_ = u.saveReceipt(receipt)
		return receipt, err
	}

	backupPath := u.BinaryPath + ".bak"
	if _, err := os.Stat(u.BinaryPath); err == nil {
		_ = os.Remove(backupPath)
		if err := os.Rename(u.BinaryPath, backupPath); err != nil {
			return nil, fmt.Errorf("failed to backup current binary: %w", err)
		}
	}

	tempFile := u.BinaryPath + ".tmp"
	if err := os.WriteFile(tempFile, newBinaryData, 0755); err != nil {
		_ = os.Rename(backupPath, u.BinaryPath)
		return nil, fmt.Errorf("failed to write new binary: %w", err)
	}

	if err := os.Rename(tempFile, u.BinaryPath); err != nil {
		_ = os.Rename(backupPath, u.BinaryPath)
		return nil, fmt.Errorf("failed to replace binary: %w", err)
	}

	receipt := &Receipt{
		Timestamp:       time.Now().UTC(),
		PreviousVersion: prevVersion,
		AppliedVersion:  newVersion,
		SHA256:          sha256Hex,
		Status:          StatusSuccess,
		Message:         "Update applied successfully with atomic backup",
	}
	_ = u.saveReceipt(receipt)
	return receipt, nil
}

func (u *Updater) Rollback() (*Receipt, error) {
	backupPath := u.BinaryPath + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		return nil, errors.New("no backup binary found for rollback")
	}

	_ = os.Remove(u.BinaryPath)
	if err := os.Rename(backupPath, u.BinaryPath); err != nil {
		return nil, fmt.Errorf("failed to restore backup binary: %w", err)
	}

	receipt := &Receipt{
		Timestamp: time.Now().UTC(),
		Status:    StatusRolledBack,
		Message:   "Rolled back to previous binary version from backup",
	}
	_ = u.saveReceipt(receipt)
	return receipt, nil
}

func (u *Updater) saveReceipt(receipt *Receipt) error {
	receiptDir := filepath.Join(u.DataDir, "receipts")
	if err := os.MkdirAll(receiptDir, 0755); err != nil {
		return err
	}
	receiptPath := filepath.Join(receiptDir, fmt.Sprintf("update-%d.json", time.Now().UnixNano()))
	b, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(receiptPath, b, 0644)
}
