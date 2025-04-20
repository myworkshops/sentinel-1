package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	KeycloakURL    string // e.g. https://auth.example.com/auth/realms/myrealm
	ClientID       string // will ser recargado por el watcher
	ClientSecret   string // idem
	RedirectURL    string // e.g. https://myapp.example.com/callback
	SecretFilePath string // e.g. /etc/secrets/keycloak

	mu sync.RWMutex // protege ClientID y ClientSecret
}

// Load reads initial config, falling back to files if env vars are missing.
func Load() (*Config, error) {
	c := &Config{
		KeycloakURL:    os.Getenv("KEYCLOAK_URL"),
		ClientID:       os.Getenv("KEYCLOAK_CLIENT_ID"),
		ClientSecret:   os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		RedirectURL:    os.Getenv("REDIRECT_URL"),
		SecretFilePath: os.Getenv("SECRET_WATCH_PATH"),
	}
	if c.KeycloakURL == "" || c.RedirectURL == "" {
		return nil, fmt.Errorf("missing KEYCLOAK_URL or REDIRECT_URL")
	}
	// If creds empty, load from files once
	if c.ClientID == "" || c.ClientSecret == "" {
		id, secret, err := readSecretsFromDir(c.SecretFilePath)
		if err != nil {
			return nil, fmt.Errorf("cannot load initial creds: %w", err)
		}
		c.ClientID = id
		c.ClientSecret = secret
	}
	return c, nil
}

func readSecretsFromDir(dir string) (string, string, error) {
	idB, err := ioutil.ReadFile(filepath.Join(dir, "client_id"))
	if err != nil {
		return "", "", err
	}
	secB, err := ioutil.ReadFile(filepath.Join(dir, "client_secret"))
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(string(idB)), strings.TrimSpace(string(secB)), nil
}

// Reload actualiza dinámicamente las credenciales del cliente
func (c *Config) Reload(newID, newSecret string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ClientID = newID
	c.ClientSecret = newSecret
}

// GetClientCreds devuelve un par ClientID/ClientSecret de forma segura
func (c *Config) GetClientCreds() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ClientID, c.ClientSecret
}
