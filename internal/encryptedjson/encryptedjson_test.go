package encryptedjson

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecryptSnapshotDecryptsCompatiblePayload(t *testing.T) {
	password := "synthetic-keychain-secret"
	dek := bytes.Repeat([]byte{0x42}, 32)
	cache := []byte(`{"cache":{"version":8,"state":{"documents":{},"transcripts":{}}}}`)
	snapshot := snapshotData{
		StorageDEK: wrapDEK(t, password, dek),
		Files: map[string][]byte{
			CacheFile: encryptJSON(t, dek, cache),
		},
	}
	secret := []byte(password)
	files, err := decryptSnapshot(context.Background(), snapshot, func(context.Context) ([]byte, error) {
		return secret, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer Clear(files)
	if !bytes.Equal(files[CacheFile], cache) {
		t.Fatalf("decrypted cache mismatch: %s", files[CacheFile])
	}
	if !bytes.Equal(secret, make([]byte, len(secret))) {
		t.Fatal("decryptor did not clear Keychain secret buffer")
	}
}

func TestDecryptSnapshotRedactsDeniedKeychainError(t *testing.T) {
	snapshot := snapshotData{
		StorageDEK: []byte("v10ciphertext"),
		Files:      map[string][]byte{CacheFile: []byte("ciphertext")},
	}
	_, err := decryptSnapshot(context.Background(), snapshot, func(context.Context) ([]byte, error) {
		return nil, errors.New("denied: synthetic-sensitive-detail")
	})
	if err == nil || strings.Contains(err.Error(), "synthetic-sensitive-detail") {
		t.Fatalf("decryptor leaked Keychain error: %v", err)
	}
}

func TestDecryptSnapshotRejectsTamperedCiphertext(t *testing.T) {
	password := "synthetic-keychain-secret"
	dek := bytes.Repeat([]byte{0x24}, 32)
	encrypted := encryptJSON(t, dek, []byte(`{"ok":true}`))
	encrypted[len(encrypted)-1] ^= 0xff
	snapshot := snapshotData{
		StorageDEK: wrapDEK(t, password, dek),
		Files:      map[string][]byte{CacheFile: encrypted},
	}
	files, err := decryptSnapshot(context.Background(), snapshot, func(context.Context) ([]byte, error) {
		return []byte(password), nil
	})
	if err == nil || len(files) != 0 {
		t.Fatalf("tampered payload was not rejected: files=%d err=%v", len(files), err)
	}
}

func TestDecryptSnapshotHonorsContextDeadline(t *testing.T) {
	snapshot := snapshotData{
		StorageDEK: []byte("v10ciphertext"),
		Files:      map[string][]byte{CacheFile: []byte("ciphertext")},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := decryptSnapshot(ctx, snapshot, func(ctx context.Context) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err == nil || !strings.Contains(err.Error(), "Keychain access") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func TestSnapshotRejectsSymlinkedInput(t *testing.T) {
	profile := t.TempDir()
	if err := os.WriteFile(filepath.Join(profile, "storage.dek"), []byte("wrapped"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(profile, "target")
	if err := os.WriteFile(target, []byte("ciphertext"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(profile, CacheFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot(profile, []string{CacheFile}); err == nil || !strings.Contains(err.Error(), "not regular") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func wrapDEK(t *testing.T, password string, dek []byte) []byte {
	t.Helper()
	plain := []byte(base64.StdEncoding.EncodeToString(dek))
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	plain = append(plain, bytes.Repeat([]byte{byte(padding)}, padding)...)
	key, err := pbkdf2.Key(sha1.New, password, []byte("saltysalt"), 1003, 16)
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	encrypted := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, bytes.Repeat([]byte{' '}, aes.BlockSize)).CryptBlocks(encrypted, plain)
	return append([]byte("v10"), encrypted...)
}

func encryptJSON(t *testing.T, dek, plain []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(dek)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		t.Fatal(err)
	}
	nonce := bytes.Repeat([]byte{0x7a}, 12)
	return append(nonce, gcm.Seal(nil, nonce, plain, nil)...)
}
