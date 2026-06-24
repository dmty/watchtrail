package auth

import (
	"crypto/subtle"
	"net/http"

	"watchtrail/internal/accountctx"
)

// CookieName is the dashboard auth cookie. Its value is HexKey(key).
const CookieName = "wt_auth"

// SetupQueryParam is the query key carrying the magic-link secret.
const SetupQueryParam = "setup"

const cookieMaxAge = 60 * 60 * 24 * 365 // 1 year

// Middleware returns an http middleware that gates requests on a cookie
// matching key, honors the setup-link query parameter to issue that cookie,
// applies a Sec-Fetch-Site CSRF guard on state-changing methods, and injects
// the default accountID into the downstream context.
func Middleware(key []byte) func(http.Handler) http.Handler {
	expected := []byte(HexKey(key))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if setup := q.Get(SetupQueryParam); setup != "" {
				candidate := []byte(setup)
				if len(candidate) == len(expected) && subtle.ConstantTimeCompare(candidate, expected) == 1 {
					http.SetCookie(w, &http.Cookie{
						Name:     CookieName,
						Value:    HexKey(key),
						Path:     "/",
						HttpOnly: true,
						Secure:   r.TLS != nil,
						SameSite: http.SameSiteStrictMode,
						MaxAge:   cookieMaxAge,
					})
					q.Del(SetupQueryParam)
					u := *r.URL
					u.RawQuery = q.Encode()
					http.Redirect(w, r, u.RequestURI(), http.StatusSeeOther)
					return
				}
				// fall through: a wrong ?setup= must not nuke a valid cookie.
			}

			c, err := r.Cookie(CookieName)
			if err != nil ||
				len(c.Value) != len(expected) ||
				subtle.ConstantTimeCompare([]byte(c.Value), expected) != 1 {
				unauthorized(w)
				return
			}

			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
					http.Error(w, "forbidden: cross-site request", http.StatusForbidden)
					return
				}
			}

			ctx := accountctx.With(r.Context(), accountctx.Default)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("unauthorized: open watchtrail via the app menu or run \"watchtrail print-link\""))
}
