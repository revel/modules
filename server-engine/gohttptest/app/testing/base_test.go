
package test

import (
    "testing"
    "flag"
    "os"
    "github.com/revel/revel"
    "github.com/revel/examples/booking/app/startup"
    "github.com/revel/modules/server-engine/gohttptest/app"
    "sync"
    "net/http"
    "fmt"
)
var importPath *string = flag.String("revel.importPath", "", "Go Import Path for the app.")
func TestMain(m *testing.M) {
//	// call flag.Parse() here if TestMain uses flags
    flag.Parse()
    locker := sync.Mutex{}
    revel.InitEventList = append(revel.InitEventList,func(event int) (returnType int) {
        if event==revel.REVEL_BEFORE_LOAD_MODULE {
            println("Adding module")
            revel.Config.SetOption("server.engine", "go-test")
            revel.Config.SetOption("module.go-test","github.com/revel/modules/server-engine/gohttptest")

        } else if event==revel.ENGINE_EVENT_STARTUP{
            go func() {
                // Wait for the server to send back a start response
                <-revel.CurrentEngine.(*app.GOHttpServer).StartedChan
                locker.Unlock()
            }()
        }
        return 0
    })
    locker.Lock()
    revel.Init("",*importPath,"")
    go startup.MainRun()
    locker.Lock()
    os.Exit(m.Run())

}

func TestIndex(t *testing.T) {
    request := revel.CurrentEngine.(*gohttptest.GOHttpServer).NewRequest(t, "/")
    revel.CurrentEngine.(*gohttptest.GOHttpServer).TestChannel<-request
    <-request.ResponseChannel
    if status := request.Response.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)
    }
    fmt.Printf("Data %#v\n", request.Response.Header())
}
func bTestLogin(t *testing.T) {
    request := revel.CurrentEngine.(*gohttptest.GOHttpServer).NewRequest(t, "/logout")
    revel.CurrentEngine.(*gohttptest.GOHttpServer).TestChannel<-request
    <-request.ResponseChannel
    if status := request.Response.Code; status != http.StatusFound {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)
    }
    //body := request.Response.Body.String()
    fmt.Printf("Data %#v\n", request.Response.Header())
    //println("Response",body)

}

