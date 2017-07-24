package revelnewrelic

import (
	"github.com/newrelic/go-agent"
	"github.com/revel/revel"
)

type ServerNewRelic struct {
	NewRelicConfig *newrelic.Config
	NewRelicApp    newrelic.Application
	revel.GoHttpServer
}

func init() {
	revel.RegisterServerEngine("newrelic", func() revel.ServerEngine {
		nr := &ServerNewRelic{
			GoHttpServer: revel.GoHttpServer{},
		}
		config := newrelic.NewConfig("Unknown Application", " Unknown Key ")

		nr.NewRelicConfig = &config
		return nr
	})

}
func (g *ServerNewRelic) Init(init *revel.EngineInit) {
	g.GoHttpServer.Init(init)
}

func (nr *ServerNewRelic) Event(event int, args interface{}) {

	switch event {
	case revel.ENGINE_BEFORE_INITIALIZED:
		nr.NewRelicConfig.AppName = revel.Config.StringDefault("app.name", "Uknown App")

		license := revel.Config.StringDefault("server.newrelic.license", "")
		if license != "" {
			nr.NewRelicConfig.License = license
			revel.TRACE.Println("Assigned NewRelic license")
		} else {
			revel.ERROR.Println("Newrelic license key not assigned, configuraiton missing 'server.newrelic.license'")
		}
		addfilter := revel.Config.BoolDefault("server.newrelic.addfilter", true)
		if addfilter {
			// Inject filter after the router (Normally position 2)
			revel.Filters = append(revel.Filters, NewRelicFilter)
			copy(revel.Filters[3:], revel.Filters[2:])
			revel.Filters[2] = NewRelicFilter
			revel.TRACE.Println("Newrelic filter injected")
		}
	case revel.ENGINE_STARTED:
		// Check to see if configuration is set
		// create the application interface
		app, err := newrelic.NewApplication(*nr.NewRelicConfig)
		if err != nil {
			revel.ERROR.Panic("Failed to start NewRelic:", err)
		}
		nr.NewRelicApp = app

	}

	nr.GoHttpServer.Event(event, args)
}
func (nr *ServerNewRelic) Name() string {
	return "newrelic"
}
func (nr *ServerNewRelic) Engine() interface{} {
	return nr
}

// This is a simplistic example of setting up a filter to record all events for the
// webserver as transactions
func NewRelicFilter(c *revel.Controller, fc []revel.Filter) {
	if nr, ok := revel.CurrentEngine.Engine().(*ServerNewRelic); ok {
		if nr.NewRelicApp != nil {
			txn := nr.NewRelicApp.StartTransaction(c.Action,
				c.Response.Out.Server.(*revel.GoResponse).Original,
				c.Request.In.(*revel.GoRequest).Original)
			defer txn.End()
		} else {
			revel.ERROR.Println("Newrelic application not initialized before filter called")
		}
	}

	fc[0](c, fc[1:])
}
