package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"html/template"
	"io"
	"math"
	"net/url"

	"strings"

	"github.com/revel/revel"
)

// allowMethods are HTTP methods that do NOT require a token
var allowedMethods = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"OPTIONS": true,
	"TRACE":   true,
}

func RandomString(length int) (string, error) {
	buffer := make([]byte, int(math.Ceil(float64(length)/2)))
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return "", err
	}
	str := hex.EncodeToString(buffer)
	return str[:length], nil
}

func RefreshToken(c *revel.Controller) string {
	token, err := RandomString(64)
	if err != nil {
		panic(err)
	}
	c.Session["csrf_token"] = token
	return token
}

// CsrfFilter enables CSRF request token creation and verification.
//
// Usage:
//  1) Add `csrf.CsrfFilter` to the app's filters (it must come after the revel.SessionFilter).
//  2) Add CSRF fields to a form with the template tag `{{ csrftoken . }}`.
// The filter adds a function closure to the `ViewArgs` that can pull out the secret and make the token as-needed,
// caching the value in the request. Ajax support provided through the `X-CSRFToken` header.
func CsrfFilter(c *revel.Controller, fc []revel.Filter) {
	t, foundToken := c.Session["csrf_token"]
	var token string

	if !foundToken {
		token = RefreshToken(c)
	} else {
		token = t.(string)
	}

	referer, refErr := url.Parse(c.Request.Referer())
	if refErr != nil {
		c.Result = c.Forbidden("REVEL CSRF: Unable to fetch referer")
		return
	}

	requestUrl := getFullRequestURL(c)
	isSameOrigin := sameOrigin(requestUrl, referer)
	// If the Request method isn't in the white listed methods
	if !allowedMethods[c.Request.Method] && !IsExempt(c) {
		validToken := validToken(token, isSameOrigin, foundToken, c)
		c.Log.Info("Validating route for token", "token", token, "wasfound", foundToken, "isvalid", validToken)
		if !validToken {
			c.Log.Warn("Invalid CSRF token", "token", token, "wasfound", foundToken)
			return
		}
	}

	fc[0](c, fc[1:])

	// Only add token to ViewArgs if the request is: not AJAX, not missing referer header, and (is same origin, or is an empty referer).
	if c.Request.GetHttpHeader("X-CSRFToken") == "" && (referer.String() == "" || isSameOrigin) {
		c.ViewArgs["_csrftoken"] = token
	}
}

// If this call should be checked validate token
func validToken(token string, isSameOrigin, foundToken bool, c *revel.Controller) (result bool) {
	// Token wasn't present at all
	if !foundToken {
		c.Result = c.Forbidden("REVEL CSRF: Session token missing.")
		return
	}

	// Same origin
	if !isSameOrigin {
		c.Result = c.Forbidden("REVEL CSRF: Same origin mismatch.")
		return
	}

	var requestToken string
	// First check for token in post data
	if c.Request.Method == "POST" {
		requestToken = c.Params.Get("csrftoken")
	}

	// Then check for token in custom headers, as with AJAX
	if requestToken == "" {
		requestToken = c.Request.GetHttpHeader("X-CSRFToken")
	}

	if requestToken == "" || !compareToken(requestToken, token) {
		c.Result = c.Forbidden("REVEL CSRF: Invalid token.")
		return
	}

	return true
}

// Helper function to fix the full URL in the request
func getFullRequestURL(c *revel.Controller) (requestUrl *url.URL) {
	requestUrl = c.Request.URL

	c.Log.Debug("Using ", "request url host", requestUrl.Host, "request host", c.Request.Host, "cookie domain", revel.CookieDomain)
	// Update any of the information based on the headers
	if host := c.Request.GetHttpHeader("X-Forwarded-Host"); host != "" {
		requestUrl.Host = strings.ToLower(host)
	}
	if scheme := c.Request.GetHttpHeader("X-Forwarded-Proto"); scheme != "" {
		requestUrl.Scheme = strings.ToLower(scheme)
	}
	if scheme := c.Request.GetHttpHeader("X-Forwarded-Scheme"); scheme != "" {
		requestUrl.Scheme = strings.ToLower(scheme)
	}

	// Use the revel.CookieDomain for the hostname, or the c.Request.Host
	if requestUrl.Host == "" {
		host := revel.CookieDomain
		if host == "" && c.Request.Host != "" {
			host = c.Request.Host
			// Slice off any port information.
			if i := strings.Index(host, ":"); i != -1 {
				host = host[:i]
			}
		}
		requestUrl.Host = host
	}

	// If no scheme found in headers use the revel server settings
	if requestUrl.Scheme == "" {
		// Fix the Request.URL, it is missing information, go http server does this
		if revel.HTTPSsl {
			requestUrl.Scheme = "https"
		} else {
			requestUrl.Scheme = "http"
		}
		fixedUrl := requestUrl.Scheme + "://" + c.Request.Host + c.Request.URL.Path
		if purl, err := url.Parse(fixedUrl); err == nil {
			requestUrl = purl
		}
	}

	c.Log.Debug("getFullRequestURL ", "requesturl", requestUrl.String())
	return
}

// Compare the two tokens
func compareToken(requestToken, token string) bool {
	// ConstantTimeCompare will panic if the []byte aren't the same length
	if len(requestToken) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(requestToken), []byte(token)) == 1
}

// Validates same origin policy
func sameOrigin(u1, u2 *url.URL) bool {
	return u1.Scheme == u2.Scheme && u1.Hostname() == u2.Hostname()
}

// Add a function to the template functions map
func init() {
	revel.TemplateFuncs["csrftoken"] = func(viewArgs map[string]interface{}) template.HTML {
		if tokenFunc, ok := viewArgs["_csrftoken"]; !ok {
			panic("REVEL CSRF: _csrftoken missing from ViewArgs.")
		} else {
			return template.HTML(tokenFunc.(string))
		}
	}
}
