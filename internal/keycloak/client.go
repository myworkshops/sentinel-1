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

type Client interface {
	AuthCodeURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
}

type kcClient struct {
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

func NewClient(
	ctx context.Context,
	publicIssuerURL,
	internalBaseURL,
	redirectURL string,
	clientID, clientSecret string,
) (Client, error) {

	internalJwksURL := internalBaseURL + "/protocol/openid-connect/certs"

	resp, err := http.Get(internalJwksURL)
	if err != nil {
		log.Printf("WARN: Initial internal JWKS fetch failed: %v", err)
	} else {
		if resp.Body != nil {
			resp.Body.Close()
		}
		log.Printf("✅ INFO: Initial internal JWKS status: %d", resp.StatusCode)
	}

	keySet := oidc.NewRemoteKeySet(context.Background(), internalJwksURL)
	verifier := oidc.NewVerifier(publicIssuerURL, keySet, &oidc.Config{ClientID: clientID})

	oauth2Cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  publicIssuerURL + "/protocol/openid-connect/auth",
			TokenURL: internalBaseURL + "/protocol/openid-connect/token",
		},
	}

	return &kcClient{
		oauth2Config: oauth2Cfg,
		verifier:     verifier,
	}, nil
}

func (c *kcClient) AuthCodeURL(state string) string {
	return c.oauth2Config.AuthCodeURL(state)
}

func (c *kcClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token field in OAuth2 token")
	}

	verifyCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if _, err := c.verifier.Verify(verifyCtx, rawID); err != nil {
		log.Printf("ID Token verification error detail: %v", err)
		return nil, fmt.Errorf("ID Token verification failed: %w", err)
	}

	return token, nil
}
