package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tomasswaier/RabbitVocab/internal/apikey"
	domainapikey "github.com/tomasswaier/RabbitVocab/internal/domain/apikey"
	"github.com/tomasswaier/RabbitVocab/internal/domain/oauth"
	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
)

const (
	accessTokenTTL  = 90 * 24 * time.Hour
	refreshTokenTTL = 180 * 24 * time.Hour
	authCodeTTL     = 60 * time.Second
)

type OAuthHandler struct {
	oauth   oauth.Repository
	apiKeys domainapikey.Repository
	users   user.Repository
	issuer  string // e.g. https://app.yourdomain.com
}

func NewOAuthHandler(oauthRepo oauth.Repository, apiKeys domainapikey.Repository, users user.Repository, issuer string) *OAuthHandler {
	return &OAuthHandler{oauth: oauthRepo, apiKeys: apiKeys, users: users, issuer: issuer}
}

// ---------- Metadata ----------

func (h *OAuthHandler) Metadata(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                h.issuer,
		"authorization_endpoint":                h.issuer + "/oauth/authorize",
		"token_endpoint":                        h.issuer + "/oauth/token",
		"registration_endpoint":                 h.issuer + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	})
}

// ---------- Dynamic Client Registration (RFC 7591) ----------

type registerClientRequest struct {
	RedirectURIs []string `json:"redirect_uris"`
	ClientName   string   `json:"client_name,omitempty"`
}

func (h *OAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.RedirectURIs) == 0 {
		http.Error(w, "redirect_uris is required", http.StatusBadRequest)
		return
	}
	for _, u := range req.RedirectURIs {
		if _, err := url.ParseRequestURI(u); err != nil {
			http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
			return
		}
	}

	clientID, err := apikey.Generate()
	if err != nil {
		http.Error(w, "failed to generate client id", http.StatusInternalServerError)
		return
	}

	var name *string
	if req.ClientName != "" {
		name = &req.ClientName
	}

	client, err := h.oauth.CreateClient(r.Context(), clientID, name, req.RedirectURIs)
	if err != nil {
		http.Error(w, "failed to register client", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ClientID,
		"client_id_issued_at":        client.CreatedAt.Unix(),
		"redirect_uris":              client.RedirectURIs,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"client_name":                req.ClientName,
	})
}

// ---------- Authorize ----------

