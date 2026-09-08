package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// AuthHandler exposes a login check for the shared app password. The actual
// gate is the requireAuth middleware in router.go — this handler exists only
// so the frontend gets a clear yes/no before it starts storing the password.
type AuthHandler struct {
	password string
}

func NewAuthHandler(password string) *AuthHandler {
	return &AuthHandler{password: password}
}

type loginRequest struct {
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// No password configured means the app is running open (local dev) —
	// any submitted password is accepted so the frontend's login flow still
	// works without extra setup.
	if h.password == "" {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !passwordsMatch(req.Password, h.password) {
		writeError(w, http.StatusUnauthorized, "sai mật khẩu")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func passwordsMatch(a, b string) bool {
	// Constant-time to avoid leaking the password length/contents through
	// response-timing side channels.
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// requireAuth rejects any request whose "Authorization: Bearer <password>"
// header doesn't match the configured app password. A blank configured
// password disables the check entirely (local dev).
func requireAuth(password string, next http.Handler) http.Handler {
	if password == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !passwordsMatch(got, password) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
