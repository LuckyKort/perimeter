package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Notify(title, body string) {
	// AppleScript strings: escape backslashes and quotes.
	q := func(s string) string {
		return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
	}
	_ = exec.Command("/usr/bin/osascript", "-e",
		fmt.Sprintf("display notification %s with title %s", q(body), q(title))).Start()
}

func OpenFile(path string) error { return exec.Command("/usr/bin/open", "-t", path).Start() }
func OpenURL(u string) error     { return exec.Command("/usr/bin/open", u).Start() }
func Reveal(path string) error   { return exec.Command("/usr/bin/open", "-R", path).Start() }

func SystemLanguage() string {
	out, _ := exec.Command("/usr/bin/defaults", "read", "-g", "AppleLanguages").Output()
	if l := firstLang(string(out)); l != "" {
		return l
	}
	return langFromEnv()
}

// firstLang picks the first preferred language from the `defaults` array output.
func firstLang(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.Trim(strings.TrimSpace(line), `",()`)
		if line != "" {
			return strings.ToLower(line[:min(2, len(line))])
		}
	}
	return ""
}

const agentLabel = "dev.perimeter.app"

func agentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", agentLabel+".plist")
}

func AutostartEnabled() bool {
	_, err := os.Stat(agentPath())
	return err == nil
}

// SetAutostart writes a LaunchAgent that starts Perimeter at login and restarts it
// after a crash (but not after "Quit", which exits with 0).
func SetAutostart(on bool) error {
	if !on {
		return os.Remove(agentPath())
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key><array><string>%s</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>ProcessType</key><string>Interactive</string>
</dict>
</plist>
`, agentLabel, xmlEscape(exe))
	if err := os.MkdirAll(filepath.Dir(agentPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(agentPath(), []byte(plist), 0o644)
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
