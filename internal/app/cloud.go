package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ef1m03/kordon/internal/netwatch"
)

// CloudURL is the small Efimov Dev service behind the user count and the
// Telegram alerts; its code is in server/ of the repository.
const CloudURL = "https://efimovdev.ru/kordon/api"

var errCloud = errors.New("the Kordon service didn't answer")

func cloudCall(ctx context.Context, method, path, secret string, body any, out any) error {
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, CloudURL+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Kordon-Secret", secret)
	}
	resp, err := netwatch.HTTPClient(15 * time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errCloud
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// ---- the user count ----

func pingDayPath() string { return filepath.Join(Dir(), "ping-day") }

// releaseVersion: dev builds aren't counted.
func releaseVersion(v string) bool {
	return v != "" && strings.Trim(v, "0123456789.") == ""
}

// PingLoop counts this Kordon among its users once a day, while enabled says
// so (usage_ping: on). It sends the version and the OS, nothing else: no id,
// and the service keeps no addresses.
func PingLoop(ctx context.Context, version string, enabled func() bool) {
	if !releaseVersion(version) {
		return
	}
	for {
		day := time.Now().UTC().Format("2006-01-02")
		last, _ := os.ReadFile(pingDayPath())
		if enabled() && strings.TrimSpace(string(last)) != day {
			c, cancel := context.WithTimeout(ctx, 20*time.Second)
			if cloudCall(c, "POST", "/ping", "", map[string]string{"v": version, "os": runtime.GOOS}, nil) == nil {
				_ = os.WriteFile(pingDayPath(), []byte(day), 0o644)
			}
			cancel()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute): // cheap: nothing is sent until the day changes
		}
	}
}

// ---- Telegram alerts ----

// Telegram links this Kordon to a chat with the Kordon bot. The secret lives
// only here (telegram.json, readable by the user only); the service keeps its
// hash and the chat id.
type Telegram struct {
	mu     sync.Mutex
	Secret string `json:"secret"`
	Linked bool   `json:"linked"`
	queue  chan string
}

var TG = loadTelegram()

func telegramPath() string { return filepath.Join(Dir(), "telegram.json") }

func loadTelegram() *Telegram {
	t := &Telegram{queue: make(chan string, 20)}
	if b, err := os.ReadFile(telegramPath()); err == nil {
		_ = json.Unmarshal(b, t)
	}
	go t.sender()
	return t
}

func (t *Telegram) save() {
	b, _ := json.Marshal(t)
	_ = os.WriteFile(telegramPath(), b, 0o600)
}

// IsLinked reports whether alerts go to a chat.
func (t *Telegram) IsLinked() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Linked
}

// Link starts linking and returns the t.me link that finishes it in Telegram.
func (t *Telegram) Link(ctx context.Context) (string, error) {
	t.mu.Lock()
	if t.Secret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		t.Secret = hex.EncodeToString(b)
		t.save()
	}
	secret := t.Secret
	t.mu.Unlock()
	var r struct {
		URL string `json:"url"`
	}
	if err := cloudCall(ctx, "POST", "/tg/link", secret, map[string]string{}, &r); err != nil {
		return "", err
	}
	if !strings.HasPrefix(r.URL, "https://t.me/") {
		return "", errCloud
	}
	return r.URL, nil
}

// Refresh asks the service whether the chat is linked (after Link, or to notice /stop).
func (t *Telegram) Refresh(ctx context.Context) bool {
	t.mu.Lock()
	secret := t.Secret
	t.mu.Unlock()
	if secret == "" {
		return false
	}
	var r struct {
		Linked bool `json:"linked"`
	}
	if err := cloudCall(ctx, "GET", "/tg/status", secret, nil, &r); err != nil {
		return t.IsLinked()
	}
	t.mu.Lock()
	if t.Linked != r.Linked {
		t.Linked = r.Linked
		t.save()
	}
	t.mu.Unlock()
	return r.Linked
}

// Unlink stops the alerts; a new link gets a new secret.
func (t *Telegram) Unlink(ctx context.Context) {
	t.mu.Lock()
	secret := t.Secret
	t.Secret, t.Linked = "", false
	t.save()
	t.mu.Unlock()
	if secret != "" {
		_ = cloudCall(ctx, "POST", "/tg/unlink", secret, map[string]string{}, nil)
	}
}

// Send queues an alert; it never blocks the caller, and a full queue drops it.
func (t *Telegram) Send(text string) {
	if !t.IsLinked() {
		return
	}
	select {
	case t.queue <- text:
	default:
	}
}

func (t *Telegram) sender() {
	for text := range t.queue {
		t.mu.Lock()
		secret := t.Secret
		t.mu.Unlock()
		for try := 0; try < 3 && secret != ""; try++ {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			err := cloudCall(ctx, "POST", "/tg/send", secret, map[string]string{"text": text}, nil)
			cancel()
			if err == nil {
				break
			}
			time.Sleep(time.Duration(try+1) * 10 * time.Second) // the VPN may still be reconnecting
		}
	}
}
