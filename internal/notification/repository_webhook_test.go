package notification_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/notification"
)

// The webhook poster refuses, at connect time, every address that is not public, whatever the URL names: a
// loopback address, and a host name that resolves to one (DNS rebinding: the name was public when the admin saved
// it). The server is never reached.
func TestWebhookPoster_RefusesAddressesThatAreNotPublicAtConnectTime(t *testing.T) {
	var hits int
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	p := notification.NewWebhookPoster(2 * time.Second)
	for _, target := range []string{
		srv.URL + "/hook",                          // 127.0.0.1
		"https://localhost:" + u.Port() + "/hook",  // a name resolving to loopback
		"https://[::1]:" + u.Port() + "/hook",      // IPv6 loopback
		"https://10.255.255.1:" + u.Port(),         // RFC 1918
		"https://169.254.169.254/latest/meta-data", // link-local: a cloud's metadata service
		"https://[fd12::1]:443/",                   // unique-local
	} {
		_, err := p.Post(t.Context(), target, []byte("{}"), nil)
		if err == nil || strings.Contains(err.Error(), "/hook") || strings.Contains(err.Error(), "meta-data") {
			t.Errorf("Post(%s) = %v, want a refusal without the URL", target, err)
		}
		if !strings.Contains(err.Error(), "not public") {
			t.Errorf("Post(%s) = %v, want it refused for its address", target, err)
		}
	}
	if hits != 0 {
		t.Errorf("the loopback server was reached %d times", hits)
	}
}
