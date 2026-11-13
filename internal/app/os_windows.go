package app

import (
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const createNoWindow = 0x08000000

func hidden(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return cmd
}

// Notify shows a Windows toast through PowerShell's own app id, which every
// Windows 10/11 has registered; an unsigned app can't register its own.
func Notify(title, body string) {
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
$x = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$t = $x.GetElementsByTagName('text')
$t.Item(0).AppendChild($x.CreateTextNode(` + q(title) + `)) > $null
$t.Item(1).AppendChild($x.CreateTextNode(` + q(body) + `)) > $null
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe').Show([Windows.UI.Notifications.ToastNotification]::new($x))`
	// -EncodedCommand takes UTF-16LE base64: no quoting problems, Cyrillic survives.
	u := utf16.Encode([]rune(script))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		b[2*i], b[2*i+1] = byte(c), byte(c>>8)
	}
	_ = hidden(exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand",
		base64.StdEncoding.EncodeToString(b))).Start()
}

// GUI programs are started without hidden(): its SW_HIDE would hide their window too.
func OpenFile(path string) error { return exec.Command("notepad.exe", path).Start() }
func OpenURL(u string) error {
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", u).Start()
}
func Reveal(path string) error { return exec.Command("explorer.exe", "/select,", path).Start() }

func SystemLanguage() string {
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err == nil && len(langs) > 0 && len(langs[0]) >= 2 {
		return strings.ToLower(langs[0][:2])
	}
	return langFromEnv()
}

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func AutostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue("Perimeter")
	return err == nil
}

func SetAutostart(on bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		return k.DeleteValue("Perimeter")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue("Perimeter", `"`+exe+`"`)
}
