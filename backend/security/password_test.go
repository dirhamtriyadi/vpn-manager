package security

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "s3cret-pass" || len(hash) < 40 {
		t.Fatalf("hash looks wrong: %q", hash)
	}
	ok, err := VerifyPassword(hash, "s3cret-pass")
	if err != nil || !ok {
		t.Fatalf("expected match, got ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(hash, "wrong")
	if err != nil {
		t.Fatalf("unexpected verify error: %v", err)
	}
	if ok {
		t.Fatal("expected mismatch for wrong password")
	}
}

func TestHashesAreSaltedUniquely(t *testing.T) {
	a, _ := HashPassword("same")
	b, _ := HashPassword("same")
	if a == b {
		t.Fatal("two hashes of the same password must differ (random salt)")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	if _, err := VerifyPassword("not-a-phc-hash", "x"); err == nil {
		t.Fatal("expected error for malformed hash")
	}
}

func TestHashRejectsEmptyPassword(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("expected error for empty password")
	}
}
