package app

import (
	"os"
	"strings"
)

func langFromEnv() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); len(v) >= 2 {
			return strings.ToLower(v[:2])
		}
	}
	return "en"
}

// Language resolves the config value ("auto", "ru", "en") to "ru" or "en".
func Language(setting string) string {
	l := setting
	if l == "auto" || l == "" {
		l = SystemLanguage()
	}
	switch l {
	case "ru", "uk", "be", "kk":
		return "ru"
	}
	return "en"
}
