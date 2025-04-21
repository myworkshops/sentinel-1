package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Config holds the configuration for the Keycloak client and other settings.
type Config struct {
	KeycloakIssuerURL   string
	KeycloakInternalURL string
	ClientID            string
	ClientSecret        string
	RedirectURL         string
	SecretFilePath      string
	mu                  sync.RWMutex
}

// Load reads the configuration from environment variables.
func Load() (*Config, error) {
	c := &Config{
		KeycloakIssuerURL:   os.Getenv("KEYCLOAK_ISSUER_URL"),
		KeycloakInternalURL: os.Getenv("KEYCLOAK_INTERNAL_URL"),
		ClientID:            os.Getenv("KEYCLOAK_CLIENT_ID"),
		ClientSecret:        os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		RedirectURL:         os.Getenv("REDIRECT_URL"),
		SecretFilePath:      os.Getenv("SECRET_WATCH_PATH"),
	}

	if c.KeycloakIssuerURL == "" || c.KeycloakInternalURL == "" || c.RedirectURL == "" {
		return nil, fmt.Errorf("missing KEYCLOAK_ISSUER_URL, KEYCLOAK_INTERNAL_URL or REDIRECT_URL environment variables")
	}

	if (c.ClientID == "" || c.ClientSecret == "") && c.SecretFilePath != "" {
		log.Printf("Client ID/Secret not found in env vars, attempting to load from path: %s", c.SecretFilePath)
		id, secret, err := readSecretsFromDir(c.SecretFilePath)
		if err != nil {
			log.Printf("WARN: Could not load initial creds from %s: %v. Relying on watcher.", c.SecretFilePath, err)
		} else {
			log.Printf("Successfully loaded initial creds from %s", c.SecretFilePath)
			c.ClientID = id
			c.ClientSecret = secret
		}
	}

	if c.ClientID == "" || c.ClientSecret == "" {
		log.Printf("WARN: ClientID or ClientSecret is still empty after checking env and files.")
	}

	return c, nil
}

// readSecretsFromDir reads the client ID and secret from the specified directory.
func readSecretsFromDir(dir string) (string, string, error) {
	if dir == "" {
		return "", "", fmt.Errorf("secret watch path is empty")
	}
	idPath := filepath.Join(dir, "client_id")
	secPath := filepath.Join(dir, "client_secret")

	idB, err := ioutil.ReadFile(idPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read %s: %w", idPath, err)
	}
	secB, err := ioutil.ReadFile(secPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read %s: %w", secPath, err)
	}
	return strings.TrimSpace(string(idB)), strings.TrimSpace(string(secB)), nil
}

// Reload updates the client ID and secret from the specified directory.
func (c *Config) Reload(newID, newSecret string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ClientID = newID
	c.ClientSecret = newSecret
	log.Printf("Keycloak credentials reloaded: client_id=%s", newID)
}

func (c *Config) GetClientCreds() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ClientID, c.ClientSecret
}

func (c *Config) GetIssuerURL() string {
	return c.KeycloakIssuerURL
}

func (c *Config) GetInternalURL() string {
	return c.KeycloakInternalURL
}

func (c *Config) GetRedirectURL() string {
	return c.RedirectURL
}
