package main

import (
	"fmt"
	"net/http"

	"github.com/windzhu0514/shiba/example/hello/hello"
	"github.com/windzhu0514/shiba/shiba"
)

func main() {
	var middlewares []shiba.Middleware
	middlewares = append(middlewares, shiba.MiddlewareRecover(func(
		w http.ResponseWriter, r *http.Request, err interface{},
	) {
		w.Write([]byte(fmt.Sprint(err)))
	}))
	svr := shiba.NewServer(shiba.WithPprof(), shiba.WithCron(), shiba.WithMiddleware(middlewares...))

	svr.RegisterModule(1, &hello.Hello{})
	svr.Start()
}
