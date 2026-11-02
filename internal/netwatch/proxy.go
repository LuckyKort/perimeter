package netwatch

import (
	"bufio"
	"net/url"
	"strings"
)

// The public IP has to be measured the way the protected apps reach the internet.
// Desktop apps (Electron, Chromium) use the system proxy, and many VPN clients
// (v2rayN, Hiddify, Clash) work exactly as a system proxy, switched on and off
// at runtime. Go doesn't read the system proxy, so it is read here on every check.

// parseScutil reads `scutil --proxy` output (macOS).
func parseScutil(out string) *url.URL {
	kv := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), " : ")
		if ok {
			kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	if kv["HTTPSEnable"] == "1" && kv["HTTPSProxy"] != "" {
		return &url.URL{Scheme: "http", Host: hostPort(kv["HTTPSProxy"], kv["HTTPSPort"], "443")}
	}
	if kv["SOCKSEnable"] == "1" && kv["SOCKSProxy"] != "" {
		return &url.URL{Scheme: "socks5", Host: hostPort(kv["SOCKSProxy"], kv["SOCKSPort"], "1080")}
	}
	if kv["HTTPEnable"] == "1" && kv["HTTPProxy"] != "" {
		return &url.URL{Scheme: "http", Host: hostPort(kv["HTTPProxy"], kv["HTTPPort"], "80")}
	}
	return nil
}

// parseWinProxy reads the Internet Settings ProxyServer value (Windows):
// "host:port" for all protocols, or "http=h:p;https=h:p;socks=h:p".
func parseWinProxy(enabled bool, server string) *url.URL {
	server = strings.TrimSpace(server)
	if !enabled || server == "" {
		return nil
	}
	if !strings.Contains(server, "=") {
		if u, err := url.Parse("http://" + strings.TrimPrefix(server, "http://")); err == nil {
			return u
		}
		return nil
	}
	parts := map[string]string{}
	for _, p := range strings.Split(server, ";") {
		if k, v, ok := strings.Cut(p, "="); ok {
			parts[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
		}
	}
	if v := parts["https"]; v != "" {
		return &url.URL{Scheme: "http", Host: v}
	}
	if v := parts["socks"]; v != "" {
		return &url.URL{Scheme: "socks5", Host: v}
	}
	if v := parts["http"]; v != "" {
		return &url.URL{Scheme: "http", Host: v}
	}
	return nil
}

func hostPort(host, port, def string) string {
	if port == "" || port == "0" {
		port = def
	}
	return host + ":" + port
}
