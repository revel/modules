package controllers

import (
	"github.com/revel/revel"
	"github.com/revel/modules/jobs/app/jobs"
	"github.com/robfig/cron"
	"strings"
)

type Jobs struct {
	*revel.Controller
}

func (c Jobs) Status() revel.Result {
	remoteAddress := c.Request.RemoteAddr
	if proxiedAddress, isProxied := c.Request.Header["X-Forwarded-For"]; isProxied {
		remoteAddress = proxiedAddress[0]
	}
	if !strings.HasPrefix(remoteAddress, "127.0.0.1") && !strings.HasPrefix(remoteAddress, "::1") {
		return c.Forbidden("%s is not local", remoteAddress)
	}
	entries := jobs.MainCron.Entries()
	return c.Render(entries)
}

func init() {
	revel.TemplateFuncs["castjob"] = func(job cron.Job) *jobs.Job {
		return job.(*jobs.Job)
	}
}
