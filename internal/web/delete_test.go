// internal/web/delete_test.go
package web

import (
	"net/http"
	"testing"
	"time"

	"watchtrail/internal/store"
)

// postNoRedirect issues a POST without following redirects, so the handler's own
// status (303 / 200) is observable. htmx toggles the HX-Request header.
func postNoRedirect(t *testing.T, url string, htmx bool) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("POST", url, nil)
	if htmx {
		req.Header.Set("HX-Request", "true")
	}
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestDeleteItemHTMXRedirectsHome(t *testing.T) {
	base := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "mX", "The Film", "vlc", base, 100, true)
	})
	resp := postNoRedirect(t, srv.URL+"/item/mX/delete", true)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("HX-Redirect"); got != "/" {
		t.Fatalf("HX-Redirect = %q want /", got)
	}
	if status, _ := bodyOf(t, srv.URL+"/item/mX", false); status != 404 {
		t.Fatalf("item after delete status %d want 404", status)
	}
}

func TestDeleteItemPlainRedirects303(t *testing.T) {
	base := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	srv := newWebServer(t, func(r *store.SQLiteRepo) {
		seedWebSession(t, r, "s1", "mX", "The Film", "vlc", base, 100, true)
	})
	resp := postNoRedirect(t, srv.URL+"/item/mX/delete", false)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status %d want 303", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "/" {
		t.Fatalf("Location = %q want /", got)
	}
}

func TestDeleteItemUnknown404(t *testing.T) {
	srv := newWebServer(t, nil)
	resp := postNoRedirect(t, srv.URL+"/item/missing/delete", false)
	if resp.StatusCode != 404 {
		t.Fatalf("status %d want 404", resp.StatusCode)
	}
}
