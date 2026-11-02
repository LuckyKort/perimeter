package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "v1.0.1", true},
		{"1.9.0", "1.10.0", true}, // numbers, not strings
		{"1.1.0", "1.1.0", false},
		{"2.0.0", "1.9.9", false},
		{"dev", "9.9.9", false}, // dev builds don't nag
		{"7fadfec", "1.0.0", false},
		{"1.0.0", "garbage", false},
	}
	for _, c := range cases {
		if got := Newer(c.cur, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v", c.cur, c.latest, got)
		}
	}
}

func TestAssetName(t *testing.T) {
	for _, c := range [][3]string{
		{"darwin", "arm64", "Kordon-macos.zip"},
		{"windows", "amd64", "kordon-windows-amd64.exe"},
		{"windows", "arm64", "kordon-windows-arm64.exe"},
		{"linux", "amd64", "kordon-linux-amd64.tar.gz"},
	} {
		if got := AssetName(c[0], c[1]); got != c[2] {
			t.Errorf("%s/%s -> %s", c[0], c[1], got)
		}
	}
}

func TestSumFor(t *testing.T) {
	sums := "aaa  Kordon-macos.zip\nBBB  kordon-windows-amd64.exe\nccc *kordon-linux-amd64.tar.gz\n"
	if sumFor(sums, "kordon-windows-amd64.exe") != "bbb" || sumFor(sums, "kordon-linux-amd64.tar.gz") != "ccc" {
		t.Fatal("sum not found")
	}
	if sumFor(sums, "Kordon-macos") != "" {
		t.Fatal("partial name matched")
	}
}

// TestFetchVerifies serves a fake release and checks that a tampered file is refused.
func TestFetchVerifies(t *testing.T) {
	name := AssetName(runtime.GOOS, runtime.GOARCH)
	body := []byte("the real build")
	sum := sha256.Sum256(body)
	tampered := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/SHA256SUMS.txt":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		case "/" + name:
			if tampered {
				w.Write([]byte("evil build"))
			} else {
				w.Write(body)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	rel := Release{Tag: "v9.9.9"}
	rel.Assets = append(rel.Assets,
		struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		}{name, srv.URL + "/" + name},
		struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		}{"SHA256SUMS.txt", srv.URL + "/SHA256SUMS.txt"})

	if _, err := fetch(context.Background(), rel, t.TempDir()); err != nil {
		t.Fatalf("good file refused: %v", err)
	}
	tampered = true
	if _, err := fetch(context.Background(), rel, t.TempDir()); !errors.Is(err, ErrChecksum) {
		t.Fatalf("tampered file accepted: %v", err)
	}
	if _, err := fetch(context.Background(), Release{Tag: "v1"}, t.TempDir()); err == nil {
		t.Fatal("release without the file accepted")
	}
}
