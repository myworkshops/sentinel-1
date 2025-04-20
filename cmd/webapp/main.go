package main

import (
	"context"
	"html/template"
	"log"
	"time"

	"github.com/myworkshops/sentinel-1/internal/config"
	"github.com/myworkshops/sentinel-1/internal/keycloak"
	"github.com/myworkshops/sentinel-1/internal/server"
	"github.com/myworkshops/sentinel-1/pkg/watcher"
)

func main() {
	// 1. Load initial configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// 2. Start secrets watcher to reload client credentials on change
	go watcher.Watch(cfg.SecretFilePath, func(newID, newSecret string) {
		cfg.Reload(newID, newSecret)
		log.Printf("Keycloak credentials reloaded: client_id=%s", newID)
	})

	// 3. Initialize OIDC client using the current credentials
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientID, clientSecret := cfg.GetClientCreds()

	kcClient, err := keycloak.NewClient(ctx,
		cfg.KeycloakURL,
		cfg.RedirectURL,
		clientID,
		clientSecret,
	)
	if err != nil {
		log.Fatalf("failed to create Keycloak client: %v", err)
	}

	// 4. Parse HTML templates
	templates := template.Must(template.ParseGlob("internal/web/templates/*.html"))

	// 5. Create and run the HTTP server
	srv := server.New(cfg, kcClient, templates)
	if err := srv.Run(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
