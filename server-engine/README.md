# Server Engines
These are some of the server engines built by the Revel team.
In order to use an engine you must import the module and set
the configuration option `server.engine` equal to the engine you want.
You may define your own engine by implementing `revel.ServerEngine` and
registering it with revel in an `init` 
function `revel.RegisterServerEngine(YOUR_ENGINE_INSTANCE)`. See
`revel.GOHttpServer` for an example


###App.conf
- **server.engine** Defaults to `go`. Specify the engine you wish to use 
 
 
