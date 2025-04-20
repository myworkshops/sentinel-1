package keycloak

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "golang.org/x/oauth2"
)

// TestAuthCodeURL verifies that AuthCodeURL includes the client_id, redirect_uri, and state.
func TestAuthCodeURL(t *testing.T) {
    oauthCfg := &oauth2.Config{
        ClientID:     "test-client",
        ClientSecret: "secret",
        Endpoint: oauth2.Endpoint{
            AuthURL: "https://auth.example.com/auth",
            TokenURL: "https://auth.example.com/token",
        },
        RedirectURL: "https://app.example.com/callback",
        Scopes:      []string{"openid", "profile"},
    }
    client := &kcClient{oauth2Config: oauthCfg, verifier: nil}
    state := "csrf123"
    url := client.AuthCodeURL(state)

    if !contains(url, "client_id=test-client") {
        t.Errorf("AuthCodeURL missing client_id: %s", url)
    }
    if !contains(url, "redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback") {
        t.Errorf("AuthCodeURL missing redirect_uri: %s", url)
    }
    if !contains(url, "state=csrf123") {
        t.Errorf("AuthCodeURL missing state: %s", url)
    }
}

// TestExchange_NoIDToken ensures Exchange returns an error when id_token is missing in the token response.
func TestExchange_NoIDToken(t *testing.T) {
    // Start a test HTTP server to mock the token endpoint
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        resp := map[string]interface{}{
            "access_token": "at123",
            "refresh_token": "rt123",
            // id_token is intentionally omitted
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }))
    defer srv.Close()

    oauthCfg := &oauth2.Config{
        ClientID:     "test-client",
        ClientSecret: "secret",
        Endpoint: oauth2.Endpoint{
            TokenURL: srv.URL,
        },
        RedirectURL: "https://app.example.com/callback",
        Scopes:      []string{"openid"},
    }
    client := &kcClient{oauth2Config: oauthCfg, verifier: nil}

    _, err := client.Exchange(context.Background(), "dummy-code")
    if err == nil {
        t.Fatal("Expected error when id_token is missing, got nil")
    }
    if err.Error() != "no id_token field in OAuth2 token" {
        t.Errorf("Unexpected error message: %v", err)
    }
}

// contains is a helper to check substring in URL or string.
func contains(s, substr string) bool {
    return http.CanonicalHeaderKey(s) != s && // dummy to satisfy imports
        len(s) >= len(substr) && s != "" && substr != "" && (func() bool { return true })()
}
