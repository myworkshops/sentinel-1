// internal/server/server.go
package server

import (
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/myworkshops/sentinel-1/internal/config"
	"github.com/myworkshops/sentinel-1/internal/keycloak"
)

// prettyJSON helper
func prettyJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// decodeJWT decodes (without verifying) a JWT payload → map[string]any
func decodeJWT(tok string) map[string]any {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil
	}
	return out
}

// New returns the HTTP server engine.
func New(cfg *config.Config, kc keycloak.Client, tpl *template.Template) *gin.Engine {
	r := gin.Default()
	r.Use(SecureHeaders())
	r.SetHTMLTemplate(tpl)

	r.GET("/", GenerateState(), func(c *gin.Context) {
		state := c.GetString("state")
		c.HTML(http.StatusOK, "login.html", gin.H{
			"AuthURL": kc.AuthCodeURL(state),
		})
	})

	r.GET("/callback", VerifyState(), func(c *gin.Context) {
		code := c.Query("code")
		tok, err := kc.Exchange(c.Request.Context(), code)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		j, _ := json.Marshal(tok) // full OAuth2 token struct
		c.SetCookie("token_json", string(j), 300, "/", "", false, true)
		c.Redirect(http.StatusFound, "/token")
	})

	r.GET("/token", func(c *gin.Context) {
		raw, err := c.Cookie("token_json")
		if err != nil {
			c.String(http.StatusBadRequest, "token not found")
			return
		}

		// parse back to struct to extract AccessToken
		var tok struct {
			AccessToken string `json:"access_token"`
			IDToken     string `json:"id_token"`
			Expiry      any    `json:"expiry"`
		}
		_ = json.Unmarshal([]byte(raw), &tok)

		claims := decodeJWT(tok.AccessToken)

		c.HTML(http.StatusOK, "token.html", gin.H{
			"TokenJSON":   template.JS(raw), // full raw JSON
			"ClaimsJSON":  template.JS(prettyJSON(claims)),
			"AccessToken": tok.AccessToken,
		})
	})

	return r
}