const loginFormHTML = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Sign in</title>
<style>
body{font-family:sans-serif;background:#0f172a;color:#e2e8f0;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
form{background:#1e293b;padding:2rem;border-radius:0.75rem;width:280px}
input{width:100%%;padding:0.5rem;margin:0.4rem 0 1rem;border-radius:0.4rem;border:1px solid #334155;background:#0f172a;color:#e2e8f0;box-sizing:border-box}
button{width:100%%;padding:0.6rem;border:none;border-radius:0.4rem;background:#4f46e5;color:white;font-weight:600;cursor:pointer}
h2{margin-top:0}
</style></head>
<body>
<form method="POST" action="/oauth/authorize">
<h2>Sign in to RabbitVocab</h2>
<label>Username</label><input name="username" required autofocus>
<label>Password</label><input name="password" type="password" required>
<input type="hidden" name="client_id" value="%s">
<input type="hidden" name="redirect_uri" value="%s">
<input type="hidden" name="state" value="%s">
<input type="hidden" name="code_challenge" value="%s">
<input type="hidden" name="code_challenge_method" value="%s">
<button type="submit">Sign in</button>
</form>
</body></html>`

func (h *OAuthHandler) AuthorizeForm(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")
	responseType := q.Get("response_type")

	client, err := h.oauth.GetClient(r.Context(), clientID)
	if err != nil {
		http.Error(w, "unknown client", http.StatusBadRequest)
		return
	}
	if !contains(client.RedirectURIs, redirectURI) {
		http.Error(w, "redirect_uri mismatch", http.StatusBadRequest)
		return
	}
	if responseType != "code" {
		redirectWithError(w, r, redirectURI, state, "unsupported_response_type")
		return
	}
	if codeChallenge == "" || codeChallengeMethod != "S256" {
		redirectWithError(w, r, redirectURI, state, "invalid_request")
		return
	}

	fmt.Fprintf(w, loginFormHTML,
		html.EscapeString(clientID),
		html.EscapeString(redirectURI),
		html.EscapeString(state),
		html.EscapeString(codeChallenge),
		html.EscapeString(codeChallengeMethod),
	)
}

func (h *OAuthHandler) AuthorizeSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")
	codeChallenge := r.FormValue("code_challenge")
	codeChallengeMethod := r.FormValue("code_challenge_method")

	client, err := h.oauth.GetClient(r.Context(), clientID)
	if err != nil || !contains(client.RedirectURIs, redirectURI) {
		http.Error(w, "invalid client or redirect_uri", http.StatusBadRequest)
		return
	}

	u, err := h.users.GetByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	rawCode, err := apikey.Generate()
	if err != nil {
		http.Error(w, "failed to generate code", http.StatusInternalServerError)
		return
	}

	if _, err := h.oauth.CreateAuthorizationCode(r.Context(), apikey.Hash(rawCode), clientID, u.ID, redirectURI, codeChallenge, codeChallengeMethod, time.Now().Add(authCodeTTL)); err != nil {
		http.Error(w, "failed to create authorization code", http.StatusInternalServerError)
		return
	}

	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	qs := redirectURL.Query()
	qs.Set("code", rawCode)
	qs.Set("state", state)
	redirectURL.RawQuery = qs.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// ---------- Token ----------

func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	switch r.FormValue("grant_type") {
	case "authorization_code":
		h.handleAuthCodeGrant(w, r)
	case "refresh_token":
		h.handleRefreshGrant(w, r)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
	}
}

func (h *OAuthHandler) handleAuthCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	verifier := r.FormValue("code_verifier")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")

	ac, err := h.oauth.ConsumeAuthorizationCode(r.Context(), apikey.Hash(code))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}
	if ac.ClientID != clientID || ac.RedirectURI != redirectURI {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}
	if !verifyPKCE(verifier, ac.CodeChallenge) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}

	h.issueTokenPair(w, r, ac.UserID, ac.ClientID)
}

func (h *OAuthHandler) handleRefreshGrant(w http.ResponseWriter, r *http.Request) {
	rawRefresh := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")

	rt, err := h.oauth.GetRefreshToken(r.Context(), apikey.Hash(rawRefresh))
	if err != nil || rt.ClientID != clientID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}

	_ = h.oauth.DeleteRefreshToken(r.Context(), apikey.Hash(rawRefresh))
	_, _ = h.apiKeys.Delete(r.Context(), rt.APIKeyID, rt.UserID)

	h.issueTokenPair(w, r, rt.UserID, rt.ClientID)
}

func (h *OAuthHandler) issueTokenPair(w http.ResponseWriter, r *http.Request, userID int64, clientID string) {
	rawAccess, err := apikey.Generate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}
	accessExpiresAt := time.Now().Add(accessTokenTTL)
	label := "oauth"

	ak, err := h.apiKeys.Create(r.Context(), userID, apikey.Hash(rawAccess), &label, &clientID, &accessExpiresAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	rawRefresh, err := apikey.Generate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	if _, err := h.oauth.CreateRefreshToken(r.Context(), apikey.Hash(rawRefresh), ak.ID, userID, clientID, time.Now().Add(refreshTokenTTL)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  rawAccess,
		"token_type":    "Bearer",
		"expires_in":    int(accessTokenTTL.Seconds()),
		"refresh_token": rawRefresh,
	})
}

// ---------- helpers ----------

func verifyPKCE(verifier, challenge string) bool {
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, state, errCode string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, errCode, http.StatusBadRequest)
		return
	}
	q := u.Query()
	q.Set("error", errCode)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}
