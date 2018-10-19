package controllers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/revel/cron"
	"github.com/revel/modules/jobs/app/jobs"
	"github.com/revel/revel"
	"net/http"
	"strings"
)

type Jobs struct {
	*revel.Controller
}

func (c *Jobs) Status() revel.Result {
	remoteAddress := c.Request.RemoteAddr
	if revel.Config.BoolDefault("jobs.auth", false) {
		user, found_user := revel.Config.String("jobs.auth.user")
		pass, found_pass := revel.Config.String("jobs.auth.pass")

		// Verify that a username and password are given in the config file
		if !found_pass || !found_user {
			return c.unauthorized()
		}

		// Verify that the Authorization header is received and valid
		auth := strings.Split(c.Request.GetHttpHeader("Authorization"), " ")
		if len(auth) < 2 {
			return c.unauthorized()
		}

		// Decode Authorization header
		decoded, err := base64.StdEncoding.DecodeString(auth[1])
		if err != nil {
			return c.unauthorized()
		}

		// Split Authorization header to user and password
		str := strings.Split(string(decoded), ":")

		// If SHA256 is enabled, hash received password
		is_sha256 := revel.Config.BoolDefault("jobs.auth.sha256", false)
		if is_sha256 {
			hash := sha256.Sum256([]byte(str[1]))
			str[1] = string(hex.EncodeToString(hash[:]))
			pass = strings.ToLower(pass)
		}

		// Compare user and password
		if user != str[0] || pass != str[1] {
			c.Log.Warn("Attempted login to /@jobs with invalid credentials")
			return c.unauthorized()
		}

	} else {
		if revel.Config.BoolDefault("jobs.acceptproxyaddress", false) {
			if proxiedAddress := c.Request.GetHttpHeader("X-Forwarded-For"); proxiedAddress != "" {
				remoteAddress = proxiedAddress
			}
		}
		if !strings.HasPrefix(remoteAddress, "127.0.0.1") &&
			!strings.HasPrefix(remoteAddress, "::1") &&
			!strings.HasPrefix(remoteAddress, "[::1]") {
			return c.Forbidden("%s is not local", remoteAddress)
		}
	}

	entries := jobs.MainCron.Entries()
	return c.Render(entries)
}

func (c *Jobs) unauthorized() revel.Result {
	c.Response.Status = http.StatusUnauthorized
	c.Response.Out.Header().Set("WWW-Authenticate", "Basic realm=\"revel jobs\"")
	return c.RenderError(errors.New("401: Not Authorized"))
}

func init() {
	revel.TemplateFuncs["castjob"] = func(job cron.Job) *jobs.Job {
		return job.(*jobs.Job)
	}
}
