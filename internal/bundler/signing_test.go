package bundler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputePackageHash(t *testing.T) {
	t.Run("produces consistent hash", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "CodePush")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("bundle"), 0o644))

		hash1, err := ComputePackageHash(dir)
		require.NoError(t, err)
		hash2, err := ComputePackageHash(dir)
		require.NoError(t, err)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("excludes .codepushrelease from hash", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "CodePush")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("bundle"), 0o644))

		hashBefore, err := ComputePackageHash(dir)
		require.NoError(t, err)

		require.NoError(t, os.WriteFile(filepath.Join(dir, ".codepushrelease"), []byte("some-jwt"), 0o644))

		hashAfter, err := ComputePackageHash(dir)
		require.NoError(t, err)
		assert.Equal(t, hashBefore, hashAfter, ".codepushrelease must be excluded from hash")
	})

	t.Run("excludes .DS_Store from hash", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "CodePush")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("bundle"), 0o644))

		hashBefore, err := ComputePackageHash(dir)
		require.NoError(t, err)

		require.NoError(t, os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("mac metadata"), 0o644))

		hashAfter, err := ComputePackageHash(dir)
		require.NoError(t, err)
		assert.Equal(t, hashBefore, hashAfter, ".DS_Store must be excluded from hash")
	})

	t.Run("includes directory name as path prefix", func(t *testing.T) {
		parent := t.TempDir()
		dirA := filepath.Join(parent, "CodePush")
		require.NoError(t, os.Mkdir(dirA, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dirA, "index.js"), []byte("bundle"), 0o644))

		dirB := filepath.Join(t.TempDir(), "other")
		require.NoError(t, os.Mkdir(dirB, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dirB, "index.js"), []byte("bundle"), 0o644))

		hashA, err := ComputePackageHash(dirA)
		require.NoError(t, err)
		hashB, err := ComputePackageHash(dirB)
		require.NoError(t, err)

		assert.NotEqual(t, hashA, hashB, "hash must differ when directory name differs")
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		dirA := filepath.Join(t.TempDir(), "CodePush")
		require.NoError(t, os.Mkdir(dirA, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dirA, "index.js"), []byte("bundle-v1"), 0o644))

		dirB := filepath.Join(t.TempDir(), "CodePush")
		require.NoError(t, os.Mkdir(dirB, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dirB, "index.js"), []byte("bundle-v2"), 0o644))

		hashA, err := ComputePackageHash(dirA)
		require.NoError(t, err)
		hashB, err := ComputePackageHash(dirB)
		require.NoError(t, err)

		assert.NotEqual(t, hashA, hashB)
	})
}

func TestSignBundle(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyPath := writeRSAKey(t, key)

	t.Run("writes .codepushrelease JWT for CodePush directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "CodePush")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("bundle"), 0o644))

		err := SignBundle(dir, keyPath)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(dir, ".codepushrelease"))
		require.NoError(t, err)

		jwt := string(data)
		parts := strings.Split(jwt, ".")
		assert.Len(t, parts, 3, "JWT must have 3 dot-separated parts")
		assert.True(t, strings.HasPrefix(jwt, "eyJ"), "JWT header must start with eyJ (base64url of {)")
	})

	t.Run("returns error when directory is not named CodePush", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "wrong-name")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("bundle"), 0o644))

		err := SignBundle(dir, keyPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `must be named "CodePush"`)
		assert.Contains(t, err.Error(), "wrong-name")
	})

	t.Run("supports PKCS8 private key", func(t *testing.T) {
		pkcs8Path := writePKCS8Key(t, key)

		dir := filepath.Join(t.TempDir(), "CodePush")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("bundle"), 0o644))

		err := SignBundle(dir, pkcs8Path)
		require.NoError(t, err)

		_, err = os.ReadFile(filepath.Join(dir, ".codepushrelease"))
		require.NoError(t, err)
	})

	t.Run("hash is stable across sign calls (deterministic hash)", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "CodePush")
		require.NoError(t, os.Mkdir(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("bundle"), 0o644))

		err := SignBundle(dir, keyPath)
		require.NoError(t, err)

		// second call: .codepushrelease now exists but must be excluded from hash
		err = SignBundle(dir, keyPath)
		require.NoError(t, err)
	})
}

// writeRSAKey writes a PKCS1 RSA private key PEM file and returns its path.
func writeRSAKey(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key.pem")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.NoError(t, pem.Encode(f, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
	return path
}

// writePKCS8Key writes a PKCS8 RSA private key PEM file and returns its path.
func writePKCS8Key(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key_pkcs8.pem")
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	require.NoError(t, pem.Encode(f, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	}))
	return path
}
