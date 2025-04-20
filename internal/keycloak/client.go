package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// Client defines the interface for interacting with Keycloak via OIDC.
type Client interface {
	// AuthCodeURL returns the URL to redirect users for login.
	AuthCodeURL(state string) string
	// Exchange exchanges an authorization code for an OAuth2 token.
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
}

// kcClient is the concrete implementation of the Client interface.
type kcClient struct {
	oauth2Config *oauth2.Config        // OAuth2 configuration
	verifier     *oidc.IDTokenVerifier // ID Token verifier
}

// NewClient initializes a Keycloak OIDC client with a custom HTTP client
// that forces IPv4 and has generous timeouts, and uses a RemoteKeySet for token verification.
func NewClient(
	ctx context.Context,
	issuerURL string,
	redirectURL string,
	clientID string,
	clientSecret string,
) (Client, error) {
	// 1) Creamos un transport que fuerza IPv4 y timeouts razonables
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		DualStack: false, // IPv4 only
	}
	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
	}

	// 2) Reemplazamos el transport por defecto para que go-oidc lo use en todas las fases
	http.DefaultTransport = transport

	// 3) Creamos un HTTP client con ese transport y un timeout general
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
	}

	// 4) (Opcional) Probar directamente la URL de JWKS para validar red
	jwksURL := issuerURL + "/protocol/openid-connect/certs"
	log.Printf("🔍 testing JWKS URI: %s", jwksURL)
	if resp, err := httpClient.Get(jwksURL); err != nil {
		return nil, fmt.Errorf("initial JWKS fetch failed: %w", err)
	} else {
		resp.Body.Close()
		log.Printf("✅ initial JWKS fetch OK, status=%d", resp.StatusCode)
	}

	// 5) Inyectamos nuestro HTTP client en el contexto para discovery
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	// 6) OIDC discovery (usará nuestro HTTP client)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to perform OIDC discovery: %w", err)
	}

	// 7) Construimos el verifier que internamente usará http.DefaultTransport
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	// 8) Configuramos OAuth2 para el flujo autorización
	oauth2Cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &kcClient{
		oauth2Config: oauth2Cfg,
		verifier:     verifier,
	}, nil
}

// AuthCodeURL returns the OAuth2 authorization URL for the given state.
func (c *kcClient) AuthCodeURL(state string) string {
	return c.oauth2Config.AuthCodeURL(state)
}

// Exchange exchanges the authorization code for tokens and verifies the ID Token.
func (c *kcClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	// Exchange code for token
	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Extract the raw ID Token for verification
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token field in OAuth2 token")
	}

	// Verify the ID Token signature and claims
	if _, err := c.verifier.Verify(ctx, rawIDToken); err != nil {
		return nil, fmt.Errorf("ID Token verification failed: %w", err)
	}

	return token, nil
}
