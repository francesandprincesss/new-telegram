package auth

import "testing"

func TestBcryptHasher(t *testing.T) {
	hasher := NewBcryptHasher(4)
	hash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "correct-password" {
		t.Fatal("Hash() returned plaintext")
	}
	if err := hasher.Compare(hash, "correct-password"); err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if err := hasher.Compare(hash, "wrong-password"); err == nil {
		t.Fatal("Compare() accepted a wrong password")
	}
}

func TestBcryptHasherRejectsInvalidLengths(t *testing.T) {
	hasher := NewBcryptHasher(4)
	for _, password := range []string{"short", string(make([]byte, 73))} {
		if _, err := hasher.Hash(password); err == nil {
			t.Fatalf("Hash(%d bytes) expected an error", len(password))
		}
	}
}
