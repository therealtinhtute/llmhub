package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	minisign "github.com/jedisct1/go-minisign"
)

const (
	keyPathEnv      = "MINISIGN_KEY_PATH"
	passwordPathEnv = "MINISIGN_PASSWORD_FILE"
)

func main() {
	args := os.Args[1:]
	command := run
	if len(args) > 0 && args[0] == "verify" {
		command = verify
		args = args[1:]
	}
	if err := command(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("sign-release-asset", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	artifactPath := flags.String("artifact", "", "path to the release artifact")
	signaturePath := flags.String("signature", "", "path to the detached signature")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *artifactPath == "" || *signaturePath == "" {
		return errors.New("artifact and signature paths are required")
	}
	if *artifactPath == *signaturePath {
		return errors.New("artifact and signature paths must differ")
	}
	resolvedArtifactPath, err := resolveArtifactPath(*artifactPath, *signaturePath)
	if err != nil {
		return fmt.Errorf("release artifact: %w", err)
	}

	keyPath := os.Getenv(keyPathEnv)
	passwordPath := os.Getenv(passwordPathEnv)
	if keyPath == "" || passwordPath == "" {
		return fmt.Errorf("%s and %s are required", keyPathEnv, passwordPathEnv)
	}

	password, err := readProtectedFile(passwordPath)
	if err != nil {
		return fmt.Errorf("password file: %w", err)
	}
	defer wipe(password)

	keyContents, err := readProtectedFile(keyPath)
	if err != nil {
		return fmt.Errorf("private key file: %w", err)
	}
	defer wipe(keyContents)

	key, err := minisign.DecodePrivateKey(string(keyContents))
	if err != nil {
		return fmt.Errorf("private key: %w", err)
	}
	defer key.Wipe()
	if !key.IsEncrypted() {
		return errors.New("private key must be passphrase-protected")
	}
	if err := key.Decrypt(string(password)); err != nil {
		return fmt.Errorf("private key decryption failed")
	}

	signature, err := key.SignFile(resolvedArtifactPath, minisign.SignOptions{Hashed: true})
	if err != nil {
		return fmt.Errorf("sign release artifact: %w", err)
	}
	if err := writeSignatureAtomically(*signaturePath, signature.Encode()); err != nil {
		return fmt.Errorf("write detached signature: %w", err)
	}
	return nil
}

func verify(args []string) error {
	flags := flag.NewFlagSet("verify-minisign-asset", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	artifactPath := flags.String("artifact", "", "path to the release artifact")
	signaturePath := flags.String("signature", "", "path to the detached signature")
	publicKeyPath := flags.String("public-key", "", "path to the trusted public key")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *artifactPath == "" || *signaturePath == "" || *publicKeyPath == "" {
		return errors.New("artifact, signature, and public-key paths are required")
	}
	for label, path := range map[string]string{
		"release artifact": *artifactPath,
		"signature":        *signaturePath,
		"public key":       *publicKeyPath,
	} {
		if err := validateRegularFile(path); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}

	publicKey, err := minisign.NewPublicKeyFromFile(*publicKeyPath)
	if err != nil {
		return fmt.Errorf("public key: %w", err)
	}
	signature, err := minisign.NewSignatureFromFile(*signaturePath)
	if err != nil {
		return fmt.Errorf("signature: %w", err)
	}
	verified, err := publicKey.VerifyFromFile(*artifactPath, signature)
	if err != nil {
		return fmt.Errorf("signature verification failed")
	}
	if !verified {
		return errors.New("signature verification failed")
	}
	return nil
}

func resolveArtifactPath(artifactPath, signaturePath string) (string, error) {
	if err := validateRegularFile(artifactPath); err == nil {
		return artifactPath, nil
	} else if filepath.Base(artifactPath) != artifactPath || !strings.HasSuffix(signaturePath, ".minisig") {
		return "", err
	}

	derivedPath := strings.TrimSuffix(signaturePath, ".minisig")
	if filepath.Base(derivedPath) != artifactPath {
		return "", fmt.Errorf("artifact name does not match signature path")
	}
	if err := validateRegularFile(derivedPath); err != nil {
		return "", err
	}
	return derivedPath, nil
}

func validateRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("must be a regular file")
	}
	return nil
}

func readProtectedFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("must not be accessible by group or other users")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return nil, errors.New("must not be empty")
	}
	return contents, nil
}

func writeSignatureAtomically(path string, contents []byte) error {
	directory := filepath.Dir(path)
	if err := validateDirectory(directory); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".minisig-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return syncDirectory(directory)
}

func validateDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("signature directory is not a directory")
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}

func wipe(contents []byte) {
	for i := range contents {
		contents[i] = 0
	}
}
