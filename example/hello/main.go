package main

import (
	"fmt"
	"net/http"

	"github.com/windzhu0514/shiba/example/hello/hello"
	"github.com/windzhu0514/shiba/shiba"
)

func main() {
	shiba.RegisterModule(1, &hello.Hello{})
	var middlewares []shiba.MiddlewareFunc
	middlewares = append(middlewares, shiba.MiddlewareRecover(func(
		w http.ResponseWriter, r *http.Request, err interface{},
	) {
		w.Write([]byte(fmt.Sprint(err)))
	}))
	shiba.Start(shiba.WithPprof(), shiba.WithCron(), shiba.WithMiddleware(middlewares...))
}
