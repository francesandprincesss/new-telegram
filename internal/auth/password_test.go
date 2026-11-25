package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestArgon2idHasher(t *testing.T) {
	hasher := Argon2idHasher{}
	password := strings.Repeat("🙂", 72)
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := hasher.Hash(password)
	if err != nil || second == hash {
		t.Fatalf("new salt expected on each hash, error = %v", err)
	}
	if err := hasher.Compare(hash, password); err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	// Only the last character differs, beyond bcrypt's 72-byte input limit.
	if err := hasher.Compare(hash, strings.Repeat("🙂", 71)+"🚀"); err == nil {
		t.Fatal("Compare() ignored the end of the password")
	}
	if err := hasher.Compare(argonPrefix+"invalid$hash", password); err == nil {
		t.Fatal("Compare() accepted a malformed hash")
	}
}

func TestRegistrationPasswordLength(t *testing.T) {
	for _, password := range []string{strings.Repeat("🙂", 7), strings.Repeat("я", 73), "invalid\xff"} {
		if _, err := (Argon2idHasher{}).Hash(password); err == nil {
			t.Fatal("accepted an invalid registration password")
		}
	}
	if err := validatePassword(strings.Repeat("я", 8)); err != nil {
		t.Fatalf("8 Unicode characters rejected: %v", err)
	}
}

func TestLegacyBcryptDoesNotAcceptTrailingBytes(t *testing.T) {
	password := strings.Repeat("я", 36)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := (Argon2idHasher{}).Compare(string(hash), password+"a"); err == nil {
		t.Fatal("Compare() accepted a password with trailing bytes")
	}
}
