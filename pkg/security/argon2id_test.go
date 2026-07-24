package security

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("s3nh4-correta")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ok, err := VerifyPassword(hash, "s3nh4-correta")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("senha correta deveria validar")
	}

	ok, err = VerifyPassword(hash, "senha-errada")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Fatal("senha errada não deveria validar")
	}
}

func TestConstantTimeEquals(t *testing.T) {
	if !ConstantTimeEquals("abc", "abc") {
		t.Fatal("strings iguais deveriam bater")
	}
	if ConstantTimeEquals("abc", "abd") {
		t.Fatal("strings diferentes não deveriam bater")
	}
	if ConstantTimeEquals("abc", "ab") {
		t.Fatal("strings de tamanhos diferentes não deveriam bater")
	}
}
