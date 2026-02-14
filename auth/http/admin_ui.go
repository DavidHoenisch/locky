package http

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
)

const adminUICookieName = "locky_admin_ui_session"

func (s *Server) handleAdminUILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectLoginWithError(w, r, "Invalid login form")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	if username != s.config.AdminUIUsername || password != s.config.AdminUIPassword {
		redirectLoginWithError(w, r, "Invalid username or password")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     adminUICookieName,
		Value:    s.adminUISessionToken(),
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   8 * 60 * 60,
	})
	http.Redirect(w, r, "/admin/ui", http.StatusFound)
}

func redirectLoginWithError(w http.ResponseWriter, r *http.Request, errMsg string) {
	u, _ := url.Parse("/admin/ui/login")
	u.RawQuery = url.Values{"error": {errMsg}}.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) handleAdminUILogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminUICookieName,
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     adminUICookieName,
		Value:    "",
		Path:     "/admin/ui",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/admin/ui", http.StatusFound)
}

func (s *Server) isAdminUIAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(adminUICookieName)
	if err != nil {
		return false
	}
	return cookie.Value == s.adminUISessionToken()
}

func (s *Server) adminUISessionToken() string {
	sum := sha256.Sum256([]byte(s.config.AdminUIUsername + ":" + s.config.AdminUIPassword + ":" + s.config.AdminAPIKey))
	return hex.EncodeToString(sum[:])
}
