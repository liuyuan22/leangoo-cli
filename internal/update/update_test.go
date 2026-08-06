package update

import "testing"

func TestAssetNameFor(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"darwin", "arm64", "leangoo_Darwin_arm64.tar.gz"},
		{"darwin", "amd64", "leangoo_Darwin_x86_64.tar.gz"},
		{"linux", "amd64", "leangoo_Linux_x86_64.tar.gz"},
		{"windows", "amd64", "leangoo_Windows_x86_64.zip"},
	}
	for _, c := range cases {
		got, err := assetNameFor(c.goos, c.goarch)
		if err != nil {
			t.Fatalf("%s/%s: %v", c.goos, c.goarch, err)
		}
		if got != c.want {
			t.Fatalf("%s/%s: got %s want %s", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestSafeJoin(t *testing.T) {
	got, err := safeJoin("/tmp/out", "leangoo_Darwin_arm64/leangoo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/out/leangoo_Darwin_arm64/leangoo" {
		t.Fatalf("got %s", got)
	}
	if _, err := safeJoin("/tmp/out", "../etc/passwd"); err == nil {
		t.Fatal("expected error")
	}
}
