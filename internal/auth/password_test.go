package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword(hash, "correct-horse-battery-staple") {
		t.Error("expected correct password to check out")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Error("expected wrong password to fail")
	}
}
