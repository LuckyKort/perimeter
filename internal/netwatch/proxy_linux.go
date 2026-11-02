package netwatch

import (
	"net/http"
	"net/url"
)

// On Linux there is no single system proxy; the usual environment variables are honoured.
func SystemProxy() *url.URL {
	req, _ := http.NewRequest("GET", "https://api.ipify.org", nil)
	u, _ := http.ProxyFromEnvironment(req)
	return u
}
