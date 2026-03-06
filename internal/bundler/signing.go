package bundler

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ComputePackageHash computes the content hash for a bundle directory using the
// same algorithm as the CodePush SDK on-device verification:
//   - Walk all files in dir, skipping .codepushrelease, __MACOSX, and .DS_Store
//   - SHA256 each file (hex-encoded)
//   - Build []string entries of "relpath:hexhash" using filepath.Dir(dir) as
//     basePath, so paths include the directory name (e.g. "CodePush/index.bundle")
//   - Sort entries alphabetically
//   - json.Marshal to a JSON array of strings
//   - Return hex(SHA256(jsonBytes))
func ComputePackageHash(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving bundle directory: %w", err)
	}

	basePath := filepath.Dir(absDir)

	var entries []string
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "__MACOSX" {
				return filepath.SkipDir
			}
			return nil
		}
		name := info.Name()
		if name == ".DS_Store" || name == ".codepushrelease" {
			return nil
		}

		relPath, relErr := filepath.Rel(basePath, path)
		if relErr != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, relErr)
		}
		relPath = filepath.ToSlash(relPath)

		hash, hashErr := sha256File(path)
		if hashErr != nil {
			return fmt.Errorf("hashing %s: %w", path, hashErr)
		}
		entries = append(entries, relPath+":"+hash)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walking bundle directory: %w", err)
	}

	sort.Strings(entries)

	manifestJSON, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("marshaling manifest: %w", err)
	}

	sum := sha256.Sum256(manifestJSON)
	return hex.EncodeToString(sum[:]), nil
}

// SignBundle signs the bundle directory and writes a .codepushrelease JWT file.
//
// The directory MUST be named "CodePush": the mobile SDK verifies package hashes
// using the directory name as a path prefix, and any other name produces a hash
// mismatch that causes the signed update to be rejected on-device.
//
// Signing must be called after Hermes compilation so the .hbc files (not the
// original .js files) are included in the hash.
func SignBundle(dir string, keyPath string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving bundle directory: %w", err)
	}

	if filepath.Base(absDir) != "CodePush" {
		return fmt.Errorf(
			"output directory must be named \"CodePush\" when signing: got %q"+
				" (the SDK verifies hashes using the directory name as a path prefix)",
			filepath.Base(absDir),
		)
	}

	key, err := loadRSAPrivateKey(keyPath)
	if err != nil {
		return fmt.Errorf("loading private key: %w", err)
	}

	contentHash, err := ComputePackageHash(absDir)
	if err != nil {
		return fmt.Errorf("computing package hash: %w", err)
	}

	jwt, err := buildRS256JWT(key, contentHash)
	if err != nil {
		return fmt.Errorf("building JWT: %w", err)
	}

	releasePath := filepath.Join(absDir, ".codepushrelease")
	if err := os.WriteFile(releasePath, []byte(jwt), 0o644); err != nil {
		return fmt.Errorf("writing .codepushrelease: %w", err)
	}

	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in %s", path)
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing PKCS1 private key: %w", err)
		}
		return key, nil
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing PKCS8 private key: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("PKCS8 key is not an RSA key")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}

func buildRS256JWT(key *rsa.PrivateKey, contentHash string) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}

	payloadJSON, err := json.Marshal(map[string]any{
		"claimVersion": "1.0.0",
		"contentHash":  contentHash,
		"iat":          time.Now().Unix(),
	})
	if err != nil {
		return "", err
	}

	h := base64RawURL(headerJSON)
	p := base64RawURL(payloadJSON)
	msg := h + "." + p

	sum := sha256.Sum256([]byte(msg))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("signing: %w", err)
	}

	return msg + "." + base64RawURL(sig), nil
}

func base64RawURL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
