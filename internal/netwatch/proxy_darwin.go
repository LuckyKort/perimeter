package netwatch

import (
	"net/url"
	"os/exec"
)

func SystemProxy() *url.URL {
	out, err := exec.Command("/usr/sbin/scutil", "--proxy").Output()
	if err != nil {
		return nil
	}
	return parseScutil(string(out))
}
