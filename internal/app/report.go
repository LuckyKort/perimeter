package app

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
)

var (
	ipv4Re = regexp.MustCompile(`\b\d{1,3}(\.\d{1,3}){3}(/\d+)?\b`)
	ipv6Re = regexp.MustCompile(`\b[0-9a-fA-F]{1,4}(:[0-9a-fA-F]{0,4}){2,7}(/\d+)?\b`)
	macRe  = regexp.MustCompile(`\b[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}\b`)
)

// Mask hides addresses in log lines before they leave the computer.
func Mask(s string) string {
	s = macRe.ReplaceAllString(s, "xx:xx:xx:xx:xx:xx")
	s = ipv6Re.ReplaceAllStringFunc(s, func(m string) string {
		if strings.Count(m, ":") < 2 || regexp.MustCompile(`^\d{2}:\d{2}(:\d{2})?$`).MatchString(m) {
			return m // a time like 14:05:33, not an address
		}
		return "x:x::x"
	})
	return ipv4Re.ReplaceAllString(s, "x.x.x.x")
}

func lastLines(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return lines
}

// ReportURL opens a new GitHub issue with the version, the system and the end of
// the log filled in. IPs and MACs are masked; the user sees everything before sending.
func ReportURL(version string) string {
	log := Mask(strings.Join(lastLines(LogPath(), 40), "\n"))
	if len(log) > 4000 {
		log = log[len(log)-4000:]
	}
	body := fmt.Sprintf("**Что случилось**\n\n<!-- опишите, что делали и что пошло не так -->\n\n"+
		"**Система**\n\nKordon %s · %s/%s\n\n**Журнал** (адреса скрыты)\n\n```\n%s\n```\n",
		version, runtime.GOOS, runtime.GOARCH, log)
	return RepoURL + "/issues/new?" + url.Values{"title": {""}, "body": {body}}.Encode()
}
