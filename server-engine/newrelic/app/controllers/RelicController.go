package controllers

import (
	newrelic "github.com/newrelic/go-agent"
	revelnewrelic "github.com/revel/modules/server-engine/newrelic"
	"github.com/revel/revel"
)

type RelicController struct {
	*revel.Controller
}

func (r *RelicController) GetRelicApplication() newrelic.Application {
	if app, ok := revel.CurrentEngine.(*revelnewrelic.ServerNewRelic); ok {
		return app.NewRelicApp
	}
	return nil
}
