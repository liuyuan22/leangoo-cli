package auth_test

import (
	"testing"

	"github.com/deepglint/leangoo-cli/internal/auth"
)

func TestDoubleMD5(t *testing.T) {
	// md5(md5("test"))
	got := auth.DoubleMD5("test")
	if len(got) != 32 {
		t.Fatalf("len=%d", len(got))
	}
	if auth.DoubleMD5(got) != got {
		t.Fatal("already-hashed password should not be rehashed")
	}
}
