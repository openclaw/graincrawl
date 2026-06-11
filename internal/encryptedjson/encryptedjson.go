package encryptedjson

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const (
	CacheFile    = "cache-v6.json.enc"
	SupabaseFile = "supabase.json.enc"

	keychainService = "Granola Safe Storage"
	keychainAccount = "Granola Key"
	maxFileBytes    = 64 << 20
	unlockTimeout   = 30 * time.Second
)

var allowedFiles = map[string]bool{
	CacheFile:    true,
	SupabaseFile: true,
}

type DecryptFunc func(context.Context, string, ...string) (map[string]json.RawMessage, error)

type snapshotData struct {
	StorageDEK []byte
	Files      map[string][]byte
}

func Decrypt(ctx context.Context, profile string, names ...string) (map[string]json.RawMessage, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("encrypted-json unlock is supported only on macOS")
	}
	snapshot, err := snapshot(profile, names)
	if err != nil {
		return nil, err
	}
	defer snapshot.clear()
	unlockCtx, cancel := context.WithTimeout(ctx, unlockTimeout)
	defer cancel()
	files, err := decryptSnapshot(unlockCtx, snapshot, readKeychainSecret)
	if errors.Is(unlockCtx.Err(), context.DeadlineExceeded) {
		return nil, errors.New("encrypted storage unlock timed out waiting for Keychain access")
	}
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if _, ok := files[name]; !ok {
			Clear(files)
			return nil, errors.New("encrypted storage decryptor omitted a requested file")
		}
	}
	return files, nil
}

func decryptSnapshot(ctx context.Context, snapshot snapshotData, keychain func(context.Context) ([]byte, error)) (map[string]json.RawMessage, error) {
	secret, err := keychain(ctx)
	if err != nil {
		return nil, errors.New("encrypted storage unlock failed; Keychain access may have been denied")
	}
	defer zero(secret)
	dek, err := unwrapDEK(secret, snapshot.StorageDEK)
	if err != nil {
		return nil, errors.New("encrypted storage could not be decrypted with the current Granola Keychain item")
	}
	defer zero(dek)
	files := make(map[string]json.RawMessage, len(snapshot.Files))
	for name, encrypted := range snapshot.Files {
		plain, err := decryptJSON(dek, encrypted)
		if err != nil {
			Clear(files)
			return nil, errors.New("encrypted storage could not be decrypted with the current Granola Keychain item")
		}
		files[name] = plain
	}
	return files, nil
}

func Clear(files map[string]json.RawMessage) {
	for name, data := range files {
		zero(data)
		delete(files, name)
	}
}

func (snapshot snapshotData) clear() {
	zero(snapshot.StorageDEK)
	for name, data := range snapshot.Files {
		zero(data)
		delete(snapshot.Files, name)
	}
}

func snapshot(profile string, names []string) (snapshotData, error) {
	if len(names) == 0 {
		return snapshotData{}, errors.New("no encrypted JSON files requested")
	}
	result := snapshotData{Files: make(map[string][]byte, len(names))}
	var err error
	result.StorageDEK, err = readStableRegularFile(filepath.Join(profile, "storage.dek"))
	if err != nil {
		return snapshotData{}, fmt.Errorf("read storage.dek: %w", err)
	}
	for _, name := range names {
		if !allowedFiles[name] {
			result.clear()
			return snapshotData{}, fmt.Errorf("unsupported encrypted JSON file %q", name)
		}
		if _, exists := result.Files[name]; exists {
			continue
		}
		result.Files[name], err = readStableRegularFile(filepath.Join(profile, name))
		if err != nil {
			result.clear()
			return snapshotData{}, fmt.Errorf("read %s: %w", name, err)
		}
	}
	return result, nil
}

func readStableRegularFile(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("file is not regular")
	}
	if before.Size() > maxFileBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxFileBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, opened) {
		return nil, errors.New("file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxFileBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxFileBytes)
	}
	after, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(opened, after) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return nil, errors.New("file changed while reading")
	}
	return data, nil
}

func readKeychainSecret(ctx context.Context) ([]byte, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("unsupported platform")
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "find-generic-password", "-s", keychainService, "-a", keychainAccount, "-w")
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	output = bytes.TrimRight(output, "\r\n")
	if len(output) == 0 {
		return nil, errors.New("empty Keychain secret")
	}
	return output, nil
}

func unwrapDEK(password, wrapped []byte) ([]byte, error) {
	if len(wrapped) <= 3 || string(wrapped[:3]) != "v10" {
		return nil, errors.New("unsupported safeStorage payload")
	}
	key := pbkdf2SHA1(password, []byte("saltysalt"), 1003, 16)
	defer zero(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ciphertext := wrapped[3:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("invalid safeStorage ciphertext")
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, bytes.Repeat([]byte{' '}, aes.BlockSize)).CryptBlocks(plain, ciphertext)
	plain, err = unpadPKCS7(plain, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	dek := make([]byte, base64.StdEncoding.DecodedLen(len(plain)))
	n, err := base64.StdEncoding.Decode(dek, plain)
	zero(plain)
	dek = dek[:n]
	if err != nil || len(dek) != 32 {
		zero(dek)
		return nil, errors.New("invalid data encryption key")
	}
	return dek, nil
}

func pbkdf2SHA1(password, salt []byte, iterations, keyLen int) []byte {
	const hashLen = sha1.Size
	blocks := (keyLen + hashLen - 1) / hashLen
	derived := make([]byte, keyLen)
	offset := 0
	var blockIndex [4]byte
	for block := 1; block <= blocks; block++ {
		binary.BigEndian.PutUint32(blockIndex[:], uint32(block))
		mac := hmac.New(sha1.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write(blockIndex[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for iteration := 1; iteration < iterations; iteration++ {
			mac = hmac.New(sha1.New, password)
			_, _ = mac.Write(u)
			next := mac.Sum(nil)
			for i := range t {
				t[i] ^= next[i]
			}
			zero(u)
			u = next
		}
		zero(u)
		offset += copy(derived[offset:], t)
		zero(t)
	}
	return derived
}

func decryptJSON(dek, encrypted []byte) (json.RawMessage, error) {
	if len(dek) != 32 || len(encrypted) < 12+16 {
		return nil, errors.New("invalid encrypted JSON payload")
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, encrypted[:12], encrypted[12:], nil)
	if err != nil {
		return nil, err
	}
	if !json.Valid(plain) {
		zero(plain)
		return nil, errors.New("decrypted payload is not JSON")
	}
	return json.RawMessage(plain), nil
}

func unpadPKCS7(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid padding")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, errors.New("invalid padding")
	}
	for _, value := range data[len(data)-padding:] {
		if int(value) != padding {
			return nil, errors.New("invalid padding")
		}
	}
	return data[:len(data)-padding], nil
}

func zero(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
