package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/revel/modules/jwtauth/app/jwt"
	"github.com/revel/modules/jwtauth/app/models"

	"github.com/revel/revel"
	"github.com/revel/revel/cache"
)

type JwtAuth struct {
	*revel.Controller
}

func (c *JwtAuth) Token() revel.Result {
	user, err := c.parseUserInfo()
	if err != nil {
		revel.ERROR.Printf("Unable to read user info %q", err)
		c.Response.Status = http.StatusBadRequest
		return c.RenderJson(map[string]string{
			"id":      "bad_request",
			"message": "Unable to read user info",
		})
	}

	if subject, pass := jwt.Authenticate(user.Username, user.Password); pass {
		token, err := jwt.GenerateToken(subject)
		if err != nil {
			c.Response.Status = http.StatusInternalServerError
			return c.RenderJson(map[string]string{
				"id":      "server_error",
				"message": "Unable to generate token",
			})
		}

		return c.RenderJson(map[string]string{
			"token": token,
		})
	}

	c.Response.Status = http.StatusUnauthorized
	c.Response.Out.Header().Set("Www-Authenticate", jwt.Realm)

	return c.RenderJson(map[string]string{
		"id":      "unauthorized",
		"message": "Invalid credentials",
	})
}

func (c *JwtAuth) RefreshToken() revel.Result {
	claims := c.Args[jwt.TokenClaimsKey].(map[string]interface{})
	revel.INFO.Printf("Claims: %q", claims)

	tokenString, err := jwt.GenerateToken(claims[jwt.SubjectKey].(string))
	if err != nil {
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJson(map[string]string{
			"id":      "server_error",
			"message": "Unable to generate token",
		})
	}

	// Issued new token and adding existing token into blocklist for remaining validitity period
	// Let's say if existing token is valid for another 10 minutes, then it reside 10 mintues
	// in the blocklist
	go addToBlocklist(c.Request, claims)

	return c.RenderJson(map[string]string{
		"token": tokenString,
	})
}

func (c *JwtAuth) ValidateToken() revel.Result {
	// When request reaches here, then it has valid auth token
	// else request would have received 401 - Unauthorized response
	return c.RenderJson(map[string]string{
		"id":      "success",
		"message": "Auth token is valid",
	})
}

func (c *JwtAuth) Logout() revel.Result {
	// Auth token will be added to blocklist for remaining token validitity period
	// Let's token is valid for another 10 minutes, then it reside 10 mintues in the blocklist
	go addToBlocklist(c.Request, c.Args[jwt.TokenClaimsKey].(map[string]interface{}))

	return c.RenderJson(map[string]string{
		"id":      "success",
		"message": "Successfully logged out",
	})
}

// Private methods
func (c *JwtAuth) parseUserInfo() (*models.JwtUser, error) {
	rUser := &models.JwtUser{}
	decoder := json.NewDecoder(c.Request.Body)
	err := decoder.Decode(rUser)
	return rUser, err
}

func addToBlocklist(r *revel.Request, claims map[string]interface{}) {
	tokenString := jwt.GetAuthToken(r)
	expriyAt := time.Minute * time.Duration(jwt.TokenRemainingValidity(claims[jwt.ExpirationKey]))

	cache.Set(tokenString, tokenString, expriyAt)
}
