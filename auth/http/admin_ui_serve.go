package http

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/locky/auth/ui"
)

// serveAdminUIRoutes handles GET /admin/ui, /admin/ui/, /admin/ui/login, and /admin/ui/*.
// It checks auth for the dashboard paths and serves the embedded Svelte app.
func (s *Server) serveAdminUIRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/admin/ui" || path == "/admin/ui/" {
		if !s.isAdminUIAuthenticated(r) {
			http.Redirect(w, r, "/admin/ui/login", http.StatusFound)
			return
		}
		// Refresh session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     adminUICookieName,
			Value:    s.adminUISessionToken(),
			Path:     "/admin",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false,
			MaxAge:   8 * 60 * 60,
		})
	}

	// SPA routes (index.html); only /admin/ui/assets/* are real files
	servePath := "/"
	if strings.HasPrefix(path, "/admin/ui/assets/") {
		servePath = "/" + strings.TrimPrefix(path[len("/admin/ui"):], "/")
	}
	// else: /admin/ui, /admin/ui/, /admin/ui/login, etc. all get index.html
	s.serveAdminUIApp(w, r, servePath)
}

func (s *Server) serveAdminUIApp(w http.ResponseWriter, r *http.Request, servePath string) {
	h := ui.Handler()
	if h == nil {
		http.Error(w, "Admin UI not available (build without embed_ui)", http.StatusNotFound)
		return
	}
	newReq := r.Clone(r.Context())
	if newReq.URL == nil {
		newReq.URL = r.URL
	}
	newReq.URL = cloneURL(r.URL)
	newReq.URL.Path = servePath
	newReq.URL.RawPath = ""
	h.ServeHTTP(w, newReq)
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	u2 := *u
	return &u2
}
