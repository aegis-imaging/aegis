// gate-server: thin HTTP server for the AEGIS landing page.
// When GATE_ENABLED=true it requires a valid HMAC-signed cookie before
// serving any page HTML. Cookie is set after a successful server-side
// invite code validation — the raw code never reaches client JavaScript.
package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed gate.html
var gateHTML []byte

var (
	gateEnabled  = os.Getenv("GATE_ENABLED") == "true"
	gateSecret   = os.Getenv("GATE_SECRET")
	apiBase      = envOr("API_BASE_URL", "https://api.aegisimaging.ai")
	staticDir    = envOr("STATIC_DIR", "/app/dist")
	cookieName   = "aegis_gate"
	cookieMaxAge = 30 * 24 * 60 * 60 // 30 days
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := envOr("PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/gate/validate", handleValidate)
	mux.HandleFunc("POST /api/gate/revoke", handleRevoke)
	mux.HandleFunc("POST /api/gate/request-access", handleRequestAccess)
	mux.HandleFunc("/", handleRoot)

	log.Printf("gate-server listening on :%s (gate_enabled=%v, static=%s)", port, gateEnabled, staticDir)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// handleRoot — all page requests pass through here.
func handleRoot(w http.ResponseWriter, r *http.Request) {
	if !gateEnabled {
		serveStatic(w, r)
		return
	}

	// Server-side ?invite=CODE handling — validate and redirect without JS involvement.
	if code := r.URL.Query().Get("invite"); code != "" {
		valid, err := callValidateAPI(r.Context(), code, clientIP(r))
		if err != nil {
			log.Printf("invite validate api error: %v", err)
		}
		if valid {
			setSessionCookie(w, r)
			// Redirect to the same path, stripping the invite param.
			dest := r.URL.Path
			if dest == "" {
				dest = "/"
			}
			http.Redirect(w, r, dest, http.StatusFound)
			return
		}
		serveGate(w, "Invalid invite code. Please check and try again.")
		return
	}

	if isAdmitted(r) {
		serveStatic(w, r)
		return
	}

	serveGate(w, "")
}

// handleValidate — POST /api/gate/validate {"code":"..."}
// Called by the gate.html form via fetch. Sets cookie on success.
func handleValidate(w http.ResponseWriter, r *http.Request) {
	if !gateEnabled {
		writeJSON(w, http.StatusOK, map[string]bool{"valid": true})
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	valid, err := callValidateAPI(r.Context(), strings.TrimSpace(body.Code), clientIP(r))
	if err != nil {
		log.Printf("validate api error: %v", err)
	}
	if valid {
		setSessionCookie(w, r)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"valid": valid})
}

// handleRequestAccess — POST /api/gate/request-access {"name":"...","email":"...","org":"...","message":"..."}
// Proxies the access request to the Go API's POST /api/invite/request endpoint.
func handleRequestAccess(w http.ResponseWriter, r *http.Request) {
	// Read and forward the body as-is to the Go API.
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "POST", apiBase+"/api/invite/request", &buf)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if ip := clientIP(r); ip != "" {
		req.Header.Set("X-Forwarded-For", ip)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("request-access: api call failed: %v", err)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}) // fail silently
		return
	}
	defer resp.Body.Close()

	// Forward the API response status and body.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	buf.Reset()
	buf.ReadFrom(resp.Body) //nolint:errcheck
	w.Write(buf.Bytes())    //nolint:errcheck
}

// handleRevoke — POST /api/gate/revoke. Clears the session cookie.
func handleRevoke(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// serveGate renders the standalone gate HTML page, optionally with an error message.
func serveGate(w http.ResponseWriter, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	html := bytes.ReplaceAll(gateHTML, []byte("{{ERROR}}"), []byte(errMsg))
	w.Write(html) //nolint:errcheck
}

// serveStatic serves files from staticDir with SPA fallback to index.html.
func serveStatic(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	fpath := filepath.Join(staticDir, filepath.Clean(upath))

	info, err := os.Stat(fpath)
	if os.IsNotExist(err) || (err == nil && info.IsDir()) {
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		return
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, fpath)
}

// isAdmitted checks the HMAC-signed session cookie.
func isAdmitted(r *http.Request) bool {
	if gateSecret == "" {
		return false
	}
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	return verifyToken(gateSecret, c.Value)
}

// setSessionCookie writes an HttpOnly Lax cookie with a 30-day HMAC token.
func setSessionCookie(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    signToken(gateSecret),
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode, // Lax so cookie is sent on top-level external navigation
	})
}

// signToken creates HMAC-SHA256 signed token: "<unix_ts>.<base64url_sig>"
func signToken(secret string) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	sig := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	return ts + "." + sig
}

// verifyToken validates the HMAC signature. Does not enforce expiry (MaxAge handles that).
func verifyToken(secret, token string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0]))
	expected := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(parts[1]))
}

// callValidateAPI calls the AEGIS API to validate an invite code.
func callValidateAPI(ctx context.Context, code, ip string) (bool, error) {
	body, _ := json.Marshal(map[string]string{"code": code})
	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/api/invite/validate", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	if ip != "" {
		req.Header.Set("X-Forwarded-For", ip)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Valid, nil
}

// clientIP extracts the real client IP from proxy headers.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[:i]
	}
	return addr
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
