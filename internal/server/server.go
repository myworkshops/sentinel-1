package server

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myworkshops/sentinel-1/internal/config"
	"github.com/myworkshops/sentinel-1/internal/keycloak"
)

// New initializes the HTTP server with routes, middleware, and HTML templates.
func New(cfg *config.Config, kc keycloak.Client, templates *template.Template) *gin.Engine {
	// gin.Default() includes Logger and Recovery middleware
	r := gin.Default()

	r.SetHTMLTemplate(templates)

	// Global security headers
	r.Use(SecureHeaders())

	// Serve login page with state generation middleware for CSRF protection
	r.GET("/", GenerateState(), func(c *gin.Context) {
		state := c.GetString("state")
		authURL := kc.AuthCodeURL(state)
		c.HTML(http.StatusOK, "login.html", gin.H{
			"AuthURL": authURL,
		})
	})

	// OAuth2 callback endpoint with state verification middleware
	r.GET("/callback", VerifyState(), func(c *gin.Context) {
		code := c.Query("code")
		token, err := kc.Exchange(c.Request.Context(), code)
		if err != nil {
			log.Printf("Token exchange error: %v", err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		c.HTML(http.StatusOK, "token.html", gin.H{
			"Token": token,
		})
	})

	return r
}
