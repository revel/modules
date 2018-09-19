# Channel Http Tester
This is a **alpha** version of the testsuite. Rather then exchanging requests using an open port
the server uses channels to exchange information. This 
requires the application compiled using Revel v0.20 or newer (This version splits the generated
code into a separate file that can be run). Note **websocket not supported** in this version.   



Sample Code file located in `github.com/revel/examples/booking/app/controllers/app_test.go` 
```go
package controllers_test

import (
	"github.com/revel/examples/booking/app/tmp/run"
	"testing"
	"github.com/revel/modules/server-engine/gohttptest/testsuite"
)
func TestMain(m *testing.M) {
	testsuite.RevelTestHelper(m, "dev",run.Run)
}

func TestIndex(t *testing.T) {
	tester := testsuite.NewTestSuite(t)
	tester.Get("/").AssertOk()
}

```

The test consists of two parts, the first is the "helper" to start the Revel server in a new 
goroutine, the second is your test code. 

To run from the command line you need to pass and args argument to 
the `go test` command like the following 
```commandline

go test -coverprofile=coverage.out github.com/revel/examples/booking/app/controllers/  -args -revel.importPath=github.com/revel/examples/booking

``` 

 
 
