package netwatch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEachServiceKeepsEveryAnswer(t *testing.T) {
	mk := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, body) }))
	}
	a, b, bad := mk("64.137.54.87"), mk("95.24.1.1"), mk("<html>")
	defer a.Close()
	defer b.Close()
	defer bad.Close()
	got := EachService(context.Background(), []string{a.URL, b.URL, bad.URL}, 3*time.Second, false, true)
	if len(got) != 3 || got[0].IP != "64.137.54.87" || got[1].IP != "95.24.1.1" || !got[2].Error {
		t.Fatalf("%+v", got)
	}
}
