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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if cfg.SecretFilePath != "" {
		go watcher.Watch(cfg.SecretFilePath, func(newID, newSecret string) {
			cfg.Reload(newID, newSecret)
		})
	} else {
		log.Println("WARN: SECRET_WATCH_PATH not set, secret watcher disabled.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientID, clientSecret := cfg.GetClientCreds()
	if clientID == "" || clientSecret == "" {
		log.Println("WARN: Starting keycloak client without initial credentials. Waiting for watcher...")
	}

	kcClient, err := keycloak.NewClient(ctx,
		cfg.GetIssuerURL(),
		cfg.GetInternalURL(),
		cfg.GetRedirectURL(),
		clientID,
		clientSecret,
	)
	if err != nil {
		log.Fatalf("failed to create Keycloak client: %v", err)
	}

	templates := template.Must(template.ParseGlob("internal/web/templates/*.html"))

	srv := server.New(cfg, kcClient, templates)
	srv.Static("/static", "./static")
	log.Println("Starting server on :8080")
	if err := srv.Run(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
