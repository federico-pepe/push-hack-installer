// Package sshsetup ports push-hack's SSH key wizard (scripts/install.sh's
// ssh_wizard) into Go: generate or load a keypair, and confirm Push has
// accepted it — without a terminal, without a blocking "press Enter".
package sshsetup

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// keyBits matches ssh-keygen's own modern default (it moved from 2048 to
// 3072 some years back); install.sh's `ssh-keygen -t rsa` with no -b picks
// whatever the local ssh-keygen defaults to, so this pins the same value
// explicitly rather than inheriting a system default that could vary.
const keyBits = 3072

// keyPath returns the same path install.sh's ssh_wizard uses:
// ~/.ssh/id_rsa (and id_rsa.pub next to it).
func keyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "id_rsa"), nil
}

// EnsureKey returns the authorized_keys-format public key line and its
// SHA256 fingerprint, generating a new RSA keypair at ~/.ssh/id_rsa if one
// doesn't already exist — mirrors ssh_wizard's "found existing key" /
// "generating one now" branch.
func EnsureKey() (pubKeyLine string, fingerprint string, err error) {
	privPath, err := keyPath()
	if err != nil {
		return "", "", err
	}
	pubPath := privPath + ".pub"

	if _, statErr := os.Stat(privPath); statErr != nil {
		if err := generateKey(privPath, pubPath); err != nil {
			return "", "", err
		}
	}

	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		return "", "", fmt.Errorf("read public key: %w", err)
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubBytes)
	if err != nil {
		return "", "", fmt.Errorf("parse public key: %w", err)
	}

	return string(ssh.MarshalAuthorizedKey(pubKey)), ssh.FingerprintSHA256(pubKey), nil
}

func generateKey(privPath, pubPath string) error {
	if err := os.MkdirAll(filepath.Dir(privPath), 0700); err != nil {
		return fmt.Errorf("create .ssh directory: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	if err := os.WriteFile(privPath, pem.EncodeToMemory(privBlock), 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		return fmt.Errorf("derive public key: %w", err)
	}
	if err := os.WriteFile(pubPath, ssh.MarshalAuthorizedKey(sshPub), 0644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	return nil
}
