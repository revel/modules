package controllers
import (
    "github.com/revel/revel"
    "github.com/newrelic/go-agent"
    "github.com/revel/modules/server-engine/newrelic"
)
type RelicController struct {
	*revel.Controller
}
func (r *RelicController) GetRelicApplication() newrelic.Application  {
    if app,ok:=revel.CurrentEngine.(*revelnewrelic.ServerNewRelic);ok {
        return app.NewRelicApp
    }
    return nil
}

