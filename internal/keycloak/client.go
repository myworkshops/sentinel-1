package keycloak

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// Public interface
type Client interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
}

// Internal implementation (without custom httpClient)
type kcClient struct {
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// NewClient using Go's default http.Client settings
func NewClient(
	ctx context.Context, // This context is not used for http client injection anymore
	issuerURL,
	redirectURL string,
	clientID, clientSecret string,
) (Client, error) {

	// Initial JWKS check using default http client
	jwksURL := issuerURL + "/protocol/openid-connect/certs"
	log.Printf("🔍 JWKS: %s", jwksURL)
	resp, err := http.Get(jwksURL) // Use standard http.Get
	if err != nil {
		return nil, fmt.Errorf("initial JWKS fetch failed: %w", err)
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("initial JWKS fetch failed: status=%d", resp.StatusCode)
	}
	log.Printf("✅ JWKS status=%d", resp.StatusCode)

	// Verifier with RemoteKeySet (will use http.DefaultClient internally)
	// Use context.Background() for the keyset to avoid premature cancellation
	keySet := oidc.NewRemoteKeySet(context.Background(), jwksURL)
	verifier := oidc.NewVerifier(issuerURL, keySet, &oidc.Config{ClientID: clientID})

	// Static OAuth2 endpoints
	oauth2Cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  issuerURL + "/protocol/openid-connect/auth",
			TokenURL: issuerURL + "/protocol/openid-connect/token",
		},
	}

	return &kcClient{oauth2Config: oauth2Cfg, verifier: verifier}, nil
}

// kcClient methods (satisfy the Client interface)
func (c *kcClient) AuthCodeURL(state string) string {
	return c.oauth2Config.AuthCodeURL(state)
}

func (c *kcClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	// Exchange call will use http.DefaultClient or client from context if oauth2 supports it
	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token field in OAuth2 token")
	}

	// Use a separate context with timeout for the verification step itself
	// Note: The HTTP request inside Verify will use http.DefaultClient
	verifyCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // 20s timeout
	defer cancel()

	if _, err := c.verifier.Verify(verifyCtx, rawID); err != nil {
		log.Printf("ID Token verification error detail: %v", err)
		return nil, fmt.Errorf("ID Token verification failed: %w", err)
	}

	return token, nil
}
