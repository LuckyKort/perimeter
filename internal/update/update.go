// Package update checks GitHub for a newer Perimeter and installs it on request.
//
// Nothing is installed silently: the menu offers the update and the user clicks.
// The downloaded file is checked against the release's SHA256SUMS.txt.
package update

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"perimeter/internal/app"
	"perimeter/internal/netwatch"
)

// ErrDisabled: this build has no update source (app.Repo is empty), so it
// never downloads anything, the original app included.
var ErrDisabled = errors.New("updates are off in this build")

// Release is the part of GitHub's release JSON that matters here.
type Release struct {
	Tag    string `json:"tag_name"`
	Page   string `json:"html_url"`
	Body   string `json:"body"`
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func (r Release) Version() string { return strings.TrimPrefix(r.Tag, "v") }

func (r Release) asset(name string) string {
	for _, a := range r.Assets {
		if a.Name == name {
			return a.URL
		}
	}
	return ""
}

// parse turns "1.2.3" into numbers; anything else (a dev build) is not a version.
func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Newer reports whether latest is a later version than current.
// Dev builds ("dev", a git hash) never get update offers.
func Newer(current, latest string) bool {
	c, ok1 := parse(current)
	l, ok2 := parse(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := range c {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// AssetName is the release file for this OS and CPU.
func AssetName(goos, goarch string) string {
	switch goos {
	case "darwin":
		return "Perimeter-macos.zip"
	case "windows":
		return "perimeter-windows-" + goarch + ".exe"
	}
	return "perimeter-linux-" + goarch + ".tar.gz"
}

// Latest asks GitHub for the newest release.
func Latest(ctx context.Context) (Release, error) {
	var r Release
	if app.Repo == "" {
		return r, ErrDisabled
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/"+app.Repo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := netwatch.HTTPClient(15 * time.Second).Do(req)
	if err != nil {
		return r, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return r, fmt.Errorf("GitHub: %s", resp.Status)
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r)
	return r, err
}

// sumFor finds a file's hash in SHA256SUMS.txt ("<hash>  <name>" per line).
func sumFor(sums, name string) string {
	sc := bufio.NewScanner(strings.NewReader(sums))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return strings.ToLower(f[0])
		}
	}
	return ""
}

// Progress is told how many bytes of the update are downloaded so far, and of how many (0 if unknown).
type Progress func(done, total int64)

type counter struct {
	done, total int64
	report      Progress
}

func (c *counter) Write(p []byte) (int, error) {
	c.done += int64(len(p))
	if c.report != nil {
		c.report(c.done, c.total)
	}
	return len(p), nil
}

func download(ctx context.Context, url, dst string, progress Progress) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := netwatch.HTTPClient(5 * time.Minute).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	body := io.Reader(io.LimitReader(resp.Body, 200<<20))
	if progress != nil {
		body = io.TeeReader(body, &counter{total: max(resp.ContentLength, 0), report: progress})
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func fileSum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var ErrChecksum = errors.New("контрольная сумма не совпала: файл повреждён или подменён, обновление не установлено")

// fetch downloads this platform's file into dir and verifies it.
func fetch(ctx context.Context, r Release, dir string, progress Progress) (string, error) {
	name := AssetName(runtime.GOOS, runtime.GOARCH)
	url, sumsURL := r.asset(name), r.asset("SHA256SUMS.txt")
	if url == "" || sumsURL == "" {
		return "", fmt.Errorf("в релизе %s нет файла %s", r.Tag, name)
	}
	sumsPath := filepath.Join(dir, "SHA256SUMS.txt")
	if err := download(ctx, sumsURL, sumsPath, nil); err != nil {
		return "", err
	}
	sums, _ := os.ReadFile(sumsPath)
	want := sumFor(string(sums), name)
	path := filepath.Join(dir, name)
	if err := download(ctx, url, path, progress); err != nil {
		return "", err
	}
	got, err := fileSum(path)
	if err != nil {
		return "", err
	}
	if want == "" || got != want {
		return "", ErrChecksum
	}
	return path, nil
}

// Install downloads, verifies and puts the new version in place of the running one.
// It returns a function that starts the new version; call it right before exiting.
// progress may be nil.
func Install(ctx context.Context, r Release, progress Progress) (restart func() error, err error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	exe, _ = filepath.EvalSymlinks(exe)
	dir, err := os.MkdirTemp("", "perimeter-update-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	file, err := fetch(ctx, r, dir, progress)
	if err != nil {
		return nil, err
	}
	return replace(exe, file, dir)
}
