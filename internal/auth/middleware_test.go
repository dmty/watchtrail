package auth

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"watchtrail/internal/accountctx"
)

func newKey(t *testing.T) []byte {
	t.Helper()
	dir := t.TempDir()
	k, _, err := LoadOrCreateKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

type capturingHandler struct{ ctx context.Context }

func (c *capturingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.ctx = r.Context()
	w.WriteHeader(http.StatusNoContent)
}

func TestMiddleware_NoCookie_Returns401(t *testing.T) {
	key := newKey(t)
	mw := Middleware(key)
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next called unexpectedly")
	})).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMiddleware_ValidCookie_PassesThroughAndInjectsAccount(t *testing.T) {
	key := newKey(t)
	cap := &capturingHandler{}
	mw := Middleware(key)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: HexKey(key)})
	rec := httptest.NewRecorder()
	mw(cap).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := accountctx.From(cap.ctx); got != accountctx.Default {
		t.Errorf("accountID = %q, want %q", got, accountctx.Default)
	}
}

func TestMiddleware_SetupLink_SetsCookieAndRedirects(t *testing.T) {
	key := newKey(t)
	mw := Middleware(key)
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next called on setup link visit")
	})).ServeHTTP(rec, httptest.NewRequest("GET", "/?setup="+HexKey(key)+"&keep=1", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if strings.Contains(loc, "setup=") {
		t.Errorf("Location still carries setup token: %q", loc)
	}
	if !strings.Contains(loc, "keep=1") {
		t.Errorf("Location dropped sibling query param: %q", loc)
	}
	var got *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == CookieName {
			got = c
		}
	}
	if got == nil {
		t.Fatalf("cookie %q not set", CookieName)
	}
	if got.Value != HexKey(key) {
		t.Errorf("cookie value = %q, want %q", got.Value, HexKey(key))
	}
	if !got.HttpOnly || got.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie not hardened: %+v", got)
	}
	if got.Path != "/" {
		t.Errorf("cookie Path = %q, want %q", got.Path, "/")
	}
	if got.MaxAge != cookieMaxAge {
		t.Errorf("cookie MaxAge = %d, want %d", got.MaxAge, cookieMaxAge)
	}
}

func TestMiddleware_WrongSetup_FallsThroughTo401(t *testing.T) {
	key := newKey(t)
	mw := Middleware(key)
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next called on bad setup")
	})).ServeHTTP(rec, httptest.NewRequest("GET", "/?setup=DEADBEEF", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMiddleware_CSRF_CrossSitePOSTRejected(t *testing.T) {
	key := newKey(t)
	mw := Middleware(key)
	req := httptest.NewRequest("POST", "/item/x/delete", strings.NewReader(""))
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.AddCookie(&http.Cookie{Name: CookieName, Value: HexKey(key)})
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next called on cross-site POST")
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestMiddleware_CSRF_SameOriginPOSTAllowed(t *testing.T) {
	key := newKey(t)
	cap := &capturingHandler{}
	mw := Middleware(key)
	req := httptest.NewRequest("POST", "/item/x/delete", strings.NewReader(""))
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: CookieName, Value: HexKey(key)})
	rec := httptest.NewRecorder()
	mw(cap).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}

func TestMiddleware_CSRF_MissingSecFetchSiteAllowed(t *testing.T) {
	key := newKey(t)
	cap := &capturingHandler{}
	mw := Middleware(key)
	req := httptest.NewRequest("POST", "/item/x/delete", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: CookieName, Value: HexKey(key)})
	rec := httptest.NewRecorder()
	mw(cap).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}

func TestMiddleware_BadCookieRejected(t *testing.T) {
	key := newKey(t)
	mw := Middleware(key)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "deadbeef"})
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next called with bad cookie")
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	body, _ := io.ReadAll(rec.Result().Body)
	if !strings.Contains(string(body), "watchtrail print-link") {
		t.Errorf("body = %q, want hint", string(body))
	}
}

func TestSetupCookieSecureOverTLS(t *testing.T) {
	key := make([]byte, 32)
	mw := Middleware(key)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "https://watchtrail.local:8443/?setup="+HexKey(key), nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if !cookies[0].Secure {
		t.Fatal("cookie set over HTTPS must have Secure=true")
	}
}

func TestSetupCookieNotSecureOverHTTP(t *testing.T) {
	key := make([]byte, 32)
	mw := Middleware(key)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "http://watchtrail.local/?setup="+HexKey(key), nil)
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Secure {
		t.Fatal("cookie set over HTTP must not have Secure=true")
	}
}
