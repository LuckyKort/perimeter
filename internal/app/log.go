package app

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const maxLog = 1 << 20 // 1 MB, then the old half goes to kordon.log.1

var logMu sync.Mutex

// Logf appends a line to kordon.log.
func Logf(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	p := LogPath()
	if st, err := os.Stat(p); err == nil && st.Size() > maxLog {
		_ = os.Rename(p, p+".1")
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}
